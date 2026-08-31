# NFR Requirements Plan — Unit 9: API Gateway

## Unit Context
Reverse-proxy/gateway thuần túy, stateless (trừ in-memory SSE connection map), forward REST tới 3 service + SSE fan-out từ RabbitMQ. Node.js/Express theo ADR-0009.

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `nfr-requirements.md`
- [x] Tạo `tech-stack-decisions.md`
- [x] Tạo ADR cho lựa chọn ngôn ngữ/framework
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Tech Stack — Framework Selection trong Node.js (BẮT BUỘC)
`technology-direction.md`/ADR-0009 đã chốt Node.js, nhưng chưa chọn cụ thể Express hay Fastify.

A) 💡 Suggested: **Express 4.x** — framework phổ biến nhất cho reverse-proxy pattern, middleware ecosystem lớn (dù Gateway chỉ cần rất ít middleware — CORS, correlation), tài liệu/cộng đồng rộng, đủ hiệu năng cho quy mô 1 Creator. Fastify nhanh hơn ở benchmark cao tải nhưng lợi thế đó không có ý nghĩa ở quy mô dự án này
   - ✅ Strengths: đơn giản, phổ biến nhất cho pattern "gateway/proxy" — nhiều ví dụ tham khảo, đủ nhanh cho use case cá nhân
   - ⚠️ Trade-offs: Fastify có throughput cao hơn — không quan trọng ở quy mô này

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Resilience Pattern — Downstream Timeout
A) 💡 Suggested: Timeout cố định 30s cho mọi request proxy (`httpClient`'s `fetch` với `AbortController`) — đủ cho REST request thông thường (`GET /plugins`, `POST /v1/sagas/render` chỉ khởi tạo Saga rồi trả ngay, không chờ toàn bộ pipeline). Timeout → Gateway trả `502` (Flow 4, LLD). Không retry tự động (nhất quán triết lý "không auto-retry" của Unit 8 — để GUI/Creator quyết định thử lại)
   - ✅ Strengths: đơn giản, timeout đủ dài cho REST request (không phải Saga hoàn chỉnh — REST chỉ khởi tạo)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Scalability — Concurrent SSE Connections
A) 💡 Suggested: Không giới hạn cứng số SSE connection đồng thời — quy mô 1 Creator, tối đa vài project theo dõi song song (tương ứng vài chục connection tối đa). Node.js event loop xử lý hàng nghìn connection đồng thời dễ dàng ở quy mô này, không cần connection pooling/limit
   - ✅ Strengths: đơn giản, đúng quy mô, Node.js's I/O model tự nhiên phù hợp nhiều connection đồng thời nhẹ
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Availability & Reliability — AMQP Consumer Reconnect
A) 💡 Suggested: Nếu Gateway mất kết nối RabbitMQ (progress consumer), tự động reconnect với backoff cố định (retry mỗi 5s, không giới hạn số lần thử) — dùng thư viện `amqplib`'s connection error event để trigger reconnect logic. Trong lúc mất kết nối, SSE client vẫn giữ kết nối mở nhưng không nhận progress update mới (không phải lỗi nghiêm trọng — GUI có thể fallback polling `GET /v1/projects/{id}` nếu cần, đã có sẵn)
   - ✅ Strengths: đơn giản, tự phục hồi khi RabbitMQ tạm gián đoạn (Docker Compose restart), không cần circuit breaker phức tạp
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Security Pattern
A) 💡 Suggested: Không auth/rate-limit cho GUI ↔ Gateway (threat model single-user local, nhất quán Unit 7/8's NFR Requirements). CORS: cho phép origin của Web GUI (Unit 10, chưa xây — sẽ xác định `WEB_GUI_ORIGIN` env var khi Unit 10 build) nếu GUI chạy port riêng khác Gateway; nếu Unit 10 serve qua cùng Gateway (chưa quyết định, để Unit 10's Infrastructure Design xác nhận) thì không cần CORS
   - ✅ Strengths: đơn giản, đúng threat model, để ngỏ quyết định CORS cụ thể cho Unit 10 xác nhận khi có đủ thông tin
   - ⚠️ Trade-offs: có thể cần điều chỉnh nhỏ khi Unit 10 xác định kiến trúc serving cụ thể (không phải rework lớn — chỉ thêm CORS middleware nếu cần)

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Maintainability — Structured Logging
A) 💡 Suggested: JSON structured logging qua thư viện `pino` (nhanh, phổ biến cho Node.js/Express, JSON-native) — mỗi log line: `timestamp`, `level`, `message`, `requestId?`, `projectId?` (SSE context). Nhất quán format JSON logging của các unit khác
   - ✅ Strengths: chuẩn cộng đồng Node.js cho structured logging hiệu năng cao, JSON nhất quán hệ thống
   - ⚠️ Trade-offs: thêm 1 dependency nhỏ (`pino`) — chấp nhận được, thay thế `console.log` không có cấu trúc

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7: Messaging & Event Participation
A) 💡 Suggested: Gateway CHỈ consume `progress.fanout` (không publish gì tới RabbitMQ, không tham gia Saga) — vai trò thuần túy là AMQP-to-SSE bridge. Delivery guarantee: at-most-once chấp nhận được cho progress (khác các unit khác cần at-least-once cho command/event nghiệp vụ) — nếu Gateway mất 1 progress message do disconnect tạm thời, GUI vẫn thấy trạng thái cuối cùng khi `GET /v1/projects/{id}` hoặc khi progress tiếp theo đến, không cần dedupe/replay phức tạp
   - ✅ Strengths: đúng bản chất — progress message chỉ mang tính thông báo UX, không phải nguồn sự thật duy nhất (source of truth là `Project.Status` ở Orchestrator, luôn query được qua GET)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8: Distributed Transaction Participation
A) 💡 Suggested: KHÔNG áp dụng — Gateway không tham gia Saga (không phải orchestrator, không phải participant), chỉ là entry point REST + progress bridge. Không có compensating action nào ở tầng Gateway
   - ✅ Strengths: đúng bản chất, rõ ràng ranh giới trách nhiệm
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
