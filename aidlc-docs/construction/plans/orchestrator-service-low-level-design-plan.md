# Low-Level Design Plan — Unit 8: Orchestrator Service

## Unit Context
- **Responsibility**: Saga coordinator duy nhất (ADR-0007) — điều phối 2 Saga (Render Pipeline: 5 bước, Publish: 1 bước, xem `services.md`), quản lý state machine của video project, kích hoạt compensating action khi 1 bước lỗi
- **Architectural style**: Hexagonal/Ports & Adapters (ADR-0002) — áp dụng theo idiom Go (package theo layer, không phải class-based); **Go** (ADR-0009 — unit đầu tiên KHÔNG dùng Python, "Saga coordinator concurrency-heavy" hưởng lợi từ goroutine/channel)
- **Interfaces**: REST `POST /v1/sagas/render`, `POST /v1/sagas/publish`, `GET /v1/projects/{project_id}` (gọi bởi Gateway, Unit 9 — CHƯA build) + AMQP: publish 6 command tới 6 service nghiệp vụ, consume 12 event loại (6 success + 6 failure) từ `orchestrator.events`
- **Depends on**: Unit 1 (RabbitMQ) + Unit 2, 3, 4, 5, 6, 7 (tất cả 6 service nghiệp vụ, đã hoàn tất — cần để test luồng Saga end-to-end)

## Saga Step Definitions (theo `services.md`, nguồn xác thực — component-methods.md có phần lỗi thời trước ADR-0014, không dùng làm nguồn chính)

| # | Step | Command → Queue | Success Event | Failure Event |
|---|---|---|---|---|
| 1 | Parse Script | `parse_script` → `script_processing.commands` | `script_parsed` | `parse_failed` |
| 2 | Classify Scenes | `classify_scenes` → `content_plugin.commands` | `scenes_classified` | `classification_failed` |
| 3 | Synthesize Speech | `synthesize_speech` → `tts.commands` | `speech_synthesized` | `synthesis_failed` |
| 4 | Render Scenes | `render_scenes` → `rendering.commands` | `rendering_completed` (+ progress `scene_rendered`) | `rendering_failed` |
| 5 | Assemble Video | `assemble_video` → `video_assembly.commands` | `video_assembled` | `assembly_failed` |
| 6 | Publish Video | `publish_video` → `publisher.commands` | `video_published` | `publish_failed` |

