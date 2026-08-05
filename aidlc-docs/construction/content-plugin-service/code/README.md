# Unit 2: Content Plugin Service — Code Summary

## Generated Files (`services/content-plugin/`)
- `domain/models.py`, `domain/ports.py`, `domain/errors.py` — pure Python, no framework dependency
- `application/list_plugins.py`, `application/classify_scene.py` — `ListPluginsUseCase`, `ClassifySceneUseCase`, `ClassifyScenesBatchUseCase` (fail-fast batch semantics)
- `adapters/plugins/registry.py` — `ContentPluginRegistry` with dynamic discovery (ADR-0006)
- `adapters/plugins/programming/plugin.py` — `ProgrammingPlugin` (FR1.2)
- `adapters/api/router.py`, `adapters/api/schemas.py` — `GET /v1/plugins` (ADR-0008), `GET /health`
- `adapters/messaging/consumer.py`, `producer.py`, `idempotency.py` — `classify_scenes` command handling, `scenes_classified`/`classification_failed` events, in-memory TTL dedupe
- `adapters/logging/correlation.py` — `X-Request-ID`/`saga_id` correlation propagation
- `main.py` — composition root (constructor injection wiring, per `dependency-injection.md`)
- `Dockerfile`, `requirements.txt`, `requirements-dev.txt`, `pyproject.toml`

## Tests
17 tests across `tests/application/`, `tests/adapters/` (API, messaging, plugin discovery) — all passing under Python 3.12 (verified via `docker run python:3.12-slim`, since local dev machine runs Python 3.9).

## Story Traceability
- Story B1 (chọn plugin) → `GET /v1/plugins`
- Story B2 (chọn loại nội dung cụ thể) → `ClassifySceneUseCase` + `ProgrammingPlugin` (category do Creator chọn qua `category_hint`, không tự suy luận — Rule 1)

## Deployment
Service `content-plugin` đã thêm vào `docker-compose.yml` gốc — build từ `./services/content-plugin`, phụ thuộc `rabbitmq` (`condition: service_healthy`), chỉ giao tiếp nội bộ qua docker network `backend`.

## Notes cho Unit tiếp theo (Unit 4: Script Processing Service)
Unit 4 sẽ gọi `GET /v1/plugins` (Content Plugin Service) và gửi command `classify_scenes` — envelope JSON và routing key đã cố định ở `interface-contracts.md`, có thể tái sử dụng nguyên schema.
