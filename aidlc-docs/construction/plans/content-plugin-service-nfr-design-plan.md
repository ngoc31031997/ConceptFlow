# NFR Design Plan — Unit 2: Content Plugin Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-design-patterns.md`
- [ ] Tạo `logical-components.md`
- [ ] Tạo `messaging-design.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: CRUD vs CQRS (BẮT BUỘC)
Unit 2 không có database (Question 6, Low-Level Design: stateless, registry in-memory). Xác nhận CQRS không áp dụng?

A) Xác nhận — N/A, không có data model nghiệp vụ cần lưu trữ

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Idempotency Store
`business-rules.md` (Rule 5) yêu cầu dedupe theo `message_id`. Lưu ở đâu?

A) 💡 Suggested: In-memory `set[message_id]` với TTL 24h (khớp message TTL của Unit 1), tự dọn định kỳ. Chấp nhận mất dedupe state khi service restart — vì bước "Classify Scenes" không có side-effect bền vững (theo NFR Requirements: không cần compensating action), xử lý trùng sau restart chỉ gây publish event trùng, và Orchestrator Service (Unit 8) cũng có idempotency riêng ở tầng của nó nên vẫn an toàn
   - ✅ Strengths: đơn giản nhất, không cần thêm database/cache ngoài (Redis) cho 1 unit không có state quan trọng
   - ⚠️ Trade-offs: mất dedupe khi restart — chấp nhận được vì hệ quả tối đa chỉ là 1 event bị publish trùng, không gây sai lệch dữ liệu

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Resilience Pattern
Unit 2 chỉ phụ thuộc RabbitMQ (Unit 1). Có cần circuit breaker hay retry logic riêng khi kết nối RabbitMQ gián đoạn không?

A) 💡 Suggested: Dùng cơ chế reconnect tự động có sẵn của `aio-pika` (built-in connection recovery) — không cần tự viết circuit breaker riêng, vì chỉ có 1 dependency hạ tầng (không phải multi-service call chain cần cô lập lỗi lẫn nhau)
   - ✅ Strengths: tận dụng thư viện có sẵn, không tự tạo thêm cơ chế phức tạp không cần thiết
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
