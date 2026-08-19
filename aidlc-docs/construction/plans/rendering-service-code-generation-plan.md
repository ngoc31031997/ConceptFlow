# Code Generation Plan — Unit 5: Rendering Service

## Unit Context
- **Stories**: B3, C3 (xem `unit-of-work-story-map.md`)
- **Dependencies**: Unit 1 (RabbitMQ) — đã hoàn tất; KHÔNG phụ thuộc trực tiếp unit nào khác
- **Interfaces**: AMQP consumer `render_scenes` (queue `rendering.commands`) → 4 loại event qua Outbox (ADR-0013)
- **Owned entities**: `outbox_events`/`processed_messages` (Postgres `rendering-db`); animation clips trên shared volume

## Coding Standards (nhất quán Unit 2/3/4)
- **Naming**: Python snake_case cho function/variable, PascalCase cho class
- **SOLID**: Bắt buộc — `RenderSceneUseCase` phụ thuộc `AnimationRendererPort` abstraction; template plugin phụ thuộc `AnimationTemplatePort`
- **Documentation**: Python docstring (Google style), giải thích WHY
- **Linting**: `ruff`

## Steps

- [x] **Step 1 — Project Structure Setup**: Tạo `services/rendering/` (domain/, application/, adapters/{messaging,persistence,rendering,rendering/templates,logging}/, main.py, tests/), `requirements.txt`, `Dockerfile`
- [x] **Step 2 — Business Logic Generation (domain/)**: `models.py` (`SceneRenderRequest`, `SceneRenderResult`), `errors.py` (`UnsupportedTemplateError`, `InvalidDurationError`, `AnimationEngineError`), `ports.py` (`AnimationRendererPort`, `AnimationTemplatePort`)
- [x] **Step 3 — Business Logic Generation (application/)**: `render_scene.py` (`RenderSceneUseCase` — zero-trust validate + idempotency check), `render_scenes_batch.py` (`RenderScenesBatchUseCase` — fail-fast batch)
- [x] **Step 4 — Business Logic Unit Testing**: `tests/domain/`, `tests/application/` — dùng `FakeAnimationRenderer`, cover mọi business rule (Rule 1-5)
- [x] **Step 5 — Storage Adapter Generation**: `adapters/storage/artifact_paths.py` (đường dẫn `/shared/{project_id}/animations/{scene_index}.mp4`, idempotency check)
- [x] **Step 6 — Animation Template Plugins**: `adapters/rendering/templates/algorithm_visualization.py`, `concept_illustration.py` (implement `AnimationTemplatePort`, dùng Manim `Code` mobject cho code_snippet khi có)
- [x] **Step 7 — Template Registry Generation**: `adapters/rendering/registry.py` (`AnimationTemplateRegistry.discover()` — dynamic discovery, mirror `ContentPluginRegistry`, ADR-0015)
- [x] **Step 8 — Manim Renderer Adapter Generation**: `adapters/rendering/manim_renderer.py` (`ManimAnimationRenderer` implements `AnimationRendererPort` — threadpool, `RENDER_TIMEOUT_SECONDS`, gọi registry + template.build_scene + Manim render)
- [x] **Step 9 — Renderer/Template Unit Testing**: `tests/adapters/test_manim_renderer.py`, `test_registry.py` — dùng `FakeAnimationTemplate`, không chạy Manim thật (mock `scene.render()`)
- [x] **Step 10 — Persistence Adapter Generation**: `adapters/persistence/{db,inbox,outbox,relay}.py` (copy nguyên mẫu Unit 2/3/4, ADR-0013)
- [x] **Step 11 — Logging Adapter Generation**: `adapters/logging/correlation.py`
- [x] **Step 12 — Messaging Layer Generation**: `adapters/messaging/consumer.py` (`RenderScenesCommandHandler` — inbox check → batch render → per-scene Outbox commit riêng (`scene_render_started`/`scene_rendered`) → event cuối + inbox mark cùng transaction → ack), `adapters/messaging/producer.py` (4 envelope builders)
- [x] **Step 13 — Messaging Layer Unit Testing**: `tests/adapters/test_messaging.py` — mock AMQP + `FakePool`, verify per-scene commit behavior
- [x] **Step 14 — Persistence/Relay Unit Testing**: `tests/adapters/test_persistence.py`, `test_relay.py` (copy nguyên mẫu Unit 2/3/4)
- [x] **Step 15 — Composition Root**: `main.py` — wiring toàn bộ (plain asyncio, `/tmp/ready` sentinel)
- [x] **Step 16 — Documentation Generation**: Cập nhật `README.md` gốc, tạo `aidlc-docs/construction/rendering-service/code/README.md`
- [x] **Step 17 — Deployment Artifacts**: Thêm service `rendering` + `rendering-db` + volume vào `docker-compose.yml` gốc (`shared_artifacts` đã có, chỉ mount thêm)

**Không áp dụng**: Repository Layer riêng ngoài Outbox/Inbox, Frontend Components, Database Migration Scripts, API Layer (không có REST).

**Note về Manim thật**: Cài đặt `manim` package đầy đủ (native deps: cairo, pango, ffmpeg) khá nặng — unit test dùng `FakeAnimationRenderer`/mock `scene.render()` nên KHÔNG cần môi trường có Manim cài đầy đủ để chạy test suite. Việc render Manim thật (integration/manual test) sẽ cần môi trường Docker build đầy đủ theo `Dockerfile`, ngoài phạm vi test tự động ở bước này.
