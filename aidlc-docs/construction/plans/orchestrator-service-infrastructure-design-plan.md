# Infrastructure Design Plan — Unit 8: Orchestrator Service

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `infrastructure-design.md`
- [x] Tạo `deployment-architecture.md`
- [x] Tạo ADR nếu có quyết định hạ tầng đáng kể (N/A — mọi quyết định đều mirror pattern đã có ADR: ADR-0013 database-per-service, ADR-0009 Go stack; không có quyết định mới đủ "costly-to-reverse")
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Deployment Environment
A) 💡 Suggested: Docker container, base `golang:1.22-alpine` (multi-stage build — build stage compile binary tĩnh, final stage `alpine:3.19` chỉ chứa binary, không cần Go toolchain runtime). Cùng docker network `backend`
   - ✅ Strengths: image final nhỏ gọn (Go static binary), nhất quán network với các unit khác
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Storage Infrastructure — PostgreSQL (ADR-0013)
A) 💡 Suggested: Container riêng `orchestrator-db` (Postgres 16, database-per-service), named volume `orchestrator_db_data` — nhất quán Unit 2–7. Chứa `projects`, `saga_steps`, `outbox_events` (commands, ADR-0019), `processed_messages`
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Storage Infrastructure — Shared Volume
Orchestrator KHÔNG tạo/đọc artifact file nào (animation, audio, video) — chỉ điều phối qua dữ liệu trong Postgres + message payload.

A) 💡 Suggested: KHÔNG mount `shared_artifacts` volume vào Orchestrator — không có nhu cầu (path đến artifact chỉ là string lưu trong `Project.scenes`/`Project.video_path`, không phải Orchestrator tự đọc/ghi file)
   - ✅ Strengths: đúng ranh giới trách nhiệm, giảm bề mặt truy cập không cần thiết
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Networking & Health Check
A) 💡 Suggested: Port 8000 (chi HTTP server) chỉ nội bộ docker network `backend`, KHÔNG map ra host — Creator truy cập qua API Gateway (Unit 9) proxy, mirror Unit 7/Publisher. Health check `GET /health` (thêm route mới vào `router.go`, kiểm tra Postgres + không cần kiểm tra RabbitMQ liveness riêng vì AMQP consumer tự log lỗi khi mất kết nối)
   - ✅ Strengths: nhất quán kiến trúc Gateway-proxy, không expose port không cần thiết
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Messaging Infrastructure
A) 💡 Suggested: Kết nối `rabbitmq:5672` (container nội bộ, Unit 1's topology) — không cần thêm exchange/queue mới ngoài những gì Unit 1 đã định nghĩa (`commands.direct`, `orchestrator.events`, `progress.fanout`, 6 `*.commands.dlq`). Consumer prefetch=1 (nhất quán các unit khác, dù Orchestrator xử lý event trong goroutine riêng — prefetch kiểm soát số message "in-flight" chưa ack, không phải số goroutine)
   - ✅ Strengths: tận dụng hạ tầng đã có, không cần thay đổi Unit 1
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Database Read/Write Splitting
A) 💡 Suggested: Single primary, KHÔNG read replica — tải đọc (`GET /v1/projects/{id}`) rất thấp (1 Creator, vài request/phút lúc theo dõi tiến trình qua GUI, phần lớn cập nhật tiến trình đến qua SSE không phải polling REST), không có lý do kỹ thuật cho read replica ở quy mô này
   - ✅ Strengths: đơn giản, đúng quy mô
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7: Database Sharding/Partitioning
A) 💡 Suggested: KHÔNG cần sharding — quy mô 1 Creator, dữ liệu (`projects`/`saga_steps`) tăng tuyến tính rất chậm (mỗi video 1 project), không có write throughput hay dataset size nào tiệm cận giới hạn 1 Postgres instance đơn (NFR Requirements không đặt mục tiêu scale nào vượt mức này)
   - ✅ Strengths: đúng bản chất — sharding sẽ over-engineer nghiêm trọng cho use case cá nhân
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8: Load Balancer
A) 💡 Suggested: KHÔNG áp dụng — chỉ 1 instance Orchestrator chạy (Scaling Configuration: Fixed, 1 instance, nhất quán tất cả unit khác), không cần load balancer
   - ✅ Strengths: đúng bản chất, tránh phức tạp không cần thiết
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 9: API Gateway (Xác nhận quyết định từ integration-boundaries.md)
A) 💡 Suggested: API Gateway (Unit 9, chưa xây) proxy `POST /v1/sagas/render`, `POST /v1/sagas/publish`, `GET /v1/projects/{id}`, `POST /v1/projects/{id}/retry` tới `orchestrator:8000` — routing rule cụ thể thiết kế ở Unit 9's Infrastructure Design (mirror cách Unit 7 ghi nhận). Gateway KHÔNG enforce auth/rate-limit (nhất quán threat model single-user)
   - ✅ Strengths: nhất quán `integration-boundaries.md` (API Gateway ↔ Orchestrator Service: Synchronous REST/HTTP+JSON)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 10: Monitoring Infrastructure
A) 💡 Suggested: Không có monitoring stack riêng (Prometheus/Grafana...) — nhất quán các unit khác (Docker Compose `docker-compose logs` + structured JSON logging, NFR Requirements Question 6, đủ cho use case cá nhân/local). RabbitMQ Management UI (`localhost:15672`, đã có từ Unit 1) đủ để quan sát queue/DLQ depth khi debug
   - ✅ Strengths: đơn giản, đúng quy mô, tận dụng công cụ đã có
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
