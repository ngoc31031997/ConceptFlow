# NFR Design Plan — Unit 9: API Gateway

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `nfr-design-patterns.md`
- [x] Tạo `logical-components.md`
- [x] Tạo `messaging-design.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: CRUD vs CQRS (BẮT BUỘC)
A) 💡 Suggested: Không áp dụng CRUD/CQRS — Gateway không có data store riêng, không đọc/ghi database nào (Question 8, NFR Requirements: stateless trừ in-memory SSE map). Đây không phải câu hỏi có ý nghĩa cho 1 pure proxy layer
   - ✅ Strengths: đúng bản chất, không ép khái niệm không áp dụng
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Resilience Pattern
A) 💡 Suggested: Xác nhận lại — timeout cố định 30s cho proxy request (NFR Requirements Question 2), không auto-retry, trả `502` khi downstream không kết nối được (LLD Flow 4). AMQP: auto-reconnect backoff 5s cố định (NFR Requirements Question 4), không giới hạn số lần thử vì đây là kết nối nội bộ Docker network (không có lý do "give up" vĩnh viễn)
   - ✅ Strengths: nhất quán NFR Requirements/LLD, đơn giản
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Idempotency Pattern
A) 💡 Suggested: Không cần — Gateway không tạo state nghiệp vụ nào (không ghi database), chỉ forward request nguyên trạng. Idempotency của các thao tác nghiệp vụ (vd. `POST /v1/sagas/render` tạo saga_id mới mỗi lần) là trách nhiệm của Orchestrator (Unit 8), không phải Gateway
   - ✅ Strengths: đúng phân tầng trách nhiệm — Gateway không nên tự quyết định request nào "trùng lặp" vì không có ngữ cảnh nghiệp vụ
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Event-Driven Design / Saga Pattern / Inbox-Outbox
A) 💡 Suggested: Xác nhận lại — Gateway CHỈ consume `progress.fanout` (AMQP-to-SSE bridge), không publish gì, không tham gia Saga (NFR Requirements Question 7/8). KHÔNG cần Inbox/Outbox pattern — Gateway không ghi database, không cần đảm bảo "exactly send once" cho bất kỳ message nào (nó không gửi message nghiệp vụ nào cả, chỉ REST proxy + AMQP consume thuần túy)
   - ✅ Strengths: đúng bản chất — Inbox/Outbox chỉ cần thiết khi có state cần đồng bộ với message gửi/nhận, Gateway không có state đó
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Security Pattern
A) 💡 Suggested: Xác nhận lại — không auth/rate-limit nội bộ (threat model single-user). Validate tối thiểu ở tầng Gateway: `project_id` là URL path param hợp lệ (không chứa ký tự nguy hiểm — path traversal defense cơ bản dù nội bộ), không validate sâu hơn body (trách nhiệm downstream)
   - ✅ Strengths: đơn giản, defense-in-depth tối thiểu hợp lý (path param sanitize) mà không trùng lặp validation của downstream
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Logical Components — Infrastructure Elements
A) 💡 Suggested: Không cần cache/circuit-breaker/rate-limiter (đã loại trừ). Thành phần cần thiết: (1) Express HTTP server, (2) `httpClient` × 3 (Orchestrator/ContentPlugin/Publisher), (3) AMQP consumer (`progress.fanout`, exclusive queue), (4) in-memory SSE connection registry (`Map<project_id, Response[]>`) — KHÔNG cần Postgres/Redis
   - ✅ Strengths: tối thiểu cần thiết, nhất quán LLD's module-structure.md
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
