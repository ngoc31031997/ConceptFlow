# Code Generation Plan — Unit 8: Orchestrator Service

## Unit Context
- **Stories**: C1 (khởi chạy render — orchestration nền), C6 (theo dõi tiến trình — nguồn sự kiện), E3 (đăng video — orchestration nền)
- **Dependencies**: RabbitMQ (Unit 1, topology đã có), 6 service nghiệp vụ (Unit 2–7, publish/consume qua AMQP — không gọi trực tiếp), API Gateway (Unit 9, chưa xây — sẽ proxy REST của Orchestrator)
- **Database entities owned**: `projects`, `saga_steps`, `outbox_events` (commands, ADR-0019), `processed_messages` — database `orchestrator-db` riêng (ADR-0013)
- **Service boundary**: Saga Orchestrator duy nhất — điều phối 2 Saga (Render 5 bước, Publish 1 bước), không chứa business logic từng bước

## Code Location
- **Application code**: `services/orchestrator/` (workspace root — mirror cấu trúc `services/{unit}/` của Unit 2–7)
- **Documentation**: `aidlc-docs/construction/orchestrator-service/code/`

## Step 3.5: Coding Standards (Confirmed từ Low-Level Design — không cần hỏi lại)
- **Naming convention**: Go idiomatic — `PascalCase` cho exported type/function, `camelCase` cho unexported/local, package name lowercase không gạch dưới (module-structure.md đã dùng đúng convention này)
- **SOLID**: Bắt buộc — Dependency Inversion đã thiết kế qua `internal/domain/ports.go` (3 interface), Interface Segregation rõ ràng (`CommandPublisherPort` tách biệt `ProgressPublisherPort` dù cùng 1 struct implement — module-structure.md)
- **Documentation style**: Go doc comments chuẩn (`// FuncName does X`, bắt đầu bằng tên identifier) trên mọi type/function exported; giải thích WHY cho các quyết định non-obvious (vd. tại sao Outbox chứa command — trỏ ADR-0019)
- **Linting/formatting**: `gofmt` (bắt buộc, chuẩn Go) + `go vet`; không có config linting đặc biệt khác trong repo cho Go (đây là unit Go đầu tiên)
- **DI mechanism**: Constructor injection thủ công (dependency-injection.md) — không dùng `wire`/`fx`

## Execution Steps

### Step 1: Project Structure Setup
- [x] Tạo `services/orchestrator/` theo `module-structure.md`: `cmd/orchestrator/`, `internal/domain/`, `internal/application/`, `internal/adapters/{http,amqp,postgres,logging}/`, `internal/config/`
- [x] `go.mod` (module `orchestrator`, Go 1.22), `go.sum` sau khi thêm dependency (`chi`, `pgx/v5`, `amqp091-go`)

**Story**: N/A (infrastructure setup)

### Step 2: Domain Layer Generation
- [x] `internal/domain/project.go` — `Project`, `Scene`, `SagaStep` struct; `ProjectStatus` enum (9 happy-path + 6 `failed_at_<step>`)
- [x] `internal/domain/errors.go` — `ErrProjectNotFound`, `ErrUnexpectedEvent`, `ErrInvalidStatus`, `ErrSagaStepNotFound`
- [x] `internal/domain/ports.go` — `CommandPublisherPort`, `ProjectRepositoryPort`, `ProgressPublisherPort`

**Story**: C1, C6, E3 (domain model nền tảng)

### Step 3: Business Logic Generation (Application Layer)
- [x] `internal/application/start_render_saga.go` — `StartRenderSagaUseCase` (Functional Design Bước 1)
- [x] `internal/application/start_publish_saga.go` — `StartPublishSagaUseCase` (Functional Design Bước 6, validate `ready_to_publish`)
- [x] `internal/application/handle_step_event.go` — `HandleStepEventUseCase` (Functional Design Bước 2–5 + xử lý lỗi + `scene_rendered` progress-only + Rule 1/2/4 validation)
- [x] `internal/application/retry_step.go` — `RetryStepUseCase` (Functional Design Rule 5, tái tạo payload từ `Project`)

**Story**: C1 (Start Render Saga), C6 (Handle Step Event/progress), E3 (Start Publish Saga)

