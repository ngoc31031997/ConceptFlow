# NFR Requirements Plan — Unit 8: Orchestrator Service

## Unit Context
Saga coordinator, hybrid REST (khởi tạo Saga, GET status, retry) + message-driven (12 event types consume, 6 command + progress publish), Postgres Inbox/Outbox (Question 6, LLD) + `projects`/`saga_steps` tables. Go/chi/pgx/amqp091-go đã dùng xuyên suốt Low-Level Design (module-structure.md).

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `nfr-requirements.md`
- [x] Tạo `tech-stack-decisions.md`
- [x] Tạo ADR cho lựa chọn ngôn ngữ/framework
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Tech Stack Consistency (BẮT BUỘC)
`technology-direction.md` (System-wide) chỉ định rõ: **Go cho Orchestrator Service** (ADR-0009, ngoại lệ polyglot duy nhất cùng Node.js/Gateway) — vì vai trò Saga coordinator concurrency-heavy phù hợp goroutine hơn Python/FastAPI dùng ở các unit khác. Low-Level Design (Unit 8) đã build toàn bộ module structure trên nền tảng này.

A) 💡 Suggested: Xác nhận **Go 1.22+** + **chi** (HTTP router, nhẹ, idiomatic — LLD đã dùng ở `adapters/http/router.go`) + **pgx** (Postgres driver hiệu năng cao, hỗ trợ connection pool native — LLD đã dùng ở `adapters/postgres/db.go`) + **amqp091-go** (RabbitMQ client chuẩn cộng đồng Go — LLD đã dùng ở `adapters/amqp/`)
   - ✅ Strengths: nhất quán 100% với Low-Level Design đã duyệt (commit `c60a06a`), đúng system-wide direction (ADR-0009), goroutine tự nhiên phù hợp việc consume song song 12 loại event
   - ⚠️ Trade-offs: là ngôn ngữ duy nhất khác Python trong hệ thống (trừ Gateway/Node.js) — nhưng đây là quyết định đã chốt từ Inception (ADR-0009), không phải quyết định mới ở Unit 8

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Scalability — Concurrent Saga Throughput
Business rule (Functional Design, Rule 6) xác nhận không giới hạn số project/Saga đồng thời ở tầng business logic. Ở tầng NFR/hạ tầng, cần giới hạn nào không (connection pool, goroutine, consumer concurrency)?

A) 💡 Suggested: AMQP consumer xử lý mỗi event trong 1 goroutine riêng (LLD's sequence-flows.md, `[goroutine]` annotation) — không giới hạn cứng số goroutine đồng thời (khối lượng tự nhiên nhỏ: 1 Creator, vài project cùng lúc trên máy dev). Postgres connection pool (pgx) giới hạn ở mức mặc định hợp lý cho máy dev cá nhân (vd. `max_conns=10`, đọc từ env var `DATABASE_MAX_CONNS` nếu cần override)
   - ✅ Strengths: đơn giản, đủ cho quy mô 1 Creator/máy dev cá nhân (nhất quán các unit khác), tránh over-engineering cho tải không tồn tại
   - ⚠️ Trade-offs: không có (nếu cần scale multi-tenant sau này, đây là điểm cần review lại — ngoài phạm vi hiện tại)

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Performance — Command Dispatch Latency
`OutboxRelay` poll `outbox_events` định kỳ để publish command (Flow 1, LLD) thay vì publish ngay lập tức trong transaction. Chu kỳ poll bao lâu là chấp nhận được?

A) 💡 Suggested: Poll mỗi **500ms** (đọc từ env var `OUTBOX_POLL_INTERVAL_MS`, default 500) — độ trễ tối đa ~0.5s giữa lúc command được enqueue và lúc thực sự publish, không đáng kể so với thời gian xử lý thực tế của các bước Saga (render/synthesize speech mất hàng giây đến hàng phút)
   - ✅ Strengths: đơn giản, độ trễ không ảnh hưởng trải nghiệm Creator (không phải hệ thống real-time), nhất quán mô hình Outbox Relay polling phổ biến
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Availability & Reliability — Consumer Crash Recovery
Nếu Orchestrator crash giữa lúc xử lý event (trước khi ack), điều gì đảm bảo event không bị mất?

