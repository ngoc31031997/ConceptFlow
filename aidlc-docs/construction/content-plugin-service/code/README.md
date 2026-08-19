# Unit 2: Content Plugin Service — Code Summary

**Revision (2026-08-07, ADR-0013)**: retrofitted with PostgreSQL-backed Inbox/Outbox (system-wide learning exercise), replacing the in-memory `IdempotencyStore` + direct `exchange.publish()`. See `adapters/persistence/` below.

## Generated Files (`services/content-plugin/`)
- `domain/models.py`, `domain/ports.py`, `domain/errors.py` — pure Python, no framework dependency
- `application/list_plugins.py`, `application/classify_scene.py` — `ListPluginsUseCase`, `ClassifySceneUseCase`, `ClassifyScenesBatchUseCase` (fail-fast batch semantics)
- `adapters/plugins/registry.py` — `ContentPluginRegistry` with dynamic discovery (ADR-0006)
- `adapters/plugins/programming/plugin.py` — `ProgrammingPlugin` (FR1.2)
- `adapters/api/router.py`, `adapters/api/schemas.py` — `GET /v1/plugins` (ADR-0008), `GET /health`
- `adapters/messaging/consumer.py` — `classify_scenes` command handling; writes outcome to the Outbox instead of publishing directly
- `adapters/messaging/producer.py` — now only builds event envelopes (`success_envelope`/`failure_envelope`); no longer touches RabbitMQ itself
- `adapters/persistence/db.py`, `inbox.py`, `outbox.py`, `relay.py` — PostgreSQL pool + schema bootstrap, `InboxRepository` (durable dedupe), `OutboxRepository` (transactional event enqueue), `OutboxRelay` (polling publisher, ~1s)
- `adapters/logging/correlation.py` — `X-Request-ID`/`saga_id` correlation propagation
- `main.py` — composition root (constructor injection wiring, per `dependency-injection.md`); starts `OutboxRelay` alongside the AMQP consumer
- `Dockerfile`, `requirements.txt` (+ `asyncpg`), `requirements-dev.txt`, `pyproject.toml`

## Tests
24 tests across `tests/application/`, `tests/adapters/` (API, messaging/Inbox-Outbox, OutboxRelay, persistence, plugin discovery) — all passing under Python 3.12 (verified via `docker run python:3.12-slim`, since local dev machine runs Python 3.9). Postgres interactions are tested against `tests/adapters/fake_postgres.py` (in-memory fakes mirroring the asyncpg surface actually used), not a real database — no integration/live-Postgres test exists yet.

## Story Traceability
- Story B1 (chọn plugin) → `GET /v1/plugins`
- Story B2 (chọn loại nội dung cụ thể) → `ClassifySceneUseCase` + `ProgrammingPlugin` (category do Creator chọn qua `category_hint`, không tự suy luận — Rule 1)

## Deployment
Service `content-plugin` đã thêm vào `docker-compose.yml` gốc — build từ `./services/content-plugin`, phụ thuộc `rabbitmq` + `content-plugin-db` (cả hai `condition: service_healthy`), chỉ giao tiếp nội bộ qua docker network `backend`. `content-plugin-db` (Postgres 16, named volume `content_plugin_db_data`) mới được thêm theo ADR-0013.

## Notes cho Unit tiếp theo (Unit 4: Script Processing Service)
Unit 4 sẽ gọi `GET /v1/plugins` (Content Plugin Service) và gửi command `classify_scenes` — envelope JSON và routing key đã cố định ở `interface-contracts.md`, có thể tái sử dụng nguyên schema.
