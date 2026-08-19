# ConceptFlow — Manim Educational Video Generation Tool

## Project Overview
ConceptFlow là một pipeline sản xuất video giáo dục hoàn chỉnh, dùng [Manim](https://www.manim.community/) làm animation engine lõi, cho phép tạo video dạy học (ban đầu tập trung vào lập trình — thuật toán, cấu trúc dữ liệu, khái niệm lập trình) theo phong cách trực quan kiểu 3Blue1Brown. Pipeline đi từ script/markdown → animation + giọng đọc TTS → video hoàn chỉnh → tự động đăng YouTube, chạy hoàn toàn local qua Docker.

Kiến trúc: Microservices + Saga Orchestration qua RabbitMQ. Xem `aidlc-docs/inception/high-level-design/` và `aidlc-docs/inception/application-design/` để biết chi tiết thiết kế, và `aidlc-docs/decisions/` cho các Architecture Decision Records (ADR).

## Prerequisites
- Docker >= 24.x và Docker Compose >= v2
- (Cho phát triển từng service riêng lẻ sau này) Python >= 3.11, Node.js >= 20.x

## Installation
```bash
cp .env.example .env
# Chỉnh sửa .env với giá trị thật (RABBITMQ_USER, RABBITMQ_PASS, v.v.)
```

## Configuration
Biến môi trường cấu hình qua file `.env` (xem `.env.example` cho danh sách đầy đủ và giá trị mẫu — không commit giá trị thật vào git).

| Biến | Mô tả |
|---|---|
| `RABBITMQ_USER` | Username đăng nhập RabbitMQ (thay thế `guest` mặc định) |
| `RABBITMQ_PASS` | Password RabbitMQ |
| `POSTGRES_USER` | Username cho mọi PostgreSQL instance (database-per-service, ADR-0013) |
| `POSTGRES_PASS` | Password PostgreSQL |

## Running the Project
```bash
docker compose up -d
```
- RabbitMQ Management UI: http://localhost:15672 (đăng nhập bằng `RABBITMQ_USER`/`RABBITMQ_PASS`)
- Content Plugin Service: nội bộ (`content-plugin:8000` trong docker network), không expose ra host — dùng `docker compose logs content-plugin` hoặc `docker exec` để kiểm tra. DB riêng: `content-plugin-db` (Postgres, Inbox/Outbox — ADR-0013)
- TTS Service: message-driven qua RabbitMQ (queue `tts.commands`), không có port HTTP nào (ADR-0014) — dùng `docker compose logs tts`. DB riêng: `tts-db` (Postgres, Inbox/Outbox — ADR-0013)

## Running Tests
Mỗi service có test suite riêng (pytest). Ví dụ cho Content Plugin Service:
```bash
cd services/content-plugin
pip install -r requirements-dev.txt
pytest -q
```
Tương tự cho TTS Service:
```bash
cd services/tts
pip install -r requirements-dev.txt
pytest -q
```
Hướng dẫn test tổng hợp toàn hệ thống sẽ được bổ sung ở giai đoạn Build and Test (`aidlc-docs/construction/build-and-test/`, sau khi tất cả unit hoàn thành).

## Project Structure
```
.
├── docker-compose.yml       # Định nghĩa toàn bộ service (bắt đầu với RabbitMQ)
├── .env.example              # Mẫu biến môi trường
├── infra/
│   └── rabbitmq/              # Cấu hình topology RabbitMQ (exchange/queue/DLQ)
├── services/
│   ├── content-plugin/         # Content Plugin Service (Python/FastAPI, Hexagonal)
│   │                             # domain/ → application/ → adapters/{api,messaging,persistence,plugins}/
│   └── tts/                     # TTS Service (Python, Hexagonal, Piper engine, message-driven — ADR-0014)
│                                 # domain/ → application/ → adapters/{messaging,persistence,tts_engines,storage,logging}/
├── frontend/                  # Web GUI (React) — sẽ bổ sung ở Unit 10
├── shared/                    # Schema/type dùng chung giữa service (nếu cần)
└── aidlc-docs/                 # Toàn bộ tài liệu AI-DLC (requirements, design, ADR, audit trail)
```

## CI/CD
Chưa thiết lập — sẽ được cấu hình ở giai đoạn Build and Test (`aidlc-docs/construction/build-and-test/ci-cd-integration-instructions.md`, sau khi tất cả unit hoàn thành Construction Phase).
