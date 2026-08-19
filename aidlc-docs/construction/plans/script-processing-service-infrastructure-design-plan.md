# Infrastructure Design Plan — Unit 4: Script Processing Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `infrastructure-design.md`
- [ ] Tạo `deployment-architecture.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Deployment Environment
A) 💡 Suggested: Docker container, base `python:3.12-slim`, build trong `docker-compose.yml` root, cùng docker network `backend` — nhất quán toàn hệ thống. Không cần system dependency đặc biệt nào (không như TTS Service cần Piper)
   - ✅ Strengths: nhất quán, đơn giản
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Storage Infrastructure — PostgreSQL (ADR-0013)
A) 💡 Suggested: Container riêng `script-processing-db` (Postgres 16, database-per-service), named volume `script_processing_db_data` — giống hệt `content-plugin-db`/`tts-db`
   - ✅ Strengths: nhất quán, database-per-service đúng ADR-0013
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Networking
A) 💡 Suggested: Không expose port nào (không có REST/HTTP, giống TTS Service sau retrofit) — chỉ kết nối outbound tới RabbitMQ + PostgreSQL
   - ✅ Strengths: nhất quán, đơn giản
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Health Check
A) 💡 Suggested: Sentinel file `/tmp/ready` (giống TTS Service sau retrofit) — ghi sau khi AMQP consumer + `OutboxRelay` khởi động thành công. Docker `healthcheck` kiểm tra file này (`test -f /tmp/ready`)
   - ✅ Strengths: nhất quán, không cần thư viện HTTP
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Load Balancer / API Gateway / Database Read-Write Splitting/Sharding
A) 💡 Suggested: Tất cả **N/A** — 1 instance cố định, không REST endpoint, Postgres chỉ chứa Outbox/Inbox (quy mô nhỏ, không cần splitting/sharding)
   - ✅ Strengths: đúng bản chất unit
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: Scaling Configuration
A) 💡 Suggested: 1 instance cố định, không auto-scaling — nhất quán toàn hệ thống
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 7: Monitoring Infrastructure
A) 💡 Suggested: Structured logging ra stdout, bao gồm `saga_id` (từ envelope AMQP) trong mọi log line — nhất quán
   - ✅ Strengths: đủ cho MVP local
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
