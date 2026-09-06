# Messaging Design — Unit 8: Orchestrator Service

## Delivery Guarantee
At-least-once (kế thừa Unit 1/RabbitMQ) cho cả command (Outbox) và event (Inbox dedupe). Không exactly-once (NFR Requirements Question 7).

## Saga Role: Orchestrator (not participant)
Orchestrator Service là **central coordinator** cho cả 2 Saga:
- **Saga Render Pipeline** (5 bước): `ParseScript → ClassifyScenes → SynthesizeSpeech → RenderScenes → AssembleVideo → (ready_to_publish)`
- **Saga Publish** (1 bước): `PublishVideo → (published)`

Compensating action: **retry-by-step**, KHÔNG rollback (Functional Design Rule 8) — artifact hợp lệ từ các bước trước giữ nguyên, chỉ bước lỗi được retry với `message_id` mới.

## Event Schema (Envelope, kế thừa Unit 1's producer.py convention)
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "string",
  "event_type": "string",
  "payload": { "..." : "tùy loại event, xem từng unit's interface-contracts.md" },
  "timestamp": "ISO8601"
}
```

## Topics/Queues (Unit 1's topology)
| Exchange/Queue | Direction | Nội dung |
|---|---|---|
| `commands.direct` (routing key theo service) | Publish | 6 loại command: `parse_script`, `classify_scenes`, `synthesize_speech`, `render_scenes`, `assemble_video`, `publish_video` |
| `orchestrator.events` | Consume | 12 loại event: 6 success + 6 failure |
| `progress.fanout` | Publish | `ProgressMessage` (ADR-0017), sau mỗi lần `HandleStepEventUseCase` xử lý xong |
| `*.commands.dlq` (6 queue, pattern chung) | Consume | Command dead-lettered — Orchestrator đánh dấu `failed_at_<step>` |

## Event Versioning
Không cần versioning phức tạp (schema registry, v2 event...) — hệ thống nội bộ, tất cả service deploy cùng lúc qua docker-compose (nhất quán các unit khác, không có backward-compatibility concern giữa các phiên bản service khác nhau chạy song song).

## Inbox/Outbox Pattern — Semantic Khác Biệt (ADR-0019)
Khác với Unit 2–7 (Outbox dùng để publish EVENT kết quả xử lý), Orchestrator dùng:
- **Outbox** (`outbox_events`) để đảm bảo COMMAND publish đúng 1 lần dù crash giữa transaction cập nhật state và gửi command. Relay: `OutboxRelay` (goroutine), poll mỗi 500ms.
- **Inbox** (`processed_messages`) để dedupe EVENT nhận vào theo `message_id`, tránh xử lý trùng khi RabbitMQ redeliver.

### Schema
```sql
-- outbox_events (commands to send)
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    routing_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ NULL
);

-- processed_messages (Inbox — dedupe incoming events)
CREATE TABLE processed_messages (
    message_id UUID PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Relay Mechanism
Polling publisher (không phải CDC/Debezium) — `OutboxRelay` chạy như 1 goroutine nền trong `main.go`, `SELECT ... WHERE published_at IS NULL` mỗi 500ms, publish qua `amqp.Publisher`, `UPDATE published_at` sau khi publish thành công.

## Idempotency Summary
Xem `nfr-design-patterns.md` — 2 tầng (message-level Inbox + step-level `SagaStep.status` guard).

## Connection Resilience (Revision 2026-09-06, ADR-0022)
Trước đây `amqp.Publisher`/`amqp.Consumer` giữ thẳng 1 `*amqp.Channel` mở lúc startup, không có cơ chế phục hồi — khi broker đóng channel (restart RabbitMQ, network blip), mọi publish sau đó fail vĩnh viễn với `channel/connection is not open`, khiến `OutboxRelay` kẹt vô hạn ở command đang publish dở và toàn bộ consumer (`orchestrator.events` + 6 DLQ) ngừng nhận message — chỉ khôi phục được bằng cách restart thủ công container.

Nay `amqp.ConnectionManager` sở hữu vòng đời Connection/Channel:
- `Publisher`/`Consumer` gọi `ConnectionManager.Channel()` mỗi lần dùng thay vì cache 1 lần — không bao giờ dùng phải channel cũ đã chết.
- Watchdog goroutine nền lắng nghe `conn.NotifyClose`/`channel.NotifyClose`, tự redial với exponential backoff (mặc định 1s → cap 30s, cấu hình qua `RABBITMQ_RECONNECT_INITIAL_DELAY_MS`/`RABBITMQ_RECONNECT_MAX_DELAY_MS`).
- Sau reconnect thành công, `ConnectionManager` gọi các callback đăng ký qua `OnReconnect(...)` — `Consumer` dùng để gọi lại `Start(ctx)`, đăng ký lại subscription trên `orchestrator.events` + 6 DLQ queue (broker tự huỷ mọi consumer khi channel đóng).
- Không cần khai báo lại exchange/queue khi reconnect — topology đã durable, do Unit 1 (RabbitMQ Infrastructure) khai báo 1 lần.
