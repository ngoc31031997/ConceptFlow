# NFR Requirements Plan — Unit 2: Content Plugin Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-requirements.md`
- [ ] Tạo `tech-stack-decisions.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Tech Stack Consistency (BẮT BUỘC)
`technology-direction.md` (HLD) đã chọn Python/FastAPI cho mọi backend service (ADR-0003). Xác nhận Unit 2 dùng đúng hướng này, không cần polyglot deviation?

A) Xác nhận — Python 3.12 + FastAPI, khớp `technology-direction.md`, không có lý do kỹ thuật nào để lệch hướng cho unit này (không có workload đặc thù cần ngôn ngữ khác)

B) Other (please describe after [Answer]: tag below)

[Answer]: A (xác nhận Python 3.12 + FastAPI cho Unit 2 — xem ADR-0009 cho quyết định polyglot có chọn lọc áp dụng ở các unit khác: Go cho Orchestrator, Node.js cho API Gateway)

### Question 2: Performance Requirements
`GET /v1/plugins` và xử lý `classify_scenes` — có yêu cầu response time cụ thể không?

A) 💡 Suggested: Không có SLA cứng (theo NFR2 hệ thống — không yêu cầu tốc độ), nhưng vì đây là pure in-memory lookup/validation (không gọi I/O ngoài), response time thực tế sẽ ở mức mili-giây — không cần tối ưu đặc biệt
   - ✅ Strengths: khớp NFR2, không cần thêm công sức tối ưu không cần thiết
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Availability Requirements
Unit 2 là dependency của Unit 4 (Script Processing) và Orchestrator. Nếu Unit 2 down, điều gì xảy ra?

A) 💡 Suggested: Chấp nhận unavailability tạm thời (không cần multi-instance/failover) — nếu Content Plugin Service down, message `classify_scenes` bị requeue bởi RabbitMQ (theo `nfr-requirements.md` Unit 1: at-least-once, retry 3 lần) và tự xử lý khi service khởi động lại; không cần HA (High Availability) phức tạp ở quy mô 1 người dùng
   - ✅ Strengths: đơn giản, tận dụng cơ chế retry đã có sẵn ở tầng RabbitMQ, không cần thêm hạ tầng HA
   - ⚠️ Trade-offs: có độ trễ nếu service down đúng lúc — chấp nhận được vì không có SLA uptime

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Security Requirements
Security Baseline extension đã tắt. Có yêu cầu bảo mật tối thiểu nào riêng cho unit này không (vd. validate input tránh injection dù chỉ nội bộ)?

A) 💡 Suggested: Validate input cơ bản qua Pydantic schema (FastAPI mặc định) để tránh lỗi runtime từ dữ liệu sai định dạng — không cần thêm biện pháp bảo mật đặc biệt (auth/rate-limit) vì chỉ có Gateway/Orchestrator nội bộ gọi tới, không expose public

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Messaging & Event Participation
Đã xác định ở Application Design/Low-Level Design: Unit 2 là consumer của `classify_scenes` command và producer của `scenes_classified`/`classification_failed` event. Xác nhận delivery guarantee kế thừa từ Unit 1 (at-least-once) áp dụng cho unit này, không cần điều chỉnh riêng?

A) Đúng — kế thừa at-least-once từ `nfr-requirements.md` (Unit 1), idempotency đã thiết kế ở Rule 5 (Functional Design) xử lý việc này

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Distributed Transaction Participation (Saga)
Xác nhận vai trò của Unit 2 trong Saga: là **participant** (thực hiện 1 bước, không phải coordinator), compensating action là gì nếu bước "Classify Scenes" cần rollback?

A) 💡 Suggested: Unit 2 là Saga participant cho bước "Classify Scenes". Vì đây là pure computation (không ghi dữ liệu bền vững, không có side-effect ngoài publish event), **không cần compensating action thực sự** — nếu bước sau (Render Scenes) thất bại, không cần "undo" việc classify vì nó không tạo ra artifact nào cần dọn dẹp. Retry đơn giản là gọi lại `classify_scenes`
   - ✅ Strengths: đơn giản nhất có thể — tận dụng tính chất "không side-effect" của bước này
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
