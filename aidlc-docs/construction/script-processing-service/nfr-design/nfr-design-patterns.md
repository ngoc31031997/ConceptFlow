# NFR Design Patterns — Unit 4: Script Processing Service

## CRUD vs CQRS
CRUD đơn giản trên `outbox_events`/`processed_messages` (bảng kỹ thuật, ADR-0013) — không phải business data model, không phải CQRS.

## Resilience Pattern
Không retry nội bộ. Lỗi cú pháp là permanent (cần Creator sửa script) — publish `parse_failed` ngay, không tự phục hồi qua retry. Khác với `TTSEngineError` (transient) ở TTS Service.

## Idempotency Pattern
Chỉ 1 tầng — message-level qua Inbox (`processed_messages`). Không có artifact-level (không có file/side-effect bền vững nào để kiểm tra tồn tại — parse là pure computation).

## Saga Pattern
Participant trực tiếp, bước đầu tiên "Parse Script" của Saga Render Pipeline (`services.md`). Không có compensating action — stateless, không side-effect bền vững.

## Event-Driven Design
Consumer `parse_script` (queue `script_processing.commands`), producer `script_parsed`/`parse_failed` (integration event, envelope chuẩn Unit 1, → `orchestrator.events`).

## Inbox/Outbox Pattern
PostgreSQL-backed (`script-processing-db`, ADR-0013) — giống hệt kiến trúc Unit 2/Unit 3:
- **Outbox**: consumer ghi `script_parsed`/`parse_failed` vào `outbox_events` trong CÙNG transaction với Inbox mark.
- **Relay**: `OutboxRelay` polling task publish event chưa gửi.

## Security Pattern
Validate input trong `MarkdownScriptParser` (domain layer). Không auth/rate-limit riêng — chỉ Orchestrator gửi command qua RabbitMQ nội bộ.
