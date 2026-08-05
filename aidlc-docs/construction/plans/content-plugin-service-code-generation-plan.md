# Code Generation Plan — Unit 2: Content Plugin Service

## Unit Context
- **Stories**: B1, B2 (xem `unit-of-work-story-map.md`)
- **Dependencies**: Unit 1 (RabbitMQ) — đã hoàn tất
- **Interfaces**: `GET /v1/plugins` (REST), consume `classify_scenes` / produce `scenes_classified`/`classification_failed` (AMQP)
- **Owned entities**: Plugin registry (in-memory, không persistence)

## Coding Standards (Step 3.5 — đã xác nhận ở Low-Level Design)
- **Naming**: Python snake_case cho function/variable, PascalCase cho class (chuẩn PEP8)
- **SOLID**: Bắt buộc — đặc biệt Dependency Inversion (use case phụ thuộc `ContentPluginPort` abstraction, không phụ thuộc `ProgrammingPlugin` cụ thể)
- **Documentation**: Python docstring (Google style) cho public class/method — giải thích WHY, không lặp lại WHAT
- **Linting**: `ruff` (nhanh, kết hợp lint + format, phù hợp dự án Python hiện đại)

## Steps

- [ ] **Step 1 — Project Structure Setup**: Tạo `services/content-plugin/` theo `module-structure.md` (domain/, application/, adapters/, main.py, tests/), `requirements.txt`, `Dockerfile`
- [ ] **Step 2 — Business Logic Generation (domain/)**: `ports.py` (ContentPluginPort), `models.py` (Scene, ClassificationResult), `errors.py` (PluginNotFoundError, InvalidCategoryError, InvalidSceneError)
- [ ] **Step 3 — Business Logic Generation (application/)**: `list_plugins.py` (ListPluginsUseCase), `classify_scene.py` (ClassifySceneUseCase) — theo `business-logic-model.md`
- [ ] **Step 4 — Business Logic Unit Testing**: `tests/domain/`, `tests/application/` — dùng `FakeContentPluginRegistry`, cover mọi business rule (Rule 1-5)
- [ ] **Step 5 — Plugin Adapter Generation**: `adapters/plugins/registry.py` (ContentPluginRegistry, dynamic discovery), `adapters/plugins/programming/plugin.py` (ProgrammingPlugin)
- [ ] **Step 6 — API Layer Generation**: `adapters/api/router.py` (`GET /v1/plugins`, `GET /health`), `adapters/api/schemas.py` (Pydantic models)
- [ ] **Step 7 — API Layer Unit Testing**: `tests/adapters/test_api.py` — dùng FastAPI `TestClient`
- [ ] **Step 8 — Messaging Layer Generation**: `adapters/messaging/consumer.py`, `adapters/messaging/producer.py`, `adapters/messaging/idempotency.py` (IdempotencyStore)
- [ ] **Step 9 — Messaging Layer Unit Testing**: `tests/adapters/test_messaging.py` — mock AMQP channel
- [ ] **Step 10 — Composition Root**: `main.py` — wiring toàn bộ (theo `dependency-injection.md`)
- [ ] **Step 11 — Documentation Generation**: Cập nhật `README.md` gốc (thêm mục cho Content Plugin Service), tạo `aidlc-docs/construction/content-plugin-service/code/README.md`
- [ ] **Step 12 — Deployment Artifacts**: Thêm service `content-plugin` vào `docker-compose.yml` gốc (theo `deployment-architecture.md`)

**Không áp dụng**: Repository Layer (không có database), Frontend Components, Database Migration Scripts.
