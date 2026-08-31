# Business Logic Model — Unit 8: Orchestrator Service

## Overview
Orchestrator điều phối 2 Saga (Render Pipeline 5 bước, Publish 1 bước) bằng cơ chế choreography-driven orchestration: nhận event từ các service nghiệp vụ qua AMQP, cập nhật state machine của `Project`, gộp dữ liệu tích lũy qua các bước, và dispatch command cho bước tiếp theo. Không chứa business logic của từng bước (đó là trách nhiệm Unit 2–7) — chỉ điều phối thứ tự, gộp dữ liệu, theo dõi trạng thái, và xử lý lỗi bằng retry-by-step.

## Saga 1: Render Pipeline (5 bước)

### Bước 1 — Parse Script
- **Trigger**: `POST /v1/sagas/render` (Creator submit script qua GUI, Story C1)
- **Input tích lũy**: `script_content`, `plugin_id`, `voice_language`, `background_music_path?` (Question 3 — optional field trong input Saga)
- **Logic**: `StartRenderSagaUseCase` tạo `Project` mới (status=`draft`), sinh `saga_id` mới, tạo `SagaStep{step: parse_script, status: in_progress}`, publish command `parse_script`, chuyển status → `parsing_script`.
- **Event chờ**: `script_parsed` (chứa `scenes: [{scene_index, narration_text, illustration_hint, code_snippet?, code_language?}]`)

### Bước 2 — Classify Scenes
- **Trigger**: nhận `script_parsed`
- **Logic**: `HandleStepEventUseCase` lưu `scenes` (narration_text, illustration_hint, code_snippet, code_language) vào `Project`, dispatch command `classify_scenes` (payload: `plugin_id` + `scenes`), chuyển status → `classifying_scenes`.
- **Event chờ**: `scenes_classified` (chứa `category`, `animation_template_id` per scene)

### Bước 3 — Synthesize Speech
- **Trigger**: nhận `scenes_classified`
- **Logic**: gộp `category`/`animation_template_id` vào `Project.scenes` theo `scene_index`, dispatch command `synthesize_speech` (payload: `scenes` + `voice_language`), chuyển status → `synthesizing_speech`.
- **Event chờ**: `speech_synthesized` (chứa `audio_path`, `duration_seconds` per scene)

### Bước 4 — Render Scenes (Data Aggregation — Question 1)
- **Trigger**: nhận `speech_synthesized`
- **Logic**: gộp `audio_path`/`duration_seconds` vào `Project.scenes`. Trước khi dispatch, validate: cả 3 nguồn dữ liệu tích lũy (`script_parsed`, `scenes_classified`, `speech_synthesized`) PHẢI cùng tập `scene_index`.
  - Nếu KHỚP: gộp theo `scene_index` làm khóa → payload đầy đủ `render_scenes` (mỗi scene có đủ `narration_text`/`illustration_hint`/`code_snippet`/`code_language`/`category`/`animation_template_id`/`audio_path`/`duration_seconds`), dispatch command, chuyển status → `rendering`.
  - Nếu KHÔNG khớp (số lượng/scene_index khác nhau giữa các nguồn): đây là lỗi hệ thống nghiêm trọng (bug ở 1 trong 3 service upstream) — raise lỗi rõ ràng, chuyển status → `failed_at_render_scenes` với `error_message` mô tả mismatch cụ thể (không dispatch command, không gộp sai lệch âm thầm).
- **Event trung gian**: `scene_rendered` (progress, per-scene, KHÔNG advance state machine — chỉ forward qua `progress.fanout`)
- **Event chờ (kết thúc bước)**: `rendering_completed` (chứa `scene_clip_paths: [{scene_index, clip_path}]`)

### Bước 5 — Assemble Video (Data Aggregation — Question 2)
- **Trigger**: nhận `rendering_completed`
- **Logic**: gộp `clip_path` (từ `rendering_completed`, theo `scene_index`) với `audio_path` đã lưu ở `Project.scenes` từ bước 3 (KHÔNG lấy lại từ `rendering_completed` — Rendering Service không trả audio_path). Dispatch command `assemble_video` (payload: `scenes: [{scene_index, clip_path, audio_path}]` + `background_music_path?` đã lưu từ input ban đầu), chuyển status → `assembling_video`.
- **Event chờ**: `video_assembled` (chứa `video_path`)

### Kết thúc Saga Render (ngầm định)
- **Trigger**: nhận `video_assembled`
- **Logic**: lưu `video_path` vào `Project`, chuyển status → `ready_to_publish`. Không có command/event riêng cho transition này — chỉ là state transition nội bộ. Saga Render kết thúc thành công tại đây.

## Saga 2: Publish Video (1 bước)

### Bước 6 — Publish Video
- **Trigger**: `POST /v1/sagas/publish` (Creator xác nhận metadata YouTube, Story E3)
- **Precondition**: `Project.Status == ready_to_publish` (else `409 Conflict`)
- **Logic**: `StartPublishSagaUseCase` lưu youtube metadata (`youtube_title`, `description?`, `tags?`, `visibility`) vào `Project`, tạo `SagaStep{step: publish_video, status: in_progress}`, dispatch command `publish_video` (payload: `video_path` + youtube metadata), chuyển status → `publishing`.
- **Event chờ**: `video_published` (chứa `youtube_video_url`)
- **Kết thúc**: lưu `youtube_video_url` vào `Project`, chuyển status → `published`. Saga Publish kết thúc — không dispatch command tiếp theo (bước cuối).

## Xử lý lỗi (mọi bước, chung 1 pattern)
- Nhận event `*_failed` (6 loại) hoặc DLQ delivery cho command tương ứng → chuyển `Project.Status` → `failed_at_<step>` với `error_message`, publish progress message `status: failed`. KHÔNG rollback artifact đã tạo ở các bước trước (animation/audio giữ nguyên).
- Creator thấy lỗi qua SSE (`progress.fanout` → Gateway), gọi `POST /v1/projects/{id}/retry` → `RetryStepUseCase` tái tạo command payload từ `Project` hiện tại, dispatch lại với `message_id` mới, chuyển status về trạng thái "đang xử lý" của đúng bước đó (retry-by-step, không phải compensating rollback).

## Concurrent Projects (Question 6)
Không giới hạn số project/Saga chạy đồng thời ở tầng Orchestrator — mỗi `project_id` độc lập hoàn toàn, không share state. Giới hạn thực tế đến từ tài nguyên các service downstream (prefetch=1), không phải business logic Orchestrator.

## Progress Reporting (ADR-0017)
Sau MỖI lần `HandleStepEventUseCase` xử lý xong 1 event (kể cả `scene_rendered`), publish 1 `ProgressMessage` tới `progress.fanout` — xem `domain-entities.md` cho cấu trúc.
