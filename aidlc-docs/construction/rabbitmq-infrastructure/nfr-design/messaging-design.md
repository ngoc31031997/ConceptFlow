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
**Không áp dụng** ở unit này — Inbox/Outbox pattern liên quan đến việc đảm bảo publish event đồng bộ với transaction database, nhưng RabbitMQ không sở hữu database nghiệp vụ nào. Các service khác (nếu cần Outbox pattern khi ghi state + publish event) sẽ tự quyết định ở NFR Design riêng của unit đó (vd. Unit 8 — Orchestrator Service, khi ghi Saga state).

## Idempotency
Idempotency được đảm bảo ở tầng consumer (mỗi service nghiệp vụ), không phải ở RabbitMQ — mỗi consumer kiểm tra `message_id` đã xử lý chưa trước khi thực hiện side-effect (chi tiết cơ chế dedupe cụ thể xác định ở Low-Level Design của từng service).
