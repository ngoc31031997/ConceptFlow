# Unit of Work

10 unit, mỗi unit tương ứng 1 microservice độc lập triển khai (theo Application Design). Thứ tự phát triển đề xuất theo dependency-first (xem `unit-of-work-dependency.md`).

## Code Organization Strategy (Greenfield, Monorepo)
```
manim-edu-video-tool/
├── docker-compose.yml
├── services/
│   ├── gateway/
│   ├── orchestrator/
│   ├── content-plugin/
│   ├── script-processing/
│   ├── rendering/
│   ├── tts/
│   ├── video-assembly/
│   └── publisher/
├── frontend/                  # Web GUI (React)
├── infra/
│   └── rabbitmq/               # cấu hình RabbitMQ (exchange/queue topology)
└── shared/                     # schema/type dùng chung giữa service (nếu cần, xác định ở Low-Level Design)
```
Mỗi thư mục trong `services/` là 1 unit độc lập, có Dockerfile riêng; `docker-compose.yml` ở root build & chạy toàn bộ theo FR8.1.

## Units

### Unit 1: RabbitMQ Infrastructure
- **Type**: Infrastructure unit (không phải business service)
- **Scope**: Cấu hình exchange/queue topology cho RabbitMQ (`infra/rabbitmq/`), định nghĩa các queue: `script_processing.commands`, `content_plugin.commands`, `tts.commands`, `rendering.commands`, `video_assembly.commands`, `publisher.commands`, `orchestrator.events`
  - **Revision (2026-08-07, ADR-0014)**: thêm queue `tts.commands` — TTS Service (Unit 3) nay là message-driven, không còn REST-only
- **Depends on**: Không có

### Unit 2: Content Plugin Service
- **Scope**: FR1.1–FR1.3, dynamic plugin loading (ADR-0006), plugin "Lập trình" (FR1.2)
- **Depends on**: Unit 1 (RabbitMQ)

### Unit 3: TTS Service
- **Scope**: FR4.1, FR4.2 — sinh giọng đọc offline song ngữ Việt/Anh
- **Depends on**: Unit 1 (RabbitMQ) — **Revision (2026-08-07, ADR-0014)**: trước đây "Không có (không tham gia RabbitMQ, chỉ REST)"; nay message-driven, bước Saga độc lập ("Synthesize Speech"), không còn được gọi trực tiếp bởi Rendering Service

### Unit 4: Script Processing Service
- **Scope**: FR2.1, FR2.2 — parse script thành scene (KHÔNG tự gọi Content Plugin Service — Orchestrator điều phối bước classify riêng, ADR-0012)
- **Depends on**: Unit 1 (RabbitMQ) — không còn phụ thuộc trực tiếp Unit 2 (không gọi Content Plugin Service)

### Unit 5: Rendering Service
- **Scope**: FR3.1, FR3.2, FR4.3 — render animation Manim, đồng bộ timing với audio đã sinh sẵn từ bước Saga "Synthesize Speech" (không tự gọi TTS Service, ADR-0014)
- **Depends on**: Unit 1 (RabbitMQ) — không còn phụ thuộc trực tiếp Unit 3 (không gọi TTS Service)

### Unit 6: Video Assembly Service
- **Scope**: FR5.1, FR5.2 — ghép animation + audio + nhạc nền thành .mp4
- **Depends on**: Unit 1 (RabbitMQ)

### Unit 7: Publisher Service
- **Scope**: FR7.1, FR7.2, FR7.3 — OAuth YouTube, upload video
- **Depends on**: Unit 1 (RabbitMQ)

### Unit 8: Orchestrator Service
- **Scope**: Saga coordination (2 Saga: Render Pipeline, Publish), state machine của video project, compensating actions (ADR-0007)
- **Depends on**: Unit 1 (RabbitMQ), Unit 2, 3, 4, 5, 6, 7 (cần tất cả service nghiệp vụ sẵn sàng để test luồng Saga end-to-end)

### Unit 9: API Gateway
- **Scope**: Routing REST/SSE cho GUI, proxy tới Orchestrator Service và Content Plugin Service (`GET /plugins`), Publisher Service (OAuth flow)
- **Depends on**: Unit 8 (Orchestrator Service), Unit 2 (Content Plugin Service), Unit 7 (Publisher Service)

### Unit 10: Web GUI
- **Scope**: FR6.1, FR6.2 — toàn bộ giao diện Creator (Epic A-F trong `stories.md`)
- **Depends on**: Unit 9 (API Gateway)