### Step 4: Business Logic Unit Testing
- [x] `internal/application/start_render_saga_test.go`
- [x] `internal/application/start_publish_saga_test.go`
- [x] `internal/application/handle_step_event_test.go` (bao gồm test case Rule 1 mismatch → `failed_at_render_scenes`, Rule 4 unexpected event → skip)
- [x] `internal/application/retry_step_test.go`
- Dùng fake implementations (in-memory) của 3 port — không cần Postgres/RabbitMQ thật (dependency-injection.md's Testability)

**Story**: N/A (test coverage cho Step 3)

### Step 5: Business Logic Summary
- [x] Tạo `aidlc-docs/construction/orchestrator-service/code/business-logic-summary.md`

### Step 6: API Layer Generation (HTTP)
- [x] `internal/adapters/http/router.go` — chi router: `POST /v1/sagas/render`, `POST /v1/sagas/publish`, `GET /v1/projects/{project_id}`, `POST /v1/projects/{project_id}/retry`, `GET /health` (Infrastructure Design)
- [x] `internal/adapters/http/dto.go` — request/response struct (`encoding/json`)

**Story**: C1, C6, E3 (REST surface)

### Step 7: API Layer Unit Testing
- [x] `internal/adapters/http/router_test.go` — `httptest`, fake use case

### Step 8: API Layer Summary
- [x] Tạo `aidlc-docs/construction/orchestrator-service/code/api-layer-summary.md`

### Step 9: Messaging Layer Generation (AMQP)
- [x] `internal/adapters/amqp/envelope.go` — envelope struct + (de)serialization
- [x] `internal/adapters/amqp/consumer.go` — consume `orchestrator.events` (12 loại) + 6 `*.commands.dlq`, spawn goroutine/event
- [x] `internal/adapters/amqp/publisher.go` — implement `CommandPublisherPort` + `ProgressPublisherPort` (publish 6 command + progress `progress.fanout`)

**Story**: C1, C6, E3 (AMQP integration)

### Step 10: Messaging Layer Unit Testing
- [x] `internal/adapters/amqp/envelope_test.go` — serialize/deserialize round-trip

### Step 11: Repository Layer Generation (Postgres)
- [x] `internal/adapters/postgres/db.go` — `pgx.Pool` + schema bootstrap
- [x] `internal/adapters/postgres/project_repository.go` — implement `ProjectRepositoryPort`
- [x] `internal/adapters/postgres/inbox.go` — `InboxRepository` (dedupe)
- [x] `internal/adapters/postgres/outbox.go` — `OutboxRepository` (commands to send, ADR-0019)
- [x] `internal/adapters/postgres/relay.go` — `OutboxRelay` (poll 500ms goroutine, NFR Requirements)

**Story**: C1, C6, E3 (persistence)

### Step 12: Repository Layer Unit Testing
- [x] `internal/adapters/postgres/project_repository_test.go` — chỉ test mapping logic thuần Go không cần DB thật (vd. status enum marshal); test tích hợp thật để lại Build & Test stage

### Step 13: Repository Layer Summary
- [x] Tạo `aidlc-docs/construction/orchestrator-service/code/repository-layer-summary.md`

### Step 14: Supporting Adapters
- [x] `internal/adapters/logging/correlation.go` — `saga_id`/`X-Request-ID` injection vào `log/slog` fields
- [x] `internal/config/config.go` — load env var (`RABBITMQ_URL`, `DATABASE_URL`, `OUTBOX_POLL_INTERVAL_MS`, `DATABASE_MAX_CONNS`)

### Step 15: Composition Root
- [x] `cmd/orchestrator/main.go` — wire toàn bộ theo `dependency-injection.md`'s Composition Root (10 bước)

### Step 16: Database Migration Scripts
- [x] Schema bootstrap SQL (trong `db.go`, `CREATE TABLE IF NOT EXISTS` cho `projects`, `saga_steps`, `outbox_events`, `processed_messages`) — nhất quán cách Unit 2–7 tự bootstrap schema lúc start (không dùng migration tool riêng)

### Step 17: Documentation Generation
- [x] Tạo `services/orchestrator/README.md` (per-unit — overview, prerequisites Go 1.22+, cách chạy `go run`/Docker, env var, cách chạy test `go test ./...`)
- [x] Cập nhật root `README.md` — thêm Orchestrator Service vào Project Structure/service list nếu chưa có

### Step 18: Deployment Artifacts Generation
- [x] `services/orchestrator/Dockerfile` (multi-stage, theo `deployment-architecture.md`)
- [x] Thêm service `orchestrator-db` + `orchestrator` vào root `docker-compose.yml` (theo `deployment-architecture.md`'s reference), khai báo volume `orchestrator_db_data`

---

## Traceability Summary
| Story | Covered by |
|---|---|
| C1 (khởi chạy render) | Step 2, 3 (`StartRenderSagaUseCase`), 6, 9, 11, 15 |
| C6 (theo dõi tiến trình) | Step 2, 3 (`HandleStepEventUseCase`), 6, 9, 11, 15 |
| E3 (đăng video) | Step 2, 3 (`StartPublishSagaUseCase`), 6, 9, 11, 15 |

**Tổng**: 18 bước. Plan này là single source of truth cho Code Generation — không tạo logic ngoài những gì các artifact Functional Design/NFR Design/Infrastructure Design/Low-Level Design đã duyệt.
