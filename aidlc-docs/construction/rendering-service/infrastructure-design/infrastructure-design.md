# Infrastructure Design — Unit 5: Rendering Service

## Deployment Environment
Docker container, base `python:3.12-slim` + system dependencies cho Manim: `ffmpeg`, `libcairo2-dev`, `libpango1.0-dev`, `pkg-config`, build tools. KHÔNG cài LaTeX (`texlive`) — không cần ở MVP (2 template hiện tại không dùng công thức toán), giữ image nhỏ.

## Storage Infrastructure — PostgreSQL (ADR-0013)
Container riêng `rendering-db` (Postgres 16, database-per-service), named volume `rendering_db_data`.

## Storage Infrastructure — Shared Volume (tái sử dụng)
Dùng lại named volume `shared_artifacts` (đã có từ TTS Service) — Rendering Service ghi vào `/shared/{project_id}/animations/`, TTS Service ghi vào `/shared/{project_id}/audio/`, cùng volume khác thư mục con.

## Networking & Health Check
Không expose port (không REST). Health check qua sentinel file `/tmp/ready`.

## Resource Limits
Không set `deploy.resources.limits` ở MVP (Docker Compose không phải Swarm, `deploy` key không có tác dụng khi chạy `docker compose up` thường). Có thể set `mem_limit`/`cpus` sau nếu cần trên máy dev yếu.

## Not Applicable
- **Load Balancer**: N/A — 1 instance cố định
- **API Gateway**: N/A — không có REST endpoint
- **Database Read/Write Splitting / Sharding**: N/A — Postgres chỉ chứa Outbox/Inbox

## Scaling
1 instance cố định, không auto-scaling.

## Monitoring
Structured logging ra stdout, bao gồm `saga_id` trong mọi log line.
