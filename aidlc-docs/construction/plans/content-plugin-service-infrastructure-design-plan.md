# Infrastructure Design Plan — Unit 2: Content Plugin Service

## Not Applicable
- **Database Read/Write Splitting/Sharding**: N/A — không có database
- **Load Balancer**: N/A — 1 instance
- **API Gateway**: N/A trong phạm vi unit này (unit này là backend, không phải gateway)

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `infrastructure-design.md`
- [ ] Tạo `deployment-architecture.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Deployment Environment & Dockerfile
Unit này cần Dockerfile riêng (không dùng image có sẵn như Unit 1). Base image nào?

A) 💡 Suggested: `python:3.12-slim` — nhẹ, đủ dùng cho FastAPI service không cần compiler nặng (khác với Rendering Service sẽ cần thêm LaTeX/ffmpeg sau này)
   - ✅ Strengths: image nhỏ, build nhanh, đủ cho nhu cầu hiện tại
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Networking
Port nào cần expose?

A) 💡 Suggested: Port 8000 (FastAPI mặc định) chỉ nội bộ docker network `backend` (giống RabbitMQ) — không map ra host vì chỉ Gateway gọi tới, không cần Creator truy cập trực tiếp
   - ✅ Strengths: giảm bề mặt expose không cần thiết
   - ⚠️ Trade-offs: không thể `curl localhost:8000` trực tiếp từ host để test nhanh — chấp nhận được, có thể `docker exec` hoặc dùng Swagger UI qua Gateway proxy sau này

B) Expose ra host để tiện test trực tiếp trong quá trình phát triển

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Health Check
Cần healthcheck cho docker-compose không?

A) 💡 Suggested: Có — FastAPI endpoint `GET /health` (trả 200 nếu registry đã discover xong plugin) + `healthcheck` trong docker-compose dùng `curl` hoặc Python script gọi endpoint này; Orchestrator Service (Unit 8, sau này) dùng `depends_on: condition: service_healthy`
   - ✅ Strengths: đảm bảo service chỉ được coi là "sẵn sàng" sau khi plugin discovery hoàn tất
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Scaling Configuration
Xác nhận: 1 instance cố định, không auto-scaling (khớp NFR Requirements)?

A) Đúng

B) Other (please describe after [Answer]: tag below)

[Answer]: A
