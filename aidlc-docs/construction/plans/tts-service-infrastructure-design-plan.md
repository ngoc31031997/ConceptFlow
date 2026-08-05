# Infrastructure Design Plan — Unit 3: TTS Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `infrastructure-design.md`
- [ ] Tạo `deployment-architecture.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Deployment Environment
Nhất quán với Unit 2 (Docker container, docker-compose)?

A) 💡 Suggested: Docker container, base image `python:3.12-slim`, build trong `docker-compose.yml` root, cùng docker network `backend`
   - ✅ Strengths: nhất quán toàn hệ thống
   - ⚠️ Trade-offs: `python:3.12-slim` có thể thiếu system dependency mà Piper cần (vd. `libsndfile`, build tools cho package Python compile native) — cần cài thêm trong Dockerfile qua `apt-get`

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Storage Infrastructure — Shared Volume (BẮT BUỘC, lần đầu áp dụng)
Đây là unit đầu tiên dùng Shared Docker Volume (Low-Level Design Question 5) cho artifact audio. Cần xác định volume cụ thể trong `docker-compose.yml`.

A) 💡 Suggested: Named volume `shared_artifacts` (Docker managed volume, không phải bind mount) mount vào `/shared` trong container TTS Service. Volume này sẽ được mount tương tự vào Rendering Service (Unit 5) khi phát triển unit đó, để đọc lại audio_path. Dùng named volume (không phải bind mount ra thư mục host) để tránh vấn đề permission/path khác nhau giữa máy dev khác nhau
   - ✅ Strengths: nhất quán giữa các service dùng chung volume, không phụ thuộc cấu trúc thư mục host cụ thể
   - ⚠️ Trade-offs: khó truy cập trực tiếp từ host để debug (cần `docker exec` hoặc `docker cp`) — chấp nhận được cho MVP

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Compute Infrastructure — Voice Model Bundling (Low-Level Design Question 4)
LLD đã quyết định bundle voice model file trong Docker image tại build stage. Cần xác nhận cơ chế cụ thể.

A) 💡 Suggested: Dockerfile tải voice model Piper (`.onnx` + config `.onnx.json`) cho `vi` và `en` từ Piper's official model repository (Hugging Face) trong build stage bằng `RUN curl`, lưu vào thư mục cố định trong image (vd. `/app/voices/`). Build sẽ cần network access lúc build image (không phải lúc chạy container) — chấp nhận được vì build chỉ chạy 1 lần trên máy dev
   - ✅ Strengths: image tự chứa đầy đủ, không phụ thuộc network lúc runtime (đúng NFR Requirements)
   - ⚠️ Trade-offs: build image lần đầu cần network + tăng thời gian build (~model size); nếu URL model thay đổi, cần cập nhật Dockerfile

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Networking
Port expose cho TTS Service?

A) 💡 Suggested: Port 8000 (FastAPI, nhất quán với Unit 2), chỉ nội bộ docker network `backend`, không map ra host (chỉ Rendering Service gọi tới, không cần từ ngoài)
   - ✅ Strengths: nhất quán, đủ bảo mật cho môi trường local
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Health Check
A) 💡 Suggested: `GET /health` (custom endpoint đơn giản — trả 200 nếu FastAPI app đã sẵn sàng VÀ voice model đã load thành công vào cache; trả 503 nếu model chưa load xong) + `healthcheck` trong docker-compose, tương tự Unit 2. Các service phụ thuộc TTS Service (Rendering Service, khi phát triển) dùng `depends_on: condition: service_healthy`
   - ✅ Strengths: đảm bảo Rendering Service không gọi TTS Service trước khi model sẵn sàng (tránh lỗi lúc mới khởi động)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: Load Balancer / API Gateway / Database Read-Write Splitting/Sharding
A) 💡 Suggested: Tất cả **N/A** — 1 instance cố định (không cần Load Balancer), không expose qua API Gateway (chỉ Rendering Service gọi nội bộ), không có database (không cần read/write splitting/sharding)
   - ✅ Strengths: đúng bản chất unit
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 7: Scaling Configuration
A) 💡 Suggested: 1 instance cố định, không auto-scaling (nhất quán toàn hệ thống, local Docker single-machine)
   - ✅ Strengths: nhất quán, đúng quy mô MVP
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 8: Monitoring Infrastructure
A) 💡 Suggested: Structured logging ra stdout (đọc qua `docker logs`), bao gồm `saga_id` (từ `X-Saga-ID` header) trong mọi log line — nhất quán với cách tiếp cận đơn giản của Unit 2 (không cần APM/metrics platform riêng cho MVP local)
   - ✅ Strengths: đủ cho nhu cầu debug MVP, không thêm hạ tầng phức tạp
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
