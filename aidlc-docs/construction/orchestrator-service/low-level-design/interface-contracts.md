# Interface Contracts — Unit 8: Orchestrator Service

## REST API (versioned per ADR-0008, gọi bởi API Gateway — Unit 9)

### `POST /v1/sagas/render`
- **Input**: `{ "project_id": "string", "script_content": "string", "plugin_id": "string", "voice_language": "vi" | "en" }`
- **Output 201**: `{ "saga_id": "uuid", "status": "started" }`
- **Behavior**: tạo `Project` (status=`draft`), tạo `saga_id` mới (Question 9), publish command `parse_script` (qua Outbox), status → `parsing_script`.

### `POST /v1/sagas/publish`
- **Input**: `{ "project_id": "string", "youtube_title": "string", "description": "string?", "tags": ["string"]?, "visibility": "public"|"unlisted"|"private" }`
- **Output 201**: `{ "saga_id": "uuid", "status": "started" }`
- **Precondition**: `Project.Status == "ready_to_publish"` (else `409 Conflict`).
- **Behavior**: publish command `publish_video`, status → `publishing`.

### `GET /v1/projects/{project_id}`
- **Output 200**: `{ "project_id", "status", "video_path"?, "scenes", "plugin_id", "voice_language", "youtube_video_url"? }`
- **Output 404**: project không tồn tại.

### `POST /v1/projects/{project_id}/retry` (Question 8)
- **Output 200**: `{ "saga_id": "uuid", "status": "string" }` (status mới, vd. `rendering` nếu retry `render_scenes`)
- **Output 409**: `Project.Status` không phải `failed_at_<step>`.
- **Behavior**: đọc bước lỗi từ `Project.Status`, tái tạo command payload từ dữ liệu đã lưu, publish lại (Outbox) với `message_id` MỚI.

### Correlation ID
Header `X-Request-ID` — Orchestrator không tự sinh REST correlation (Gateway chịu trách nhiệm, mirror Unit 2/7's convention). `saga_id` được Orchestrator SINH MỚI khi khởi tạo Saga (Question 9) — không phải Gateway truyền vào.

## AMQP Interface

### Commands Published (6 loại, tới `commands.direct`)
| Command | Routing Key | Payload (tóm tắt, xem từng unit's interface-contracts.md để biết đầy đủ) |
|---|---|---|
| `parse_script` | `script_processing` | `{ script_content }` |
| `classify_scenes` | `content_plugin` | `{ plugin_id, scenes }` (từ `script_parsed`) |
| `synthesize_speech` | `tts` | `{ scenes: [{scene_index, narration_text, language}] }` (từ `scenes_classified` + `voice_language`) |
| `render_scenes` | `rendering` | `{ scenes: [...] }` — Orchestrator GỘP dữ liệu từ `script_parsed`+`scenes_classified`+`speech_synthesized` (đã ghi nhận là ràng buộc thiết kế cho Unit 8 từ Unit 5's Functional Design) |
| `assemble_video` | `video_assembly` | `{ scenes: [{scene_index, clip_path, audio_path}], background_music_path? }` — từ `rendering_completed` |
| `publish_video` | `publisher` | `{ video_path, title, description, tags, visibility }` — từ `video_assembled` + Saga Publish's input |

### Events Consumed (12 loại, từ `orchestrator.events`)
6 success (`script_parsed`, `scenes_classified`, `speech_synthesized`, `rendering_completed`, `video_assembled`, `video_published`) + 6 failure (`parse_failed`, `classification_failed`, `synthesis_failed`, `rendering_failed`, `assembly_failed`, `publish_failed`). Progress event `scene_rendered` (Rendering Service, per-scene) cũng consume để forward qua `progress.fanout` (không advance state machine).

### Progress Messages Published (ADR-0017, tới `progress.fanout`)
```json
{
  "project_id": "string",
  "step": "string (vd. render_scenes)",
  "status": "in_progress | completed | failed",
  "scene_index": "int?",
  "scene_total": "int?",
  "error_message": "string?"
}
```
Publish sau MỖI lần `HandleStepEventUseCase` xử lý xong 1 event (kể cả progress event `scene_rendered`).

### DLQ Consumption
1 consumer chung lắng nghe pattern 6 queue `*.commands.dlq` (Unit 1's ghi chú) — khi 1 command bị dead-letter (retry exceeded ở tầng RabbitMQ, hiếm khi xảy ra vì các service không tự retry nội bộ), Orchestrator đánh dấu project status = `failed_at_<step>` với `error_message = "message dead-lettered: exceeded delivery limit"`.

## Event Sequence Per Saga
Saga Render Pipeline (thành công): 5 command + 5 event success (+ N event `scene_rendered` progress) + 1 event kết thúc ngầm định (status → `ready_to_publish`, không có command/event riêng cho việc này — chỉ là state transition nội bộ sau `video_assembled`).
Saga Publish (thành công): 1 command + 1 event.

## Delivery Guarantee & Idempotency (Question 6)
At-least-once (kế thừa Unit 1). **Inbox**: dedupe `message_id` của event NHẬN VÀO (`processed_messages`), tránh xử lý trùng khi RabbitMQ requeue. **Outbox**: dùng để gửi COMMAND (không phải event như các unit khác) — đảm bảo command được publish đúng 1 lần dù Orchestrator crash giữa lúc cập nhật state và gửi command; `OutboxRelay` poll `outbox_events` (semantic: "commands to send") và publish qua `amqp.Publisher`.

## Saga Instance Tracking (Question 5)
Mỗi event envelope có `saga_id`+`project_id`. `HandleStepEventUseCase` tra `saga_steps` theo `saga_id`+`step_name` (suy từ `event_type`, vd. `scenes_classified` → step `classify_scenes`) để xác nhận bước đang `in_progress` — event không khớp bước đang chờ → log warning, KHÔNG xử lý (chống race condition/redelivery bất thường), vẫn ack message (Inbox đã dedupe theo `message_id`, không phải theo step).

## Internal Port Contracts (internal/domain/ports.go)
```go
type CommandPublisherPort interface {
    PublishCommand(ctx context.Context, routingKey string, envelope Envelope) error
}

type ProgressPublisherPort interface {
    PublishProgress(ctx context.Context, msg ProgressMessage) error
}

type ProjectRepositoryPort interface {
    Get(ctx context.Context, projectID string) (*Project, error)
    Save(ctx context.Context, project *Project) error
    UpdateStatus(ctx context.Context, projectID string, status ProjectStatus) error
    GetStep(ctx context.Context, sagaID, stepName string) (*SagaStep, error)
    UpdateStep(ctx context.Context, step *SagaStep) error
}
```
