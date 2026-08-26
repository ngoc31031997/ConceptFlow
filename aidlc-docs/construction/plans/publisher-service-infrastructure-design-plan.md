# Infrastructure Design Plan — Unit 7: Publisher Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `infrastructure-design.md`
- [ ] Tạo `deployment-architecture.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Deployment Environment
A) 💡 Suggested: Docker container, base `python:3.12-slim` — không cần system dependency đặc biệt nào (không có native extension, khác Unit 5/6). Cùng docker network `backend`
   - ✅ Strengths: đơn giản, image nhỏ gọn
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 2: Storage Infrastructure — PostgreSQL (ADR-0013)
A) 💡 Suggested: Container riêng `publisher-db` (Postgres 16, database-per-service), named volume `publisher_db_data` — nhất quán Unit 2/3/4/5/6. Chứa cả Outbox/Inbox VÀ `oauth_credentials`
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 3: Storage Infrastructure — Shared Volume (tái sử dụng, read-only)
A) 💡 Suggested: Dùng lại named volume `shared_artifacts` đã có — Publisher Service CHỈ ĐỌC `/shared/{project_id}/video/final.mp4` (từ Unit 6), không ghi gì vào shared volume. Mount như read-only (`:ro`) trong docker-compose để rõ ràng về ý định (Publisher Service không tạo file nào trên shared volume)
   - ✅ Strengths: đúng quy ước đã thiết lập, mount `:ro` làm rõ ranh giới trách nhiệm (defense against accidental writes)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 4: Networking & Health Check
A) 💡 Suggested: Port 8000 (FastAPI) chỉ nội bộ docker network `backend`, KHÔNG map ra host (khác Unit 2 — vì OAuth callback được Gateway proxy tới, Creator không truy cập trực tiếp Publisher Service, mirror cách Gateway proxy `/plugins` tới Content Plugin Service ở Unit 2). Health check `GET /health` (mirror Unit 2)
   - ✅ Strengths: nhất quán kiến trúc Gateway-proxy đã thiết lập (ADR-0004/ADR-0005), không expose port không cần thiết ra host
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 5: OAuth Credentials Configuration (BẮT BUỘC — đặc thù unit này)
`google-auth-oauthlib` cần Google OAuth Client ID/Secret (đăng ký trên Google Cloud Console) để xây dựng authorization flow.

A) 💡 Suggested: Đọc từ biến môi trường `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URI` (mặc định trỏ tới Gateway's public URL cho `/v1/auth/youtube/callback`) — thêm vào `.env.example` (không commit giá trị thật, nhất quán `RABBITMQ_USER`/`POSTGRES_USER` pattern đã có ở root README). Creator tự đăng ký OAuth Client trên Google Cloud Console theo hướng dẫn trong README (ngoài phạm vi tự động hóa của hệ thống — đây là bước setup 1 lần, thủ công, giống việc lấy API key cho bất kỳ dịch vụ bên thứ 3 nào)
   - ✅ Strengths: không hardcode secret, nhất quán pattern biến môi trường đã dùng cho `RABBITMQ_USER`/`POSTGRES_USER`
   - ⚠️ Trade-offs: Creator cần tự thực hiện bước đăng ký OAuth Client thủ công trước khi dùng tính năng Publish — cần tài liệu hướng dẫn rõ ràng ở README (Code Generation sẽ bổ sung)

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 6: Resource Limits (Docker Compose `deploy.resources`)
A) 💡 Suggested: KHÔNG set `deploy.resources.limits` ở MVP — nhất quán Unit 5/6 (Docker Compose không phải Swarm/K8s)
   - ✅ Strengths: đơn giản, đúng công cụ đang dùng
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 7: Load Balancer / API Gateway / Database Read-Write Splitting/Sharding
A) 💡 Suggested: Load Balancer/Sharding: N/A — 1 instance cố định, Postgres chỉ chứa Outbox/Inbox + 1 row credential. API Gateway: CÓ liên quan — Publisher Service's REST endpoint (`/v1/auth/youtube/*`) được API Gateway (Unit 9) proxy tới, theo `component-methods.md`/`integration-boundaries.md` đã xác nhận ở Inception (ADR-0004/ADR-0005). Chi tiết routing rule cụ thể sẽ được thiết kế ở Unit 9's Infrastructure Design, không phải ở Unit 7 này
   - ✅ Strengths: đúng phân chia trách nhiệm — Unit 7 chỉ cần biết nó sẽ được proxy tới, không cần tự thiết kế routing
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 8: Scaling Configuration
A) 💡 Suggested: 1 instance cố định, không auto-scaling — nhất quán toàn hệ thống
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 9: Monitoring Infrastructure
A) 💡 Suggested: Structured logging ra stdout, bao gồm `saga_id` (AMQP) hoặc `X-Request-ID` (REST) trong mọi log line — nhất quán
   - ✅ Strengths: đủ cho MVP local
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a
