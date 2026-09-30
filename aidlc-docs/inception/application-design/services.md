# Services (Orchestration Layer)

**Cập nhật theo ADR-0007**: Orchestration Service (mới) là Saga coordinator duy nhất, giao tiếp với các service nghiệp vụ qua RabbitMQ (command/event, bất đồng bộ). API Gateway chỉ còn routing + khởi tạo Saga qua REST.

**Revision (2026-08-07, ADR-0014)**: TTS Service ban đầu được gọi trực tiếp qua REST đồng bộ bên trong bước "Render Scenes". Theo ADR-0014, TTS Service nay là **bước Saga độc lập** ("Synthesize Speech"), tách khỏi Rendering Service hoàn toàn — Rendering Service không còn gọi TTS. Saga Step Definitions và sequence diagram bên dưới đã cập nhật theo quyết định này.

**Cập nhật (CR-055, 2026-09-30)**: Content Plugin Service (gỡ ở CR-020) và Script Processing Service (gỡ ở CR-040) không còn; thêm Authoring Service (CR-040, ADR-0029) và LLM Service (CR-039). Nội dung dưới đây theo hệ đang chạy.

## Saga: Render Pipeline (Story A1 → D1)

```mermaid
sequenceDiagram
    participant GUI
    participant GW as API Gateway
    participant ORCH as Orchestrator Service
    participant MQ as RabbitMQ
    participant RD as Rendering Service
    participant TTS as TTS Service
    participant VA as Video Assembly

    GUI->>GW: POST /projects/{id}/script
    GW->>ORCH: POST /sagas/render
    ORCH->>MQ: command validate_script (CR-040 — thay parse_script + classify_scenes)
    MQ->>RD: (deliver) validate_script
    RD-->>MQ: event script_validated (scene_class_name, engine, beats)
    MQ-->>ORCH: (deliver) script_validated
    ORCH-->>GW: SSE progress
    GW-->>GUI: SSE progress

    ORCH->>MQ: command synthesize_speech (ADR-0014 — bước Saga độc lập, tách khỏi Rendering)
    MQ->>TTS: (deliver) synthesize_speech
    TTS-->>MQ: event speech_synthesized (audio_path + duration per scene)
    MQ-->>ORCH: (deliver) speech_synthesized
    ORCH-->>GW: SSE progress (status=speech_synthesized)

    ORCH->>MQ: command render_scenes (chỉ animation — audio đã có sẵn)
    MQ->>RD: (deliver) render_scenes
    activate RD
    RD-->>MQ: event scene_rendered (per scene, tiến trình)
    MQ-->>ORCH: (deliver) scene_rendered
    ORCH-->>GW: SSE progress
    RD-->>MQ: event rendering_completed
    deactivate RD
    MQ-->>ORCH: (deliver) rendering_completed

    ORCH->>MQ: command assemble_video
    MQ->>VA: (deliver) assemble_video
    VA-->>MQ: event video_assembled
    MQ-->>ORCH: (deliver) video_assembled
    ORCH-->>GW: SSE done (status=ready_to_publish)
    GW-->>GUI: SSE done
```

## Saga: Publish (Story E1 → E3)

```mermaid
sequenceDiagram
    participant GUI
    participant GW as API Gateway
    participant ORCH as Orchestrator Service
    participant MQ as RabbitMQ
    participant PB as Publisher Service
    participant YT as YouTube API

    GUI->>GW: GET /auth/youtube/start (nếu chưa xác thực)
    GW->>PB: GET /auth/youtube/start (REST trực tiếp, ngoài Saga)
    PB-->>GUI: redirect OAuth
    GUI->>PB: (browser) OAuth callback
    PB-->>GW: authenticated=true

    GUI->>GW: POST /projects/{id}/publish
    GW->>ORCH: POST /sagas/publish
    ORCH->>MQ: command publish_video
    MQ->>PB: (deliver) publish_video
    PB->>YT: upload video + metadata
    YT-->>PB: youtube_video_url
    PB-->>MQ: event video_published
    MQ-->>ORCH: (deliver) video_published
    ORCH-->>GW: SSE done (status=published, youtube_video_url)
    GW-->>GUI: SSE done
```

## Saga Step Definitions

| # | Step | Command | Service | Success Event | Failure Event |
|---|---|---|---|---|---|
| 1 | Validate Script | `validate_script` | Rendering (CR-040) | `script_validated` | `validation_failed` |
| 2 | Synthesize Speech | `synthesize_speech` | TTS (ADR-0014 — bước độc lập, không còn gọi từ Rendering) | `speech_synthesized` | `synthesis_failed` |
| 3 | Render Scenes | `render_scenes` | Rendering (chỉ animation — audio đã có sẵn từ bước 2) | `rendering_completed` | `rendering_failed` |
| 4 | Assemble Video | `assemble_video` | Video Assembly | `video_assembled` | `assembly_failed` |
| — | Generate Clips (theo yêu cầu, CR-007) | `generate_clips` | Video Assembly | `clips_generated` | *(không có; lỗi từng clip nằm trong payload)* |
| 5 | Publish Video (Saga Publish) | `publish_video` | Publisher | `video_published` | `publish_failed` |

## Error Handling & Compensating Actions (Saga Orchestration, ADR-0007)
- Khi Orchestrator nhận event `*_failed`, đặt trạng thái project = `failed_at_<step>`, giữ nguyên kết quả các bước trước (artifact đã tạo trong shared volume không bị xóa).
- **Compensating action** cụ thể theo bước:
  - `validation_failed`: không có artifact cần dọn — Creator sửa script và retry từ bước 1
  - `synthesis_failed`: giữ audio đã sinh thành công (idempotent theo `project_id`+`scene_index`, file trong shared volume), retry chỉ scene lỗi
  - `rendering_failed`: giữ scene đã render thành công (idempotent theo `project_id`+`scene_index`), retry chỉ scene lỗi
  - `assembly_failed`: giữ animation/audio clip, retry chỉ bước Assembly
  - `publish_failed`: không có gì cần rollback ở phía YouTube (chưa tạo gì), retry bước Publish
- GUI hiển thị lỗi cụ thể (Story C6) và nút retry tương ứng bước lỗi — Orchestrator chỉ gửi lại command của bước đó, không chạy lại toàn bộ Saga.
- **Idempotency**: mọi command mang `saga_id` + `project_id`; consumer kiểm tra artifact đã tồn tại trước khi tạo lại (tránh xử lý trùng khi message được deliver lại do requeue).
