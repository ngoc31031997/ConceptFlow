# Module Structure — Unit 5: Rendering Service

## Layering (Hexagonal / Ports & Adapters — ADR-0002)

```
services/rendering/
├── domain/
│   ├── models.py                  # SceneRenderRequest, SceneRenderResult (value objects)
│   ├── errors.py                   # UnsupportedTemplateError, InvalidDurationError, AnimationEngineError (InvalidDurationError added at Functional Design, Business Rule 1)
│   └── ports.py                    # AnimationRendererPort, AnimationTemplatePort
├── application/
│   ├── render_scene.py             # RenderSceneUseCase (single scene, idempotency check)
│   └── render_scenes_batch.py      # RenderScenesBatchUseCase (fail-fast batch, mirror Unit 2/3/4)
├── adapters/
│   ├── messaging/
│   │   ├── consumer.py              # AMQP consumer for render_scenes (queue rendering.commands)
│   │   └── producer.py              # Envelope builders: scene_render_started, scene_rendered, rendering_completed, rendering_failed
│   ├── persistence/
│   │   ├── db.py                     # PostgreSQL pool + schema bootstrap (ADR-0013, copied verbatim)
│   │   ├── inbox.py                  # InboxRepository
│   │   ├── outbox.py                 # OutboxRepository
│   │   └── relay.py                  # OutboxRelay
│   ├── rendering/
│   │   ├── registry.py               # AnimationTemplateRegistry — dynamic discovery (ADR-0015, mirrors Unit 2's ContentPluginRegistry)
│   │   ├── manim_renderer.py         # ManimAnimationRenderer implements AnimationRendererPort — threadpool + timeout, calls registry to pick template, executes Manim
│   │   └── templates/
│   │       ├── algorithm_visualization.py   # AlgorithmVisualizationTemplate implements AnimationTemplatePort
│   │       └── concept_illustration.py       # ConceptIllustrationTemplate implements AnimationTemplatePort
│   ├── storage/
│   │   └── artifact_paths.py         # Shared-volume path convention (mirror TTS's artifact_paths.py)
│   └── logging/
│       └── correlation.py            # saga_id injection vào log context
├── main.py                          # Composition root — wiring, AMQP consumer + OutboxRelay startup, /tmp/ready sentinel
└── tests/
    ├── domain/
    ├── application/
    └── adapters/
```

## Dependency Direction
`adapters/` → `application/` → `domain/`. `domain/` không import Manim/aio-pika/asyncpg. `application/` chỉ phụ thuộc `domain/` qua `AnimationRendererPort` (abstraction) — không biết chi tiết Manim/template registry cụ thể nào. `adapters/rendering/manim_renderer.py` implement `domain/ports.py::AnimationRendererPort`; các template plugin implement `domain/ports.py::AnimationTemplatePort` và được `AnimationTemplateRegistry` tự động discover (ADR-0015) — cơ chế giống hệt `ContentPluginRegistry` ở Unit 2.

## Module Responsibilities

| Module | Responsibility |
|---|---|
| `domain/models.py` | `SceneRenderRequest` (`project_id`, `scene_index`, `narration_text`, `illustration_hint`, `code_snippet`, `code_language`, `animation_template_id`, `duration_seconds` — target từ audio), `SceneRenderResult` (`animation_path`, `duration_seconds` — thực tế) |
| `domain/errors.py` | `UnsupportedTemplateError` (không tìm thấy `animation_template_id` trong registry), `InvalidDurationError` (`duration_seconds` ≤ 0 — thêm ở Functional Design, Business Rule 1), `AnimationEngineError` (Manim crash/timeout) |
| `domain/ports.py` | `AnimationRendererPort` (abstract: `render(request, output_path) -> float`), `AnimationTemplatePort` (abstract: `template_id` property, `build_scene(request) -> manim.Scene`) |
| `application/render_scene.py` | `RenderSceneUseCase(renderer: AnimationRendererPort)` — tính đường dẫn artifact, kiểm tra idempotency (file tồn tại), gọi `renderer.render(...)` nếu chưa có |
| `application/render_scenes_batch.py` | `RenderScenesBatchUseCase(single: RenderSceneUseCase)` — lặp qua scenes, fail-fast, publish `scene_render_started` trước mỗi scene (qua callback tới consumer — xem `interface-contracts.md`) |
| `adapters/rendering/registry.py` | `AnimationTemplateRegistry.discover()` — scan `adapters/rendering/templates/`, đăng ký mọi class implement `AnimationTemplatePort` (mirror `ContentPluginRegistry.discover()`, ADR-0006/ADR-0015) |
| `adapters/rendering/manim_renderer.py` | `ManimAnimationRenderer` — implement `AnimationRendererPort`; lấy template từ registry theo `animation_template_id` (raise `UnsupportedTemplateError` nếu không có), chạy Manim's `scene.render()` trong `ThreadPoolExecutor`, timeout đọc từ env var (mặc định 300s), đo duration thực tế từ file output |
| `adapters/rendering/templates/algorithm_visualization.py` | `AlgorithmVisualizationTemplate` — `template_id = "algorithm_visualization"`, build Manim `Scene` minh họa thuật toán từng bước; nếu `code_snippet` có, chèn Manim `Code` mobject (Pygments syntax highlight, Story B3) |
| `adapters/rendering/templates/concept_illustration.py` | `ConceptIllustrationTemplate` — `template_id = "concept_illustration"`, build Manim `Scene` minh họa khái niệm; cũng chèn `Code` mobject nếu có `code_snippet` |
| `adapters/storage/artifact_paths.py` | Đường dẫn quy ước `/shared/{project_id}/animations/{scene_index}.mp4`, kiểm tra tồn tại (idempotency) |
| `adapters/messaging/consumer.py` | Consume `render_scenes`, publish `scene_render_started` trước mỗi scene + `scene_rendered` sau mỗi scene (nhiều Outbox row/command) + `rendering_completed`/`rendering_failed` cuối cùng |
| `adapters/logging/correlation.py` | Gắn `saga_id` vào log context |
