# Infrastructure Design — Unit 2: Content Plugin Service

## Deployment Environment
Docker container (custom image, base `python:3.12-slim`), trong docker-compose, cùng docker network `backend` với RabbitMQ.

## Not Applicable
- **Database Read/Write Splitting / Sharding**: N/A
- **Load Balancer**: N/A — 1 instance
- **API Gateway**: N/A trong phạm vi unit này

## Networking
Port 8000 (FastAPI) chỉ nội bộ docker network `backend`, không map ra host.

## Health Check
`GET /health` (custom endpoint, kiểm tra registry đã discover xong) + `healthcheck` trong docker-compose. Các service phụ thuộc dùng `depends_on: condition: service_healthy`.

## Scaling
1 instance cố định, không auto-scaling.
