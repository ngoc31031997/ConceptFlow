# Dependency Injection — Unit 8: Orchestrator Service

## Mechanism
Constructor injection thủ công (Go idiom chuẩn) — không dùng DI framework (`wire`, `fx`). Nhất quán tinh thần constructor injection đã dùng ở mọi unit Python trước, chuyển sang idiom Go.

## What Gets Injected vs Constructed Directly
- **Injected (abstraction)**: `CommandPublisherPort`, `ProjectRepositoryPort`, `ProgressPublisherPort` — inject vào từng use case struct qua constructor function (`NewHandleStepEventUseCase(repo ProjectRepositoryPort, publisher CommandPublisherPort, progress ProgressPublisherPort) *HandleStepEventUseCase`). Cho phép test độc lập (fake implementations).
- **Constructed directly**: `pgx.Pool`, `amqp091-go` connection/channel (constructed 1 lần ở `main.go` qua `amqp.ConnectionManager`, truyền handle cụ thể vào adapter constructors — không phải abstraction, đây là infrastructure client, không phải business port). **Revision (2026-09-06, ADR-0022)**: `amqp.Publisher`/`amqp.Consumer` không còn nhận thẳng `*amqp.Channel` — nhận `*amqp.ConnectionManager` và gọi `.Channel()` mỗi lần dùng, để sống sót qua reconnect khi broker đóng channel.

## Composition Root
`cmd/orchestrator/main.go`:
1. Load config từ env var (`internal/config`).
2. Kết nối PostgreSQL (`pgx.Pool`), bootstrap schema (`projects`, `saga_steps`, `outbox_events`, `processed_messages`).
3. Kết nối RabbitMQ qua `amqp.ConnectionManager` (`Connect(ctx)` — dial + mở channel lần đầu, fail-fast nếu RabbitMQ không reachable, giống hành vi cũ). **Revision (2026-09-06, ADR-0022)**: `ConnectionManager` khởi động 1 watchdog goroutine nền, tự reconnect với backoff cấp số nhân (`RABBITMQ_RECONNECT_INITIAL_DELAY_MS` mặc định 1s, cap `RABBITMQ_RECONNECT_MAX_DELAY_MS` mặc định 30s) khi connection/channel bị đóng.
4. Khởi tạo `postgres.ProjectRepository`, `postgres.InboxRepository`, `postgres.OutboxRepository`.
5. Khởi tạo `amqp.Publisher` (implement cả `CommandPublisherPort` + `ProgressPublisherPort`), nhận `ConnectionManager` thay vì channel trực tiếp — gọi `.Channel()` mỗi lần publish.
6. Khởi tạo 4 use case (`StartRenderSagaUseCase`, `StartPublishSagaUseCase`, `HandleStepEventUseCase`, `RetryStepUseCase`), inject repository + publisher.
7. Khởi tạo `amqp.Consumer` (nhận `ConnectionManager`), đăng ký queue `orchestrator.events` + 6 DLQ queue, wire `HandleStepEventUseCase`. Đăng ký `ConnectionManager.OnReconnect(...)` gọi lại `consumer.Start(ctx)` sau mỗi lần reconnect — reconnect ở broker làm mất hết consumer đăng ký trước đó, và `Start` gọi lại an toàn (delivery loop cũ đã tự thoát khi channel cũ đóng).
8. Khởi động `postgres.OutboxRelay` như goroutine riêng (background).
9. Khởi tạo `chi` router, wire 4 REST handler tới use case tương ứng.
10. Start HTTP server (`net/http.ListenAndServe`), start AMQP consumer loop (goroutine).

## Wiring Diagram
```
main.go
  ├── postgres.ProjectRepository (implements ProjectRepositoryPort)
  ├── postgres.InboxRepository, OutboxRepository
  ├── amqp.ConnectionManager (owns Connection/Channel + reconnect-with-backoff, ADR-0022)
  │     └── amqp.Publisher (implements CommandPublisherPort + ProgressPublisherPort, ADR-0017) — calls .Channel() per publish
  │     └── amqp.Consumer — registered via OnReconnect to re-subscribe after reconnect
  │
  ├── StartRenderSagaUseCase(repo, publisher) ──┐
  ├── StartPublishSagaUseCase(repo, publisher) ─┤
  ├── HandleStepEventUseCase(repo, publisher, progress) ─┤─── injected into → http.Router (4 REST handlers)
  ├── RetryStepUseCase(repo, publisher) ─────────┘         └── injected into → amqp.Consumer (event handler)
  │
  └── postgres.OutboxRelay(pool, amqp.Publisher) — background goroutine
        └── poll outbox_events chưa publish → publish command qua amqp.Publisher → đánh dấu published_at
```

## Testability
Mọi use case chỉ phụ thuộc interface (`ProjectRepositoryPort`, `CommandPublisherPort`, `ProgressPublisherPort`) — unit test dùng fake implementation (in-memory struct implement interface tương ứng), không cần Postgres/RabbitMQ thật.

## Concurrency Model (Question 7)
Mỗi AMQP event message xử lý trong 1 goroutine riêng (spawn từ consumer loop). Đồng bộ hóa truy cập `projects`/`saga_steps` qua Postgres transaction (row-level locking, không cần mutex tầng ứng dụng). Progress publishing (ADR-0017) không block goroutine xử lý event — publish tới RabbitMQ là fire-and-forget qua channel riêng, không chờ GUI client đọc (Gateway/Unit 9 chịu trách nhiệm buffer/fan-out).
