# Infrastructure Design Plan — Unit 5: Rendering Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `infrastructure-design.md`
- [ ] Tạo `deployment-architecture.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Deployment Environment — System Dependencies cho Manim (BẮT BUỘC, đặc thù unit này)
Manim cần nhiều system dependency hơn các unit khác: `ffmpeg` (ghép frame thành video), `libcairo2`/`libpango` (render text/vector graphics). LaTeX (cho công thức toán) KHÔNG cần ở MVP (chỉ minh họa code/thuật toán/khái niệm lập trình, không có công thức toán phức tạp).

A) 💡 Suggested: Docker container, base `python:3.12-slim` + cài qua `apt-get`: `ffmpeg`, `libcairo2-dev`, `libpango1.0-dev`, `pkg-config`, build tools cần thiết cho Manim's native extensions (build từ `pip install manim` sẽ compile 1 số phần). KHÔNG cài LaTeX (`texlive`) — giảm đáng kể kích thước image (`texlive` có thể nặng hàng GB), chấp nhận Manim's `Tex`/`MathTex` mobject sẽ lỗi nếu ai đó dùng (không dùng ở MVP vì chỉ có 2 template hiện tại, không cần công thức toán)
   - ✅ Strengths: image nhỏ hơn đáng kể so với cài đủ LaTeX, đủ cho 2 template MVP (`algorithm_visualization`, `concept_illustration` — không cần công thức toán)
   - ⚠️ Trade-offs: nếu tương lai cần template dùng công thức toán (vd. minh họa complexity analysis O(n²)), phải bổ sung LaTeX vào Dockerfile — chấp nhận được, chưa cần ở MVP

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Storage Infrastructure — PostgreSQL (ADR-0013)
A) 💡 Suggested: Container riêng `rendering-db` (Postgres 16, database-per-service), named volume `rendering_db_data` — nhất quán Unit 2/3/4
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Storage Infrastructure — Shared Volume (tái sử dụng)
A) 💡 Suggested: Dùng lại named volume `shared_artifacts` đã có (từ TTS Service) — Rendering Service ghi vào `/shared/{project_id}/animations/`, TTS Service ghi vào `/shared/{project_id}/audio/` — cùng volume, khác thư mục con. Không cần volume riêng mới
   - ✅ Strengths: đơn giản, đúng quy ước đã thiết lập từ Unit 3 (1 volume dùng chung cho mọi artifact media)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Networking & Health Check
A) 💡 Suggested: Không expose port (không REST, nhất quán TTS/Script Processing). Health check qua sentinel file `/tmp/ready` (mirror TTS/Script Processing)
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Resource Limits (Docker Compose `deploy.resources`)
NFR Requirements ghi nhận không giới hạn cứng ở tầng ứng dụng, để Infrastructure Design quyết định ở tầng Docker.

A) 💡 Suggested: KHÔNG set `deploy.resources.limits` trong `docker-compose.yml` ở MVP — để container dùng tài nguyên máy dev tự do (Docker Compose không phải Swarm/K8s, `deploy` key thường bị bỏ qua khi chạy `docker compose up` thông thường, chỉ có tác dụng với Swarm). Nếu cần giới hạn thực sự, dùng `mem_limit`/`cpus` (Compose v2 hỗ trợ) — nhưng chưa cần ở MVP máy dev cá nhân
   - ✅ Strengths: đơn giản, đúng công cụ đang dùng (`docker compose`, không phải Swarm)
   - ⚠️ Trade-offs: nếu máy dev yếu, Manim render có thể chiếm hết tài nguyên tạm thời — chấp nhận được, người dùng có thể set `mem_limit` thủ công nếu cần

B) Other (please describe after [Answer]: tag below)

[Answer]:
A
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
