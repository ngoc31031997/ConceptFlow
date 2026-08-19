# Infrastructure Design — Unit 3: TTS Service

**Revision (2026-08-07, ADR-0014, ADR-0013)**: Port/health check section updated (no more FastAPI/REST); new PostgreSQL container added.

## Deployment Environment
Docker container (custom image, base `python:3.12-slim` + system dependencies cần thiết cho Piper qua `apt-get`), trong docker-compose, cùng docker network `backend`. Không đổi.

## Storage Infrastructure — Shared Volume
Không đổi — named volume `shared_artifacts` mount vào `/shared`.

## Compute Infrastructure — Voice Model Bundling
Không đổi.

## Storage Infrastructure — PostgreSQL (MỚI, ADR-0013)
Named volume `tts_db_data` cho container `tts-db` (Postgres 16, database-per-service — riêng biệt hoàn toàn với `content-plugin-db`).

## Networking
**Revised**: Không còn expose port HTTP (không còn FastAPI/REST) — service chỉ kết nối outbound tới RabbitMQ + PostgreSQL, không có inbound port nào cần lắng nghe.

## Health Check
**Revised**: Không còn `GET /health` (không còn REST). Dùng sentinel file: `main.py` ghi `/tmp/ready` sau khi AMQP consumer + `OutboxRelay` đã khởi động thành công (mirror ý nghĩa "voice model đã load" trước đây — nay mở rộng thành "sẵn sàng nhận command"). Docker `healthcheck` kiểm tra file này tồn tại (`test: ["CMD", "test", "-f", "/tmp/ready"]`) — không cần thư viện HTTP nào.

## Not Applicable
- **Load Balancer**: N/A — 1 instance cố định
- **API Gateway**: N/A — không còn REST endpoint nào để expose
- **Database Read/Write Splitting / Sharding**: N/A — Postgres chỉ chứa Outbox/Inbox, quy mô nhỏ

## Scaling
1 instance cố định, không auto-scaling — không đổi.

## Monitoring
Structured logging ra stdout, bao gồm `saga_id` (từ envelope AMQP, không còn header HTTP) trong mọi log line.