Saga Render Pipeline = bước 1-5. Saga Publish = bước 6 (Saga riêng, khởi tạo độc lập sau khi Render Pipeline xong).

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `module-structure.md`
- [ ] Tạo `dependency-injection.md`
- [ ] Tạo `interface-contracts.md`
- [ ] Tạo `sequence-flows.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Layering & Package Structure (BẮT BUỘC, đặc thù Go)
Go không có class/interface implement ngầm định như Python's ABC — Hexagonal áp dụng qua package boundary + Go interface (structural typing).

A) 💡 Suggested: `internal/domain/` (structs thuần Go: `Project`, `SagaInstance`, `SagaStep`; interfaces: `CommandPublisherPort`, `ProjectRepositoryPort` — không import RabbitMQ/Postgres/HTTP cụ thể) → `internal/application/` (`SagaOrchestrator` struct: `StartRenderSaga()`, `StartPublishSaga()`, `HandleStepEvent()` — điều phối domain + port) → `internal/adapters/` (`amqp/` consumer+publisher dùng `amqp091-go`, `http/` REST handler dùng chuẩn `net/http`+`chi` router, `postgres/` implement `ProjectRepositoryPort` dùng `pgx`) → `cmd/orchestrator/main.go` (composition root)
   - ✅ Strengths: khớp Go idiom chuẩn (`internal/` cho code không export ra ngoài module, `cmd/` cho entrypoint), vẫn giữ đúng tinh thần Hexagonal (dependency chỉ đi vào trong)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 2: Dependency Injection (BẮT BUỘC, đặc thù Go)
Go không có DI container/framework phổ biến như FastAPI's `Depends()`.

A) 💡 Suggested: Constructor injection thủ công (Go idiom chuẩn) — `NewSagaOrchestrator(publisher CommandPublisherPort, repo ProjectRepositoryPort) *SagaOrchestrator`, wire cụ thể ở `cmd/orchestrator/main.go`. Không dùng DI framework (vd. `wire`, `fx`) — không cần thiết ở quy mô 1 service
   - ✅ Strengths: đơn giản, đúng Go idiom, nhất quán tinh thần constructor injection đã dùng ở mọi unit Python trước
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 3: State Machine — States & Persistence (BẮT BUỘC — trung tâm của unit này)
`components.md` liệt kê state machine sơ bộ nhưng lỗi thời (thứ tự trước ADR-0014). Cần định nghĩa lại theo đúng 6 bước Saga hiện tại.

A) 💡 Suggested: States: `draft` → `parsing_script` → `classifying_scenes` → `synthesizing_speech` → `rendering` → `assembling` → `ready_to_publish` → `publishing` → `published`; và `failed_at_<step>` (vd. `failed_at_render_scenes`) khi 1 bước lỗi. Lưu trong Postgres (`orchestrator-db`, database-per-service ADR-0013) — bảng `projects` (project_id, status, script_content, plugin_id, voice_language, scenes JSONB, video_path, youtube metadata, created_at, updated_at) + bảng `saga_steps` (saga_id, step_name, status: pending/in_progress/completed/failed, started_at, completed_at, error_message) để trace lịch sử từng bước cho mục đích debug/audit
   - ✅ Strengths: state machine rõ ràng khớp đúng thứ tự Saga hiện tại, `saga_steps` cho phép GUI (Story C6) hiển thị tiến trình chi tiết và Orchestrator biết chính xác bước nào cần retry
   - ⚠️ Trade-offs: 2 bảng thay vì 1 — nhưng cần thiết vì `projects` là aggregate hiện tại, `saga_steps` là lịch sử/audit trail

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 4: Progress Event Delivery to Gateway (BẮT BUỘC — quyết định liên-unit quan trọng, ràng buộc thiết kế cho Unit 9)
`component-methods.md`: Gateway's `GET /projects/{id}/events` (SSE) "forward event nhận được từ Orchestrator Service". Unit 9 (Gateway) chưa build — cần quyết định CƠ CHẾ Orchestrator đẩy tiến trình sang Gateway.

A) 💡 Suggested: Orchestrator TỰ expose SSE endpoint `GET /v1/projects/{project_id}/events` (dùng Go's `net/http` streaming response, `text/event-stream`) — Gateway (Unit 9) chỉ cần reverse-proxy luồng SSE này tới GUI (không cần Gateway tự quản lý fan-out event). Orchestrator publish vào in-memory channel per-project khi nhận event Saga, SSE handler subscribe channel đó
   - ✅ Strengths: đơn giản nhất — Gateway chỉ cần proxy, không cần tự implement pub/sub; đúng vai trò Gateway "chỉ còn routing" (`components.md`)
   - ⚠️ Trade-offs: nếu Gateway restart, mất kết nối SSE hiện tại (client tự reconnect — chấp nhận được, chuẩn hành vi SSE)

B) Orchestrator publish progress qua RabbitMQ vào 1 exchange riêng (`progress.fanout`), Gateway tự subscribe và quản lý SSE fan-out tới nhiều GUI client
   - ✅ Strengths: tách rời hoàn toàn Orchestrator khỏi việc quản lý HTTP connection dài hạn (SSE), đúng triết lý message-driven toàn hệ thống
   - ⚠️ Trade-offs: phức tạp hơn đáng kể (Gateway cần thêm AMQP consumer + SSE fan-out logic), Gateway hiện đã "chỉ còn routing" theo `components.md` — thêm trách nhiệm này lệch khỏi định nghĩa ban đầu

C) Other (please describe after [Answer]: tag below)

[Answer]:B

### Question 5: Saga Instance Tracking & Event Correlation
Orchestrator consume 12 loại event từ `orchestrator.events` (1 queue chung, theo `logical-components.md` của RabbitMQ Infrastructure). Cần biết event nào thuộc saga/project nào và bước nào đang chờ.

A) 💡 Suggested: Mỗi event envelope có `saga_id`+`project_id` (đã thiết kế từ Unit 1). Khi nhận event, Orchestrator tra `saga_steps` (Question 3) theo `saga_id`+`step_name` (suy ra từ `event_type` — vd. `scenes_classified` → step "classify_scenes") để xác nhận đây có phải bước đang `in_progress` hay không (chống xử lý event không mong đợi/trùng lặp). Cập nhật `saga_steps` + `projects.status`, sau đó dispatch command cho bước tiếp theo (nếu còn) hoặc kết thúc Saga
   - ✅ Strengths: dùng đúng dữ liệu đã có sẵn trong envelope, không cần cơ chế tra cứu phức tạp
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 6: Idempotency at Orchestrator Level
Orchestrator vừa là consumer (nhận 12 loại event) vừa là publisher (gửi 6 loại command) — cần cơ chế idempotency riêng gì không, khác các service nghiệp vụ (vốn dùng Inbox/Outbox, ADR-0013)?

A) 💡 Suggested: Áp dụng ĐÚNG Inbox/Outbox pattern (ADR-0013) như mọi unit khác — Inbox dedupe `message_id` của event nhận được (tránh xử lý trùng khi RabbitMQ requeue), Outbox cho command gửi đi (đảm bảo command được gửi đúng 1 lần dù Orchestrator crash giữa chừng cập nhật state và gửi command). Bảng `outbox_events`(dùng để gửi COMMAND thay vì event — khác các unit khác 1 chút về semantic nhưng cùng cơ chế kỹ thuật)/`processed_messages` trong `orchestrator-db`
   - ✅ Strengths: nhất quán ADR-0013 toàn hệ thống, đảm bảo command không bị gửi trùng hoặc mất khi Orchestrator crash
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 7: Concurrency Model (Go-specific, lý do chọn Go theo ADR-0009)
A) 💡 Suggested: Mỗi AMQP event message được xử lý trong 1 goroutine riêng (spawn từ consumer loop), đồng bộ hóa truy cập `projects`/`saga_steps` qua transaction Postgres (không cần mutex ở tầng ứng dụng — Postgres's row-level locking đã đủ để tránh race condition khi 2 event của cùng 1 project tới gần như đồng thời, vd. `scene_rendered` progress events dồn dập). SSE handler (Question 4) dùng buffered channel per-project, không block goroutine xử lý event nếu GUI client chậm đọc
   - ✅ Strengths: tận dụng đúng goroutine/channel — lý do chọn Go (ADR-0009), Postgres transaction đủ mạnh để tránh cần thêm cơ chế lock riêng
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 8: Compensating Action & Retry API
`services.md` đã xác nhận: không cần rollback thực sự (idempotent theo `project_id`+`scene_index`/`project_id`), chỉ cần retry đúng bước lỗi.

A) 💡 Suggested: Thêm REST endpoint `POST /v1/projects/{project_id}/retry` — Orchestrator đọc `projects.status` hiện tại (`failed_at_<step>`), gửi lại command của đúng bước đó (dùng LẠI dữ liệu đã lưu ở `projects` — vd. `scenes` đã parse từ trước — không yêu cầu Creator nhập lại từ đầu, khớp Story C6/E3's AC). `message_id` MỚI cho command retry (để không bị Inbox của service đích chặn — các service đã thiết kế artifact-level idempotency riêng để không làm lại phần đã xong)
   - ✅ Strengths: đúng khớp compensating action đã duyệt ở Inception, tái sử dụng dữ liệu đã lưu thay vì bắt Creator nhập lại
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 9: API Versioning & Correlation ID
A) 💡 Suggested: URI versioning `/v1/...` (nhất quán ADR-0008). Correlation ID: `saga_id` cho luồng AMQP (gắn vào mọi log line); REST từ Gateway dùng `X-Request-ID` (nhất quán Unit 2/7) — Gateway tạo `X-Request-ID` mới cho mỗi request, Orchestrator tạo `saga_id` MỚI khi khởi tạo Saga (không phải Gateway tạo)
   - ✅ Strengths: nhất quán hệ thống
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 10: HTTP Router/Framework (Go-specific)
A) 💡 Suggested: `net/http` chuẩn thư viện + `chi` router (nhẹ, idiomatic, hỗ trợ tốt path params như `{project_id}` và middleware chain) — KHÔNG dùng framework nặng như Gin/Echo (không cần thiết cho ~4 endpoint REST)
   - ✅ Strengths: tối giản, đúng triết lý Go "prefer standard library", `chi` là lựa chọn phổ biến/nhẹ cho router bổ sung path params
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a
