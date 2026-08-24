# Infrastructure Design — Unit 6: Video Assembly Service

## Deployment Environment
Docker container, base `python:3.12-slim` + system dependency: `ffmpeg` (bao gồm `ffprobe`). Không cần thêm dependency nào khác — không có native extension cần compile (khác Unit 5's Manim).

## Storage Infrastructure — PostgreSQL (ADR-0013)
Container riêng `video-assembly-db` (Postgres 16, database-per-service), named volume `video_assembly_db_data`.

## Storage Infrastructure — Shared Volume (tái sử dụng)
Dùng lại named volume `shared_artifacts` (đã có từ TTS/Rendering Service) — Video Assembly Service đọc `/shared/{project_id}/animations/` (Unit 5) + `/shared/{project_id}/audio/` (Unit 3), ghi `/shared/{project_id}/video/final.mp4` + `_tmp/` (file trung gian, dọn sau khi thành công).

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
