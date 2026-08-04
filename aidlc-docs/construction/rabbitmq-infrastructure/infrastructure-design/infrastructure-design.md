# Infrastructure Design — Unit 1: RabbitMQ Infrastructure

## Deployment Environment
Docker container trong docker-compose, chạy trên máy cá nhân (local-only, theo NFR3/`requirements.md`). Image: `rabbitmq:3.13-management`.

## Not Applicable
- **Database Read/Write Splitting / Sharding**: N/A — không phải database quan hệ
- **Load Balancer**: N/A — 1 instance cố định
- **API Gateway**: Ngoài phạm vi unit này (xem ADR-0004, Unit 9)

## Storage
Docker named volume `rabbitmq_data` mount vào `/var/lib/rabbitmq` — tách biệt vòng đời dữ liệu khỏi vòng đời container.

## Networking
- AMQP (5672): chỉ nội bộ docker-compose network, không map ra host
- Management UI (15672): map ra `localhost:15672` cho mục đích dev/debug

## Scaling
1 instance cố định, không auto-scaling (khớp NFR2 — không yêu cầu render/scale song song).

## Health Check
`healthcheck` dùng `rabbitmq-diagnostics ping`; các service phụ thuộc (Unit 2, 4, 5, 6, 7, 8) dùng `depends_on: condition: service_healthy` để đợi RabbitMQ sẵn sàng.
