# Unit 1: RabbitMQ Infrastructure — Code Summary

## Generated Artifacts
- `docker-compose.yml` (root) — service `rabbitmq`, image `rabbitmq:3.13-management`, named volume `rabbitmq_data`, healthcheck, internal-only AMQP port, exposed Management UI port
- `infra/rabbitmq/definitions.json` — topology: 3 exchange (`commands.direct`, `events.direct`, `dlx.direct`), 6 queue nghiệp vụ (5 command + 1 event), 5 DLQ, đầy đủ bindings theo `logical-components.md`
- `infra/rabbitmq/rabbitmq.conf` — trỏ RabbitMQ nạp `definitions.json` lúc khởi động (`management.load_definitions`)
- `.env.example` (root) — mẫu biến môi trường `RABBITMQ_USER`, `RABBITMQ_PASS`
- `README.md` (root) — tài liệu tổng quan dự án, hướng dẫn cài đặt/chạy

## Story Traceability
Unit này không map trực tiếp tới story nào (hạ tầng nền) — hỗ trợ gián tiếp Story C1, C6, E3 (theo `unit-of-work-story-map.md`) bằng cách cung cấp kênh giao tiếp đáng tin cậy cho Orchestrator Service.

## Notes cho các Unit phụ thuộc (Unit 2, 4, 5, 6, 7, 8)
- Retry với exponential backoff (1s/5s/15s, tối đa 3 lần — theo `nfr-requirements.md`) là trách nhiệm của **consumer** (mỗi service), không phải của RabbitMQ core (không dùng delay-plugin ở giai đoạn này) — cần implement logic requeue-with-delay hoặc dùng thư viện hỗ trợ (`aio-pika` retry decorator) khi code các unit đó.
- Kết nối AMQP dùng client `aio-pika` (theo `tech-stack-decisions.md`), connect qua `amqp://<user>:<pass>@rabbitmq:5672/` (tên service `rabbitmq` trong docker network `backend`).
- Mọi service phải khai báo `depends_on: rabbitmq: condition: service_healthy` trong `docker-compose.yml` khi được thêm vào.

## Deployment
Không cần Dockerfile riêng — dùng image chính thức `rabbitmq:3.13-management` từ Docker Hub, cấu hình qua volume mount (`definitions.json`, `rabbitmq.conf`) và biến môi trường.