A) 💡 Suggested: Dựa hoàn toàn vào RabbitMQ's at-least-once delivery (Unit 1) — message chưa ack sẽ được redeliver khi consumer reconnect (RabbitMQ tự động requeue message của consumer bị mất kết nối). Inbox (`processed_messages`, dedupe theo `message_id`) đảm bảo idempotency khi message được xử lý lại. KHÔNG cần thêm cơ chế health-check/circuit-breaker phức tạp — Docker Compose's `restart: unless-stopped` (nhất quán các unit khác) đủ cho máy dev cá nhân
   - ✅ Strengths: đơn giản, tận dụng đúng cơ chế RabbitMQ + Inbox đã thiết kế ở LLD, không over-engineer cho môi trường single-node local
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Security — Internal Service Trust Boundary
Orchestrator nhận REST request từ API Gateway (Unit 9, chưa xây) và AMQP message từ 6 service nghiệp vụ. Có cần xác thực/authorization giữa các service nội bộ không?

A) 💡 Suggested: KHÔNG cần auth giữa các service nội bộ (Gateway ↔ Orchestrator, Orchestrator ↔ service nghiệp vụ) — toàn bộ chạy trong 1 Docker network cô lập trên máy cá nhân (không expose ra internet, trừ port GUI/Gateway), nhất quán quyết định NFR đã áp dụng cho các unit trước (Security Baseline extension = No, theo `aidlc-state.md`). RabbitMQ dùng credential mặc định trong `.env` (không phải secret production-grade)
   - ✅ Strengths: đơn giản, đúng phạm vi dự án cá nhân/local, nhất quán extension configuration đã tắt Security Baseline
   - ⚠️ Trade-offs: không phù hợp nếu deploy production/multi-user sau này — ngoài phạm vi hiện tại

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Maintainability — Structured Logging & Correlation
LLD đã có `adapters/logging/correlation.go` cho `saga_id`/`X-Request-ID`. Format log cụ thể là gì?

A) 💡 Suggested: JSON structured logging (thư viện `log/slog`, Go 1.21+ standard library — không cần dependency ngoài) — mỗi log line gồm `timestamp`, `level`, `message`, `saga_id?`, `project_id?`, `step?` (fields tùy ngữ cảnh). Nhất quán format JSON logging của các unit Python khác (dễ đọc/tổng hợp qua `docker-compose logs`)
   - ✅ Strengths: dùng standard library (không thêm dependency), đơn giản, format nhất quán hệ thống
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7: Messaging — Delivery Guarantee (Đã xác nhận ở LLD, xác nhận lại chính thức)
A) 💡 Suggested: At-least-once cho cả command (Outbox) và event (Inbox dedupe) — đã xác nhận ở LLD's interface-contracts.md Question 6. Không cần exactly-once (chi phí triển khai cao, không cần thiết cho quy mô 1 Creator/máy dev cá nhân)
   - ✅ Strengths: nhất quán LLD, đủ cho yêu cầu thực tế
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8: Distributed Transaction Participation — Saga Orchestrator Role (Đã xác nhận ở Functional Design, xác nhận lại chính thức)
A) 💡 Suggested: Orchestrator Service là **Saga Orchestrator** duy nhất trong hệ thống (không phải participant) — sở hữu toàn bộ định nghĩa thứ tự bước, state machine, và compensating action (retry-by-step, KHÔNG rollback — Functional Design Rule 8). 6 service nghiệp vụ còn lại là Saga participants (choreography step, chỉ biết publish event kết quả, không biết bước tiếp theo)
   - ✅ Strengths: nhất quán ADR-0007 (Orchestration-based Saga) và toàn bộ Functional Design/LLD đã duyệt
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
