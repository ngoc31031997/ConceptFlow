# NFR Design Patterns — Unit 2: Content Plugin Service

**Revision (2026-08-07, ADR-0013)**: Idempotency Pattern below (in-memory `set[message_id]`) is superseded by a PostgreSQL-backed Inbox/Outbox pattern, retrofitted as part of a system-wide learning exercise. Registry itself remains in-memory (no change — plugin discovery is still pure startup-time computation, not persisted state).

## CRUD vs CQRS
Simple CRUD only, on two tables: `outbox_events` (queued events pending publish) and `processed_messages` (durable idempotency record) — not a business data model, purely messaging-plumbing tables (ADR-0013). Not CQRS: no read/write model split, no query-optimized projection needed at this scale.

## Idempotency & Outbox Pattern
**Superseded — see Revision above.** PostgreSQL-backed Inbox/Outbox (`content-plugin-db`, ADR-0013):
- **Inbox**: `processed_messages` table — `InboxRepository.has_processed`/`mark_processed`, durable across restarts (previously an in-memory `set[message_id]` that reset on restart).
- **Outbox**: consumer writes the outgoing event (`scenes_classified`/`classification_failed`) to `outbox_events` in the SAME DB transaction as the Inbox mark — atomicity between "processed this command" and "recorded the resulting event," which the in-memory approach couldn't guarantee (a crash between classify and publish could previously lose the event; now it can't, since the write is transactional and the event survives a restart in the table).
- **Relay**: `OutboxRelay` (background polling task, ~1s interval) reads unpublished rows and publishes them to RabbitMQ, then marks `published_at`.
- Original rationale for accepting dedupe loss on restart (no durable side-effect to duplicate) no longer applies as a justification — it's now moot, since dedupe state itself survives restarts.

## Resilience Pattern
Dùng cơ chế reconnect tự động có sẵn của `aio-pika` (built-in connection recovery) cho kết nối RabbitMQ — không tự viết circuit breaker riêng vì chỉ có 1 dependency hạ tầng duy nhất.

## Saga Pattern
Participant (không phải coordinator) cho bước "Classify Scenes". Không có compensating action (theo NFR Requirements — pure computation, không side-effect bền vững).

## Security Pattern
Input validation qua Pydantic schema (FastAPI). Không auth/rate-limit riêng (chỉ gọi nội bộ từ Gateway/Orchestrator).
