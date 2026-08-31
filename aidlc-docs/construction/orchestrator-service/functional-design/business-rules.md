# Business Rules — Unit 8: Orchestrator Service

## Rule 1: Scene Data Aggregation Integrity (Question 1)
Khi gộp dữ liệu 3 nguồn (`script_parsed`, `scenes_classified`, `speech_synthesized`) để tạo payload `render_scenes`:
- Gộp theo `scene_index` làm khóa chính.
- Cả 3 nguồn PHẢI có cùng số lượng scene với cùng tập giá trị `scene_index` (bất biến tự nhiên vì cả 3 đều xuất phát từ cùng 1 `script_parsed` ban đầu).
- Nếu vi phạm (mismatch số lượng hoặc tập scene_index): **không** gộp sai lệch âm thầm — coi là lỗi hệ thống, raise lỗi tường minh, chuyển `Project.Status → failed_at_render_scenes` với `error_message` mô tả cụ thể mismatch.

## Rule 2: Audio Path Single Source of Truth (Question 2)
`audio_path` cho mỗi scene CHỈ có 1 nguồn hợp lệ: event `speech_synthesized` (bước 3). Khi gộp payload `assemble_video`, Orchestrator dùng `audio_path` đã lưu ở `Project.scenes` từ bước 3 — KHÔNG được đọc/ghi đè từ `rendering_completed` (Rendering Service không trả lại `audio_path`, chỉ trả `clip_path`, theo interface-contracts.md).

## Rule 3: Background Music Path là Input Tĩnh (Question 3)
`background_music_path` là optional field trong input `POST /v1/sagas/render` (Creator chọn lúc cấu hình project, Story C5). Được lưu vào `Project` ngay lúc khởi tạo Saga (bước 1) và tái sử dụng nguyên trạng khi dispatch `assemble_video` (bước 5) — không có bước Saga riêng để lấy giá trị này, không thay đổi trong suốt vòng đời Saga.

## Rule 4: State Transition Validity (Question 4)
- Mỗi transition trong `Project.Status` chỉ hợp lệ khi status HIỆN TẠI đúng bằng trạng thái liền trước theo thứ tự Saga đã định nghĩa (vd. `synthesizing_speech → rendering` chỉ hợp lệ nếu status hiện tại đang là `synthesizing_speech`).
- `HandleStepEventUseCase` validate transition này TRƯỚC khi update — nếu event đến không khớp bước đang chờ (`SagaStep.status != in_progress`), log warning và bỏ qua xử lý (Flow 6, "unexpected event" safeguard), vẫn ack message.
- `POST /v1/sagas/publish` chỉ hợp lệ khi `Project.Status == ready_to_publish` — vi phạm trả `409 Conflict`.
- `POST /v1/projects/{id}/retry` chỉ hợp lệ khi `Project.Status` có dạng `failed_at_<step>` — vi phạm trả `409 Conflict`.

## Rule 5: Retry Payload Reconstruction (Question 5)
`Project` là "single source of truth" tích lũy đầy đủ dữ liệu cần thiết để tái tạo command payload của BẤT KỲ bước nào, không cần lưu riêng từng payload command đã gửi trước đó:
- `script_content`, `plugin_id`, `voice_language`, `background_music_path?` — input ban đầu (bước 1).
- `scenes[]` — tích lũy dần: `narration_text`/`illustration_hint`/`code_snippet`/`code_language` (sau bước 1), `category`/`animation_template_id` (sau bước 2), `audio_path`/`duration_seconds` (sau bước 3), `clip_path` (sau bước 4).
- `video_path` — sau bước 5.
- youtube metadata (`youtube_title`, `description?`, `tags?`, `visibility`) — input Saga Publish.

Retry bước N: đọc `Project` hiện tại, tái tạo payload đúng format command bước N từ dữ liệu đã tích lũy, publish lại với `message_id` MỚI (Outbox) — không cần gọi lại các bước trước.

## Rule 6: Concurrency Isolation (Question 6)
Không giới hạn số Saga/project chạy đồng thời ở tầng business logic Orchestrator. Mỗi `project_id` là 1 đơn vị cô lập hoàn toàn — không đọc/ghi state của project khác. Giới hạn tài nguyên (nếu có) là vấn đề hạ tầng của các service downstream, không phải business rule.

## Rule 7: Progress Message Content (Question 7)
Cấu trúc `ProgressMessage` cố định (xem `domain-entities.md`): `project_id`, `step`, `status`, `scene_index?`, `scene_total?`, `error_message?`. Với event progress `scene_rendered`: map `step = "render_scenes"`, `status = "in_progress"`, kèm `scene_index`+`scene_total` từ event payload. Publish 1 message sau MỖI lần `HandleStepEventUseCase` xử lý xong (kể cả các event progress không advance state machine).

## Rule 8: No Compensating Rollback
Khi 1 bước thất bại (`*_failed` event hoặc DLQ), Orchestrator KHÔNG rollback/xóa artifact đã tạo ở các bước trước đó (animation clip, audio file...). Compensating action duy nhất là retry-by-step (Rule 5) — không phải Saga rollback truyền thống.

## Rule 9: Inbox/Outbox Semantics (kế thừa LLD, ghi nhận lại như business rule)
- **Inbox**: dedupe event NHẬN VÀO theo `message_id` — đảm bảo 1 event chỉ được xử lý business logic đúng 1 lần dù bị RabbitMQ redeliver.
- **Outbox**: dùng để đảm bảo COMMAND gửi đi đúng 1 lần dù Orchestrator crash giữa lúc cập nhật state và gửi command (khác các unit khác dùng Outbox cho event).
