# Domain Entities — Unit 8: Orchestrator Service

## Entity: Project
Đại diện 1 video project của Creator, là aggregate root và "single source of truth" tích lũy dữ liệu qua toàn bộ Saga Render + Publish.

| Field | Type | Ghi chú |
|---|---|---|
| `project_id` | string (uuid) | Khóa chính, do Creator/GUI sinh lúc tạo project |
| `status` | `ProjectStatus` (enum) | Xem bên dưới |
| `script_content` | string | Input ban đầu, bước 1 |
| `plugin_id` | string | Input ban đầu |
| `voice_language` | `"vi" \| "en"` | Input ban đầu |
| `background_music_path` | string? | Input ban đầu (Rule 3), optional |
| `scenes` | `Scene[]` | Tích lũy dần qua bước 1–4 |
| `video_path` | string? | Set sau bước 5 (`video_assembled`) |
| `youtube_title` | string? | Set lúc Saga Publish bắt đầu |
| `youtube_description` | string? | Set lúc Saga Publish bắt đầu |
| `youtube_tags` | string[]? | Set lúc Saga Publish bắt đầu |
| `youtube_visibility` | `"public"\|"unlisted"\|"private"`? | Set lúc Saga Publish bắt đầu |
| `youtube_video_url` | string? | Set sau `video_published` |
| `error_message` | string? | Set khi status = `failed_at_<step>` |

## Value Object: Scene
Phần tử của `Project.scenes`, tích lũy dữ liệu theo `scene_index` qua các bước.

| Field | Type | Nguồn / Bước lưu |
|---|---|---|
| `scene_index` | int | `script_parsed`, bước 1 |
| `narration_text` | string | `script_parsed`, bước 1 |
| `illustration_hint` | string | `script_parsed`, bước 1 |
| `code_snippet` | string? | `script_parsed`, bước 1 |
| `code_language` | string? | `script_parsed`, bước 1 |
| `category` | string | `scenes_classified`, bước 2 |
| `animation_template_id` | string | `scenes_classified`, bước 2 |
| `audio_path` | string | `speech_synthesized`, bước 3 (single source of truth — Rule 2) |
| `duration_seconds` | float | `speech_synthesized`, bước 3 |
| `clip_path` | string? | `rendering_completed`, bước 4 |

## Entity: SagaStep
Theo dõi trạng thái từng bước trong 1 Saga instance.

| Field | Type | Ghi chú |
|---|---|---|
| `saga_id` | string (uuid) | Sinh mới bởi Orchestrator lúc `StartRenderSagaUseCase`/`StartPublishSagaUseCase` |
| `step_name` | string | vd. `parse_script`, `classify_scenes`, ..., `publish_video` |
| `status` | `"in_progress"\|"completed"\|"failed"` | |
| `error_message` | string? | Set khi `status = failed` |

Quan hệ: 1 `Project` có thể có nhiều `saga_id` theo thời gian (Saga Render rồi Saga Publish là 2 saga_id khác nhau cho cùng 1 project_id — Question 9, LLD). Mỗi `saga_id` có N `SagaStep` (1 record mỗi bước đã/đang xử lý).

## Enum: ProjectStatus
```
draft
parsing_script
classifying_scenes
synthesizing_speech
rendering
assembling_video
ready_to_publish
publishing
published
failed_at_parse_script
failed_at_classify_scenes
failed_at_synthesize_speech
failed_at_render_scenes
failed_at_assemble_video
failed_at_publish_video
```
9 trạng thái "happy path" (draft → ... → published) + 6 trạng thái `failed_at_<step>` (1 cho mỗi bước có thể lỗi trong Saga Render + Publish).

**Transition hợp lệ** (Rule 4): chỉ đi từ đúng trạng thái liền trước theo thứ tự trên; bất kỳ trạng thái nào cũng có thể transition sang `failed_at_<step_hiện_tại>` khi nhận event `*_failed` tương ứng; `failed_at_<step>` → trạng thái "đang xử lý" của step đó khi retry thành công (vd. `failed_at_render_scenes` → `rendering`).

## Value Object: ProgressMessage (ADR-0017, Rule 7)
```json
{
  "project_id": "string",
  "step": "string",
  "status": "in_progress | completed | failed",
  "scene_index": "int?",
  "scene_total": "int?",
  "error_message": "string?"
}
```
Publish tới `progress.fanout` sau mỗi lần `HandleStepEventUseCase` xử lý xong 1 event (bao gồm event progress `scene_rendered`, không advance state machine).

## Value Object: Command Envelope (outbound, 6 loại)
Payload cụ thể từng command tham chiếu `interface-contracts.md`; nguồn dữ liệu gộp theo `business-logic-model.md` (Bước 1–6).

## Quan hệ tổng quan
```
Project (1) ---- (N, theo thời gian) SagaStep   [liên kết qua project_id ⇄ saga_id]
Project (1) ---- (N) Scene   [embedded, không phải bảng riêng theo domain model — nhưng có thể map sang bảng con ở tầng persistence]
```
