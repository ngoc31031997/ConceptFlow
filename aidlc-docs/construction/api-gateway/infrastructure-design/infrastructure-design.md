# Infrastructure Design — Unit 9: API Gateway

## Deployment Environment
Docker container, `node:20-alpine`, network `backend` — chạy local, không cloud provider.

## Compute Infrastructure
1 container `api-gateway`, fixed 1 instance, không auto-scaling.

## Storage Infrastructure
Không áp dụng — Gateway hoàn toàn stateless, không database/volume riêng.

## Database Read/Write Splitting & Sharding
Không áp dụng — không có database.

## Messaging Infrastructure
Kết nối `rabbitmq:5672` nội bộ, declare 1 exclusive queue bind vào `progress.fanout` lúc start — dùng lại topology Unit 1, không thêm exchange/queue mới.

## Networking Infrastructure — Entry Point Duy Nhất
Khác mọi service khác (chỉ nội bộ `backend`), Gateway PHẢI expose port ra host — đây là entry point duy nhất cho GUI (browser Creator). Map `8080:8080`. Đây là port thứ 2 (sau RabbitMQ Management UI `15672`, dev-only) được publish ra host trong toàn hệ thống.

## Health Check
`GET /health` → `200 {status: "ok"}`, KHÔNG kiểm tra downstream (tránh cascading failure khi 1 service downstream tạm gián đoạn nhưng Gateway vẫn phục vụ các route khác bình thường).

## Load Balancer
Không áp dụng — 1 instance duy nhất.

## API Gateway (chính unit này)
Unit 9 LÀ implementation cụ thể của quyết định "API Gateway" tại `integration-boundaries.md` (Inception) — không cần thêm sản phẩm gateway thương mại (Kong/Traefik/AWS API Gateway) phía trước.

## Monitoring Infrastructure
Không có stack riêng — `docker-compose logs` + `pino` JSON logging.

## Shared Infrastructure
Dùng lại: `rabbitmq` (Unit 1), `backend` network. Kết nối trực tiếp tới 3 service downstream đã build (`orchestrator:8000`, `content-plugin:8000`, `publisher:8000` — nội bộ network).
