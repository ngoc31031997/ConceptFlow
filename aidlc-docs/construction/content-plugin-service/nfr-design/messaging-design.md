# Messaging Design — Unit 2: Content Plugin Service

## Delivery Guarantee
At-least-once (kế thừa Unit 1) — manual ack sau khi xử lý xong, kể cả khi kết quả là lỗi validation (ack để tránh vòng lặp retry vô ích cho lỗi logic — theo Rule 3, Functional Design).

## Event Schema
Theo `interface-contracts.md` (Low-Level Design) — envelope chuẩn (`message_id`, `saga_id`, `project_id`, `schema_version`, `timestamp`, `payload`), additive-only versioning.

## Saga Role
Participant. Không compensating action (pure computation).

## Idempotency
In-memory `set[message_id]`, TTL 24h — xem `nfr-design-patterns.md`.

## Inbox/Outbox Pattern
**Không áp dụng** — unit không ghi dữ liệu vào database trong cùng transaction với việc publish event (không có database). Publish event trực tiếp sau khi xử lý xong, không cần Outbox relay.
