# Infrastructure Design — Unit 7: Publisher Service

## Deployment Environment
Docker container, base `python:3.12-slim` — không cần system dependency đặc biệt (không có native extension, khác Unit 5/6). Cùng docker network `backend`.

## Storage Infrastructure — PostgreSQL (ADR-0013)
Container riêng `publisher-db` (Postgres 16, database-per-service), named volume `publisher_db_data`. Chứa Outbox/Inbox VÀ `oauth_credentials`.

## Storage Infrastructure — Shared Volume (tái sử dụng, read-only)
Dùng lại named volume `shared_artifacts` — Publisher Service CHỈ ĐỌC `/shared/{project_id}/video/final.mp4` (từ Unit 6). Mount read-only (`:ro`) để làm rõ ranh giới trách nhiệm (không ghi gì vào shared volume).

## Networking & Health Check
Port 8000 (FastAPI) chỉ nội bộ docker network `backend`, KHÔNG map ra host — OAuth callback được API Gateway (Unit 9) proxy tới, Creator không truy cập trực tiếp. Health check `GET /health`.

## OAuth Credentials Configuration
Biến môi trường `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URI` — thêm vào `.env.example` (không commit giá trị thật). Creator tự đăng ký OAuth Client trên Google Cloud Console (bước setup 1 lần, thủ công, ngoài phạm vi tự động hóa hệ thống — hướng dẫn trong README, Code Generation).

## Resource Limits
Không set `deploy.resources.limits` ở MVP (Docker Compose không phải Swarm).

## Load Balancer / API Gateway / Database Read-Write Splitting / Sharding
- **Load Balancer / Sharding**: N/A — 1 instance cố định, Postgres chỉ chứa Outbox/Inbox + 1 row credential.
- **API Gateway**: Publisher Service's REST endpoint (`/v1/auth/youtube/*`) được API Gateway (Unit 9) proxy tới (ADR-0004/ADR-0005). Chi tiết routing rule cụ thể thiết kế ở Unit 9's Infrastructure Design.

## Scaling
1 instance cố định, không auto-scaling.

## Monitoring
Structured logging ra stdout, bao gồm `saga_id` (AMQP) hoặc `X-Request-ID` (REST) trong mọi log line.
