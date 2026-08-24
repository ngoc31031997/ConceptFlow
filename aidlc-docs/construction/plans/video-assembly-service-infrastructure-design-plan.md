# Infrastructure Design Plan — Unit 6: Video Assembly Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `infrastructure-design.md`
- [ ] Tạo `deployment-architecture.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Deployment Environment — System Dependencies cho ffmpeg (BẮT BUỘC, đặc thù unit này)
Video Assembly Service cần `ffmpeg` + `ffprobe` (cùng bộ cài đặt) — nhẹ hơn nhiều so với Manim's dependency (không cần `libcairo2`/`libpango`/build tools).

A) 💡 Suggested: Docker container, base `python:3.12-slim` + cài qua `apt-get`: `ffmpeg` (bao gồm `ffprobe`). Không cần thêm system dependency nào khác (không có native extension nào cần compile, khác Unit 5's Manim)
   - ✅ Strengths: image nhỏ gọn, đơn giản hơn đáng kể so với Unit 5's Dockerfile
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Storage Infrastructure — PostgreSQL (ADR-0013)
A) 💡 Suggested: Container riêng `video-assembly-db` (Postgres 16, database-per-service), named volume `video_assembly_db_data` — nhất quán Unit 2/3/4/5
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Storage Infrastructure — Shared Volume (tái sử dụng)
A) 💡 Suggested: Dùng lại named volume `shared_artifacts` đã có — Video Assembly Service đọc `/shared/{project_id}/animations/` (từ Unit 5) + `/shared/{project_id}/audio/` (từ Unit 3), ghi `/shared/{project_id}/video/final.mp4` + `_tmp/`. Không cần volume riêng mới
   - ✅ Strengths: đơn giản, đúng quy ước đã thiết lập từ Unit 3/5 (1 volume dùng chung cho mọi artifact media)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Networking & Health Check
A) 💡 Suggested: Không expose port (không REST, nhất quán TTS/Script Processing/Rendering). Health check qua sentinel file `/tmp/ready` (mirror Unit 3/4/5)
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Resource Limits (Docker Compose `deploy.resources`)
NFR Requirements ghi nhận không giới hạn cứng ở tầng ứng dụng, để Infrastructure Design quyết định ở tầng Docker.

A) 💡 Suggested: KHÔNG set `deploy.resources.limits` trong `docker-compose.yml` ở MVP — nhất quán Unit 5 (Docker Compose không phải Swarm/K8s, `deploy` key thường bị bỏ qua khi chạy `docker compose up` thông thường)
   - ✅ Strengths: đơn giản, đúng công cụ đang dùng
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: Load Balancer / API Gateway / Database Read-Write Splitting/Sharding
A) 💡 Suggested: Tất cả **N/A** — 1 instance cố định, không REST endpoint, Postgres chỉ chứa Outbox/Inbox
   - ✅ Strengths: đúng bản chất unit
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 7: Scaling Configuration
A) 💡 Suggested: 1 instance cố định, không auto-scaling — nhất quán toàn hệ thống
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 8: Monitoring Infrastructure
A) 💡 Suggested: Structured logging ra stdout, bao gồm `saga_id` trong mọi log line — nhất quán
   - ✅ Strengths: đủ cho MVP local
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A