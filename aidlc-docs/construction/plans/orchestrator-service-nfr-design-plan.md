# NFR Design Plan — Unit 8: Orchestrator Service

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `nfr-design-patterns.md`
- [x] Tạo `logical-components.md`
- [x] Tạo `messaging-design.md`
- [x] Tạo ADR cho các pattern quyết định ở stage này
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: CRUD vs CQRS (BẮT BUỘC)
A) 💡 Suggested: CRUD đơn giản trên `projects`/`saga_steps` (Domain Entities, Functional Design) + `outbox_events`/`processed_messages` (Inbox/Outbox, LLD Question 6). Không phải CQRS — `GET /v1/projects/{id}` đọc trực tiếp cùng bảng `projects` mà `HandleStepEventUseCase` ghi, không có nhu cầu tách read/write model hay query phức tạp
   - ✅ Strengths: nhất quán các unit khác, đơn giản, đúng quy mô 1 Creator/máy dev cá nhân
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Resilience Pattern
A) 💡 Suggested: Không retry nội bộ tự động cho bất kỳ command nào — khi 1 bước lỗi (`*_failed` event hoặc DLQ), Orchestrator chuyển `failed_at_<step>` ngay (Functional Design Rule 8) và chờ Creator chủ động gọi `POST /v1/projects/{id}/retry` (retry-by-step, không phải automatic backoff-retry). Lý do: retry tự động 1 bước tốn tài nguyên (render/synthesize speech) có thể lặp lại lỗi giống hệt nếu nguyên nhân không phải transient (vd. lỗi Manim template) — để Creator xem lỗi cụ thể qua SSE trước khi quyết định retry là phù hợp hơn cho use case giáo dục nội dung (không phải hệ thống tự phục hồi high-availability)
   - ✅ Strengths: đơn giản, đúng bản chất UX (Creator cần biết lỗi gì trước khi retry, không phải luôn luôn retry mù quáng), nhất quán retry-by-step đã thiết kế ở LLD/Functional Design
   - ⚠️ Trade-offs: lỗi transient (vd. RabbitMQ tạm gián đoạn) cũng cần Creator bấm retry thủ công thay vì tự phục hồi — chấp nhận được ở quy mô single-user

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Idempotency Pattern
A) 💡 Suggested: 2 tầng — (1) Message-level: Inbox dedupe `message_id` cho MỌI event nhận vào (12 loại), đảm bảo `HandleStepEventUseCase` không chạy 2 lần cho cùng 1 event bị redeliver. (2) Step-level: `SagaStep.status` guard (Functional Design Rule 4) — event chỉ được xử lý nếu bước tương ứng đang `in_progress`, chặn double-processing nếu 2 event cho cùng bước đến (vd. do 1 service lỗi publish 2 lần). KHÔNG cần artifact-level idempotency riêng ở Orchestrator (đó là trách nhiệm từng service nghiệp vụ, vd. Rendering Service's skip-scene-đã-render)
   - ✅ Strengths: đơn giản, đúng phân tầng trách nhiệm (Orchestrator lo idempotency ở tầng điều phối, không lo idempotency ở tầng artifact)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Saga Pattern / Event-Driven Design / Inbox-Outbox Pattern
A) 💡 Suggested: Xác nhận lại — Saga: **Orchestrator** (central coordinator, orchestration-based per ADR-0007), không phải choreography. Event-Driven: consume 12 loại event (6 success + 6 failure) + `scene_rendered` progress; publish 6 loại command + progress message (`progress.fanout`, ADR-0017). Inbox/Outbox: PostgreSQL-backed, NHƯNG khác các unit khác — Outbox ở đây dùng để gửi COMMAND (không phải event, vì Orchestrator là người khởi phát action chứ không chỉ báo cáo kết quả), Inbox dùng để dedupe EVENT nhận vào (LLD Question 6, Module Structure). `OutboxRelay` poll `outbox_events` mỗi 500ms (NFR Requirements Question 3)
   - ✅ Strengths: nhất quán ADR-0007/LLD, ghi nhận rõ điểm khác biệt Outbox semantic so với các unit khác (quan trọng để tránh nhầm lẫn khi code generation)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Security Pattern
A) 💡 Suggested: Không auth/rate-limit riêng cho REST hay AMQP nội bộ (threat model single-user local, nhất quán NFR Requirements Question 5). Validate input ở REST layer (zero-trust cơ bản — vd. `project_id` tồn tại, `visibility` đúng enum) trước khi tạo/update `Project`, KHÔNG cần validate lại dữ liệu từ AMQP event (các service nghiệp vụ đã validate nội bộ trước khi publish event, tin tưởng lẫn nhau trong hệ thống nội bộ)
   - ✅ Strengths: đơn giản, đúng threat model, tránh double-validation không cần thiết
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Logical Components — Infrastructure Elements
A) 💡 Suggested: Không cần circuit breaker/cache/rate-limiter (đã loại trừ ở Question 2/5 và NFR Requirements). Thành phần hạ tầng cần thiết: (1) Postgres (projects, saga_steps, outbox_events, processed_messages — cùng 1 database instance, `orchestrator-db`), (2) RabbitMQ connection (consumer cho 12 event queue + 6 DLQ, publisher cho 6 command + progress fanout), (3) `OutboxRelay` background goroutine (poll 500ms), (4) HTTP server (chi router, 4 endpoint)
   - ✅ Strengths: tối thiểu cần thiết, nhất quán LLD's module-structure.md
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
