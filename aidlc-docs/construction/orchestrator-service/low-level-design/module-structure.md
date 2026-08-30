# Module Structure — Unit 8: Orchestrator Service

## Layering (Hexagonal / Ports & Adapters — ADR-0002, applied via Go package boundaries)

```
services/orchestrator/
├── cmd/
│   └── orchestrator/
│       └── main.go              # Composition root — wiring, HTTP server + AMQP consumer + OutboxRelay startup
├── internal/
│   ├── domain/
│   │   ├── project.go            # Project, SagaStep structs; ProjectStatus enum (Question 3)
│   │   ├── errors.go              # ErrProjectNotFound, ErrUnexpectedEvent, ErrInvalidStatus
│   │   └── ports.go               # CommandPublisherPort, ProjectRepositoryPort, ProgressPublisherPort
│   ├── application/
│   │   ├── start_render_saga.go   # StartRenderSagaUseCase
│   │   ├── start_publish_saga.go  # StartPublishSagaUseCase
│   │   ├── handle_step_event.go   # HandleStepEventUseCase (Question 5)
│   │   └── retry_step.go          # RetryStepUseCase (Question 8)
│   ├── adapters/
│   │   ├── http/
│   │   │   ├── router.go           # chi router, /v1/sagas/{render,publish}, /v1/projects/{project_id}, /v1/projects/{project_id}/retry
│   │   │   └── dto.go              # request/response structs (Go's stdlib encoding/json, no separate schema library)
│   │   ├── amqp/
│   │   │   ├── consumer.go         # consumes orchestrator.events (12 event types) + *.dlq queues
│   │   │   ├── publisher.go        # publishes 6 command types + progress.fanout messages (ADR-0017)
│   │   │   └── envelope.go         # envelope struct + (de)serialization (Go equivalent of Python's producer.py build_envelope)
│   │   ├── postgres/
│   │   │   ├── db.go                # pgx pool + schema bootstrap (ADR-0013, extends with projects/saga_steps tables)
│   │   │   ├── inbox.go             # InboxRepository (dedupe incoming event message_id)
│   │   │   ├── outbox.go            # OutboxRepository (queues outgoing commands, NOT events — Question 6)
│   │   │   ├── relay.go             # OutboxRelay — polls outbox_events, publishes commands via amqp.Publisher
│   │   │   └── project_repository.go # implements ProjectRepositoryPort — projects + saga_steps CRUD
│   │   └── logging/
│   │       └── correlation.go       # saga_id (AMQP) / X-Request-ID (REST) injection into structured log fields
│   └── config/
│       └── config.go               # env var loading (RABBITMQ_URL, DATABASE_URL, etc.)
└── tests/                          # Go convention: *_test.go colocated with source, not a separate tests/ tree
```

**Note**: Đi ngược quy ước `tests/` riêng của các unit Python — Go idiom chuẩn là đặt file `*_test.go` cùng thư mục với file nguồn (vd. `internal/application/handle_step_event_test.go` cạnh `handle_step_event.go`). Code Generation sẽ tuân theo quy ước này thay vì tạo thư mục `tests/` riêng.

## Dependency Direction
`adapters/` → `application/` → `domain/`. `domain/` không import `pgx`/`amqp091-go`/`chi`. `application/` chỉ phụ thuộc `domain/` qua interface (`CommandPublisherPort`, `ProjectRepositoryPort`, `ProgressPublisherPort`) — không biết chi tiết Postgres/RabbitMQ cụ thể nào. `adapters/postgres/project_repository.go` implement `domain/ports.go::ProjectRepositoryPort`; `adapters/amqp/publisher.go` implement cả `CommandPublisherPort` VÀ `ProgressPublisherPort` (2 interface riêng biệt dù cùng 1 struct implement — Interface Segregation, SOLID).

## Module Responsibilities

| Module | Responsibility |
|---|---|
| `internal/domain/project.go` | `Project` (project_id, status, script_content, plugin_id, voice_language, scenes []Scene, video_path, youtube metadata), `SagaStep` (saga_id, step_name, status, error_message), `ProjectStatus` enum (Question 3's 9 states + `failed_at_<step>`) |
| `internal/domain/ports.go` | `CommandPublisherPort` (`PublishCommand(ctx, queue string, envelope Envelope) error`), `ProjectRepositoryPort` (`Get`, `Save`, `UpdateStatus`, `GetStep`, `UpdateStep`), `ProgressPublisherPort` (`PublishProgress(ctx, ProgressMessage) error` — ADR-0017) |
| `internal/application/start_render_saga.go` | `StartRenderSagaUseCase` — tạo `Project` mới (status=`draft`), tạo `SagaStep` cho bước 1 (`parse_script`, status=`in_progress`), gọi `CommandPublisherPort.PublishCommand` (qua Outbox, Question 6) |
| `internal/application/start_publish_saga.go` | `StartPublishSagaUseCase` — validate project ở status `ready_to_publish`, tạo `SagaStep` cho bước 6 (`publish_video`), publish command |
| `internal/application/handle_step_event.go` | `HandleStepEventUseCase` — nhận event đã parse, tra `SagaStep` theo `saga_id`+suy ra `step_name` từ `event_type` (Question 5), validate đang `in_progress` (chống event không mong đợi), cập nhật `SagaStep`+`Project.Status`, dispatch command bước tiếp theo hoặc kết thúc Saga; publish progress message (ADR-0017) sau mỗi lần xử lý |
| `internal/application/retry_step.go` | `RetryStepUseCase` — đọc `Project.Status` (`failed_at_<step>`), tái tạo command payload từ dữ liệu đã lưu ở `Project`, publish lại với `message_id` mới (Question 8) |
| `internal/adapters/http/router.go` | REST handlers: `POST /v1/sagas/render` → `StartRenderSagaUseCase`, `POST /v1/sagas/publish` → `StartPublishSagaUseCase`, `GET /v1/projects/{project_id}` → `ProjectRepositoryPort.Get`, `POST /v1/projects/{project_id}/retry` → `RetryStepUseCase` |
| `internal/adapters/amqp/consumer.go` | Consume `orchestrator.events` (12 event types) + 6 DLQ queues (1 consumer chung theo pattern, Unit 1's note) → parse envelope → `HandleStepEventUseCase` |
| `internal/adapters/amqp/publisher.go` | Implement `CommandPublisherPort` (publish tới `commands.direct` với routing key tương ứng service) VÀ `ProgressPublisherPort` (publish tới `progress.fanout`, ADR-0017) |
| `internal/adapters/postgres/outbox.go` | Outbox cho COMMAND gửi đi (khác semantic các unit khác dùng Outbox cho event, nhưng cùng cơ chế kỹ thuật — Question 6) |
| `internal/adapters/postgres/project_repository.go` | Implement `ProjectRepositoryPort` — CRUD trên bảng `projects`+`saga_steps` |
