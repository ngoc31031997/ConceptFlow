# NFR Requirements — Unit 8: Orchestrator Service

## Scalability
- Không giới hạn số project/Saga chạy đồng thời ở tầng business logic (mỗi `project_id` cô lập hoàn toàn — Functional Design Rule 6).
- Mỗi event AMQP xử lý trong 1 goroutine riêng, không giới hạn cứng số goroutine đồng thời — phù hợp quy mô 1 Creator/máy dev cá nhân.
- Postgres connection pool (pgx) mặc định `max_conns=10`, override qua env var `DATABASE_MAX_CONNS` nếu cần.

## Performance
- `OutboxRelay` poll `outbox_events` mỗi **500ms** (env var `OUTBOX_POLL_INTERVAL_MS`, default 500) — độ trễ dispatch command tối đa ~0.5s, không đáng kể so với thời gian xử lý thực tế của các bước Saga.

## Availability & Reliability
- Dựa vào RabbitMQ's at-least-once delivery: message chưa ack được redeliver khi consumer reconnect.
- Inbox (`processed_messages`, dedupe theo `message_id`) đảm bảo idempotency khi message được xử lý lại.
- Docker Compose `restart: unless-stopped` cho container recovery — không cần health-check/circuit-breaker phức tạp ở quy mô single-node local.

## Security
- Không cần authentication/authorization giữa các service nội bộ (Gateway ↔ Orchestrator, Orchestrator ↔ service nghiệp vụ) — toàn bộ chạy trong 1 Docker network cô lập, không expose internet (trừ port GUI/Gateway). Nhất quán Security Baseline extension = No (`aidlc-state.md`).
- RabbitMQ dùng credential từ `.env`, không phải secret production-grade.

## Maintainability
- Structured JSON logging qua `log/slog` (Go standard library) — mỗi log line: `timestamp`, `level`, `message`, `saga_id?`, `project_id?`, `step?`.
- Correlation: `saga_id` (AMQP context) / `X-Request-ID` (REST context) injected vào mọi log field liên quan (LLD's `adapters/logging/correlation.go`).

## Messaging & Event Participation
- Delivery guarantee: **at-least-once** cho cả command (Outbox) và event (Inbox dedupe) — không cần exactly-once.
- Orchestrator consume 12 loại event + publish 6 loại command + progress message (`progress.fanout`, ADR-0017).

## Distributed Transaction Participation
- Orchestrator Service là **Saga Orchestrator** duy nhất (không phải participant) — sở hữu toàn bộ định nghĩa thứ tự bước, state machine, compensating action.
- Compensating action: retry-by-step (không rollback) — nhất quán ADR-0007 và Functional Design Rule 8.
- 6 service nghiệp vụ còn lại là Saga participants (choreography step).

## Caching
- Không áp dụng — không có endpoint đọc nặng hoặc external API call cần cache ở Orchestrator (GET /v1/projects/{id} đọc trực tiếp từ Postgres, tần suất thấp, dữ liệu thay đổi liên tục trong lúc Saga chạy nên cache sẽ nhanh stale).

## Usability
- Không áp dụng trực tiếp (Orchestrator không có UI) — progress message (ADR-0017) là input cho GUI hiển thị tiến trình, format đã xác nhận ở Functional Design Rule 7.
