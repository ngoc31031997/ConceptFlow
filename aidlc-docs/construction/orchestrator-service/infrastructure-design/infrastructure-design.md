# Infrastructure Design — Unit 8: Orchestrator Service

## Deployment Environment
Docker container, chạy trong `docker-compose.yml` gốc, network `backend` (nội bộ, cùng các unit khác). Không có cloud provider — hệ thống chạy hoàn toàn local trên máy cá nhân (`technology-direction.md`'s constraint).

## Compute Infrastructure
- 1 container `orchestrator` (Go binary, build từ `services/orchestrator/`).
- **Scaling**: Fixed, 1 instance. Không auto-scaling (nhất quán tất cả unit khác, quy mô 1 Creator).

## Storage Infrastructure
- **PostgreSQL**: container riêng `orchestrator-db` (Postgres 16, database-per-service — ADR-0013), named volume `orchestrator_db_data`. Bảng: `projects`, `saga_steps`, `outbox_events` (commands — ADR-0019), `processed_messages`.
- **Shared Volume**: KHÔNG mount `shared_artifacts` — Orchestrator không đọc/ghi file artifact trực tiếp, chỉ lưu path dưới dạng string trong `Project`.

## Database Read/Write Splitting
Single primary, không read replica — tải đọc rất thấp (1 Creator, GET status không phải hot path, phần lớn cập nhật tiến trình qua SSE thay vì polling REST).

## Database Sharding/Partitioning
Không cần sharding — quy mô dữ liệu (1 project/video, tăng chậm) không tiệm cận giới hạn 1 Postgres instance đơn. Ghi nhận rõ: không có target write throughput/dataset size nào trong NFR Requirements biện minh cho sharding.

## Messaging Infrastructure
Kết nối `rabbitmq:5672` (container `rabbitmq`, hạ tầng Unit 1) — dùng lại topology đã định nghĩa: `commands.direct` (publish 6 command), `orchestrator.events` (consume 12 event), `progress.fanout` (publish progress), 6 `*.commands.dlq` (consume). Không cần thêm exchange/queue mới. Consumer prefetch=1.

## Networking Infrastructure
- Port 8000 (chi HTTP server) chỉ nội bộ `backend` network, KHÔNG map ra host.
- Health check: `GET /health` (route mới, kiểm tra kết nối Postgres).
- TLS: không áp dụng (nội bộ Docker network, không expose internet).

## Load Balancer
Không áp dụng — 1 instance duy nhất.

## API Gateway
API Gateway (Unit 9, chưa xây) proxy 4 endpoint REST của Orchestrator (`/v1/sagas/render`, `/v1/sagas/publish`, `/v1/projects/{id}`, `/v1/projects/{id}/retry`) tới `orchestrator:8000`. Routing rule cụ thể thiết kế ở Unit 9's Infrastructure Design. Gateway không enforce auth/rate-limit (threat model single-user, nhất quán NFR Design).

## Monitoring Infrastructure
Không có monitoring stack riêng — `docker-compose logs` + structured JSON logging (`log/slog`, NFR Requirements). RabbitMQ Management UI (`localhost:15672`, Unit 1) dùng để quan sát queue/DLQ depth khi cần debug.

## Shared Infrastructure
Dùng lại: `rabbitmq` (Unit 1), `backend` network (root `docker-compose.yml`), quy ước database-per-service (ADR-0013). Không tạo shared infrastructure mới ở Unit 8.
