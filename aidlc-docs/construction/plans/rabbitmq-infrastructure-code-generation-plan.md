# Code Generation Plan — Unit 1: RabbitMQ Infrastructure

## Unit Context
- **Stories**: Không có story trực tiếp (hạ tầng nền — hỗ trợ C1, C6, E3 gián tiếp, theo `unit-of-work-story-map.md`)
- **Dependencies**: Không có (unit đầu tiên trong dependency-first sequence)
- **Interfaces**: AMQP (port 5672, internal network), Management UI (port 15672)
- **Owned entities**: Exchange/queue topology (không phải business data)
- **Project type**: Greenfield multi-unit (microservices, monorepo) — đây là unit đầu tiên nên cũng thiết lập luôn cấu trúc thư mục gốc theo `unit-of-work.md`

## Coding Standards (Step 3.5)
Unit này chủ yếu là config (YAML/JSON), không có business logic code, nên phần lớn quy tắc SOLID/OOP không áp dụng trực tiếp. Xác nhận:
- **Naming convention**: biến môi trường `UPPER_SNAKE_CASE` (chuẩn Docker/12-factor), tên service/container `kebab-case` (`rabbitmq`)
- **Documentation**: README.md gốc dự án + comment trong docker-compose.yml cho phần cấu hình không hiển nhiên
- **Linting**: không cần linter riêng cho YAML/JSON ở mức này (dự án cá nhân, quy mô nhỏ)

Nếu bạn muốn điều chỉnh, trả lời trong [Answer] bên dưới, nếu không tôi dùng mặc định trên.

[Answer]: (để trống nếu đồng ý dùng mặc định trên)

## Steps

- [x] **Step 1 — Project Structure Setup (root, greenfield)**: Tạo cấu trúc thư mục gốc theo `unit-of-work.md`: `docker-compose.yml`, `.env.example`, `README.md`, thư mục `services/`, `frontend/`, `infra/rabbitmq/`, `shared/`
- [x] **Step 2 — RabbitMQ Topology Definition**: Tạo `infra/rabbitmq/definitions.json` khai báo exchange (`commands.direct`, `events.direct`, `dlx.direct`), 5 queue command + 1 queue event + 5 DLQ, bindings, theo `logical-components.md`
- [x] **Step 3 — docker-compose Service Entry**: Thêm service `rabbitmq` vào `docker-compose.yml` theo `deployment-architecture.md` (image, volume, network, healthcheck, ports, load definitions.json khi khởi động qua `rabbitmq.conf`)
- [x] **Step 4 — Environment Variables**: Thêm `RABBITMQ_USER`, `RABBITMQ_PASS` vào `.env.example`
- [x] **Step 5 — Documentation Generation**: Tạo `README.md` gốc dự án; tạo `aidlc-docs/construction/rabbitmq-infrastructure/code/README.md` tóm tắt phần code của unit này
- [x] **Step 6 — Deployment Artifacts**: Xác nhận `docker-compose.yml` + `infra/rabbitmq/definitions.json` + `infra/rabbitmq/rabbitmq.conf` là đầy đủ deployment artifact cho unit này; đã validate qua `docker compose config` (OK) và JSON syntax check (OK)

**Không áp dụng cho unit này**: Business Logic Generation, API Layer, Repository Layer, Frontend Components, Database Migration Scripts (unit thuần hạ tầng messaging, không có code nghiệp vụ hay database riêng).
