# Messaging Design — Unit 1: RabbitMQ Infrastructure

## Delivery Guarantee
At-least-once, với consumer ack sau khi xử lý xong (manual ack, không dùng auto-ack) — nếu consumer crash trước ack, RabbitMQ requeue message.

## Event Schema & Versioning
Message payload dạng JSON, mỗi message có envelope chung:
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "uuid",
  "schema_version": "1.0",
  "timestamp": "ISO8601",
  "payload": { ... }
}
```
- **Versioning approach**: Payload/field-level, additive-only (không xóa/đổi tên field trong version hiện tại; version mới nếu cần breaking change) — phù hợp vì chỉ có 1 producer/consumer cho mỗi loại message (nội bộ hệ thống, không phải public API), rủi ro breaking change thấp.
- Chi tiết payload từng loại command/event: xem `component-methods.md` (Application Design).

## Event Type: Integration Events
Toàn bộ message là **integration event** (dùng để phối hợp giữa các service kỹ thuật), không phải domain event công khai cho hệ thống bên ngoài — không cần schema registry phức tạp ở quy mô hiện tại.

## Saga Role
**N/A (transport only)** — xem `nfr-design-patterns.md`. Compensating actions được định nghĩa và thực thi bởi Orchestrator Service (Unit 8), không phải bởi RabbitMQ.

## Inbox/Outbox Pattern
**Không áp dụng ở unit này** — RabbitMQ (Unit 1) không sở hữu database nghiệp vụ nào, nên "Inbox/Outbox pattern" không áp dụng cho chính hạ tầng message broker.

**Revision (2026-08-07, ADR-0013)**: Các business service (Unit 2, 3, 4, và các unit tương lai) nay retrofit Inbox/Outbox pattern riêng lẻ (PostgreSQL per service) như một bài tập học kiến trúc microservices có chủ đích, không phải vì yêu cầu quy mô hiện tại đòi hỏi. Xem ADR-0013 và `nfr-design-patterns.md` của từng unit tương ứng.

## Idempotency
Idempotency được đảm bảo ở tầng consumer (mỗi service nghiệp vụ), không phải ở RabbitMQ — mỗi consumer kiểm tra `message_id` đã xử lý chưa trước khi thực hiện side-effect (chi tiết cơ chế dedupe cụ thể xác định ở Low-Level Design của từng service).
