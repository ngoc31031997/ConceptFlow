# Logical Components — Unit 8: Orchestrator Service

## Component Diagram (logical, technology-agnostic)

```
┌─────────────────────────────────────────────────────────────┐
│                     Orchestrator Service                      │
│                                                                 │
│  ┌───────────────┐        ┌────────────────────────────┐     │
│  │  HTTP Server   │        │   AMQP Consumer             │     │
│  │  (4 endpoints) │        │   (12 event types + 6 DLQ)  │     │
│  └───────┬────────┘        └──────────┬───────────────────┘   │
│          │                            │                        │
│          ▼                            ▼                        │
│  ┌──────────────────────────────────────────────────┐         │
│  │              Application / Use Cases               │         │
│  │  StartRenderSagaUseCase, StartPublishSagaUseCase,  │         │
│  │  HandleStepEventUseCase, RetryStepUseCase           │         │
│  └───────┬─────────────────────────────┬──────────────┘         │
│          │                             │                        │
│          ▼                             ▼                        │
│  ┌───────────────┐          ┌─────────────────────┐            │
│  │ Project Repo   │          │  Inbox Repo          │            │
│  │ (projects,     │          │  (processed_messages)│            │
│  │  saga_steps)   │          └─────────────────────┘            │
│  └───────┬────────┘                                             │
│          │                  ┌─────────────────────┐            │
│          │                  │  Outbox Repo         │            │
│          │                  │  (outbox_events)      │            │
│          │                  └──────────┬────────────┘            │
│          │                             │                        │
│          ▼                             ▼                        │
│  ┌────────────────────────────────────────────┐                │
│  │            PostgreSQL (orchestrator-db)      │                │
│  └────────────────────────────────────────────┘                │
│                                                                 │
│  ┌──────────────────────────┐                                  │
│  │  OutboxRelay (goroutine)  │──── poll 500ms ──▶  publish      │
│  └──────────────────────────┘                     command       │
└─────────────────────────────┬───────────────────────────────────┘
                               │
                        ┌──────▼──────┐
                        │  RabbitMQ    │
                        │ commands.direct, orchestrator.events,
                        │ progress.fanout, *.commands.dlq
                        └──────────────┘
```

## Components

| Component | Type | Responsibility |
|---|---|---|
| HTTP Server (chi router) | Adapter, inbound | `POST /v1/sagas/render`, `POST /v1/sagas/publish`, `GET /v1/projects/{id}`, `POST /v1/projects/{id}/retry` |
| AMQP Consumer | Adapter, inbound | Consume `orchestrator.events` (12 event types) + `*.commands.dlq` (6 queues); parse envelope; invoke use case per goroutine |
| Application/Use Cases | Application layer | Business logic orchestration — no infra dependency (see LLD Dependency Direction) |
| Project Repository | Adapter, outbound (Postgres) | CRUD `projects` + `saga_steps` |
| Inbox Repository | Adapter, outbound (Postgres) | Dedupe `message_id` của event nhận vào |
| Outbox Repository | Adapter, outbound (Postgres) | Queue command để gửi (semantic: "commands to send", khác Unit khác) |
| OutboxRelay | Background process (goroutine) | Poll `outbox_events` mỗi 500ms, publish qua AMQP Publisher |
| AMQP Publisher | Adapter, outbound | Publish 6 loại command (`commands.direct`) + progress message (`progress.fanout`, ADR-0017) |
| PostgreSQL (`orchestrator-db`) | Data store | `projects`, `saga_steps`, `outbox_events`, `processed_messages` — 1 database instance |

## Infrastructure Elements — Không áp dụng
- **Cache**: không cần (NFR Requirements — không có endpoint đọc nặng/external API call cần cache).
- **Circuit Breaker**: không cần (không có dependency đồng bộ ngoài Postgres/RabbitMQ nội bộ).
- **Rate Limiter**: không cần (threat model single-user local).
- **Read replica / CQRS store**: không áp dụng (CRUD, xem `nfr-design-patterns.md`).

## Scaling Boundaries
- Consumer: 1 goroutine/event, không giới hạn cứng số goroutine đồng thời — giới hạn tự nhiên đến từ Postgres connection pool (`max_conns=10`, env `DATABASE_MAX_CONNS`).
- HTTP server: xử lý đồng thời theo mô hình mặc định của `net/http`/chi (goroutine-per-request), không cần cấu hình riêng ở quy mô này.
