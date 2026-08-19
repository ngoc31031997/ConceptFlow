# NFR Design Patterns — Unit 5: Rendering Service

## CRUD vs CQRS
CRUD đơn giản trên `outbox_events`/`processed_messages` (bảng kỹ thuật, ADR-0013) — không phải business data model, không phải CQRS.

## Resilience Pattern
Không retry nội bộ. Manim lỗi/timeout (sau `RENDER_TIMEOUT_SECONDS`) → `AnimationEngineError` → `rendering_failed` ngay, để Orchestrator quyết định retry (transient). Lỗi validation (zero-trust, `UnsupportedTemplateError`/`InvalidDurationError`) là permanent — raise ngay, không cần retry tự động.

## Idempotency Pattern
2 tầng, mirror TTS Service:
- **Message-level**: Inbox (`processed_messages`) dedupe `message_id` — 1 command = 1 message_id, dù publish nhiều event.
- **Artifact-level**: kiểm tra file `.mp4` tồn tại tại `/shared/{project_id}/animations/{scene_index}.mp4` trước khi render lại (Functional Design Rule 5).

Không cần lock/race-condition handling — Orchestrator gọi tuần tự, cùng lý do đã chấp nhận ở TTS Service.

## Saga Pattern
Participant trực tiếp, bước "Render Scenes" (sau "Synthesize Speech", ADR-0014) trong Saga Render Pipeline. Không có compensating action — idempotent theo `project_id`+`scene_index`.

## Event-Driven Design
Consumer `render_scenes` (queue `rendering.commands`), publish 4 loại event: `scene_render_started`, `scene_rendered` (per scene, tiến trình — Low-Level Design Question 9), `rendering_completed`, `rendering_failed` (1 lần, cuối batch). **Khác biệt so với Unit 2/3/4**: nhiều Outbox row/command (2N+1 với N scene thành công) thay vì 1 row — Outbox pattern vẫn hoạt động bình thường, không giới hạn 1 event/command.

## Inbox/Outbox Pattern
PostgreSQL-backed (`rendering-db`, ADR-0013), cùng kiến trúc Unit 2/3/4 — `OutboxRelay` publish bất đồng bộ. **Khác biệt về transaction boundary** so với Unit 2/3/4 (vốn ghi Outbox + Inbox trong đúng 1 transaction duy nhất): mỗi `scene_render_started`/`scene_rendered` được COMMIT RIÊNG NGAY LẬP TỨC (transaction độc lập, không gộp) — để `OutboxRelay` publish kịp thời, cho GUI thấy tiến trình thực (Low-Level Design Question 9), thay vì phải đợi cả batch xong mới thấy event đầu tiên. CHỈ event cuối cùng (`rendering_completed`/`rendering_failed`) được ghi CÙNG transaction với việc mark Inbox (`processed_messages`) — đây là điểm đánh dấu "command đã xử lý xong", đúng ý nghĩa atomicity của Inbox/Outbox pattern (đảm bảo không mark Inbox nếu chưa chắc chắn đã publish được kết quả cuối).

## Security Pattern
Zero-trust validation trong domain layer (`RenderSceneUseCase`, Functional Design Business Rule 1) — mạnh hơn các unit khác (validate toàn bộ input, không chỉ phần cần cho logic). Không auth/rate-limit riêng — chỉ Orchestrator gửi command qua RabbitMQ nội bộ.
