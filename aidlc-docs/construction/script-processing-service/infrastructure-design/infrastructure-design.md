# Infrastructure Design — Unit 4: Script Processing Service

## Deployment Environment
Docker container, base `python:3.12-slim`, build trong `docker-compose.yml` root, cùng docker network `backend`. Không cần system dependency đặc biệt (không như TTS Service cần Piper).

## Storage Infrastructure — PostgreSQL (ADR-0013)
Container riêng `script-processing-db` (Postgres 16, database-per-service), named volume `script_processing_db_data` — giống hệt `content-plugin-db`/`tts-db`.

## Networking
Không expose port nào — không có REST/HTTP (giống TTS Service sau retrofit). Chỉ kết nối outbound tới RabbitMQ + PostgreSQL.

## Health Check
Sentinel file `/tmp/ready` — ghi sau khi AMQP consumer + `OutboxRelay` khởi động thành công. Docker `healthcheck` kiểm tra file này (`test -f /tmp/ready`).

## Not Applicable
- **Load Balancer**: N/A — 1 instance cố định
- **API Gateway**: N/A — không có REST endpoint nào để expose
- **Database Read/Write Splitting / Sharding**: N/A — Postgres chỉ chứa Outbox/Inbox, quy mô nhỏ

## Scaling
1 instance cố định, không auto-scaling.

## Monitoring
Structured logging ra stdout, bao gồm `saga_id` (từ envelope AMQP) trong mọi log line.
