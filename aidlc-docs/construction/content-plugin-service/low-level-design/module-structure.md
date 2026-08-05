# Module Structure — Unit 2: Content Plugin Service

## Layering (Hexagonal / Ports & Adapters — ADR-0002)

```
services/content-plugin/
├── domain/
│   ├── ports.py              # ContentPluginPort (abstract interface)
│   ├── models.py              # Scene, ClassificationResult (value objects, pure Python)
│   └── errors.py              # Domain-specific exceptions (PluginNotFoundError, ClassificationError)
├── application/
│   ├── list_plugins.py        # ListPluginsUseCase
│   └── classify_scene.py      # ClassifySceneUseCase
├── adapters/
│   ├── api/
│   │   ├── router.py           # FastAPI router, /v1/plugins (ADR-0008)
│   │   └── schemas.py          # Pydantic request/response models
│   ├── messaging/
│   │   ├── consumer.py         # AMQP consumer cho command classify_scenes
│   │   └── producer.py         # AMQP producer cho event scenes_classified/classification_failed
│   ├── plugins/
│   │   ├── registry.py         # ContentPluginRegistry — dynamic discovery (ADR-0006)
│   │   └── programming/
│   │       └── plugin.py       # ProgrammingPlugin implements ContentPluginPort
│   └── logging/
│       └── correlation.py      # Correlation ID injection vào log context
├── main.py                     # Composition root — wiring, FastAPI app, AMQP consumer startup
└── tests/
    ├── domain/
    ├── application/
    └── adapters/
```

## Dependency Direction
`adapters/` → `application/` → `domain/`. `domain/` không import bất kỳ thứ gì từ `adapters/` hay `application/`, không import FastAPI/aio-pika. `application/` chỉ phụ thuộc `domain/` (qua port interface), không biết chi tiết FastAPI/AMQP/plugin cụ thể nào. `adapters/plugins/programming/plugin.py` implement `domain/ports.py::ContentPluginPort` — đây chính là cơ chế pluggable theo NFR1/ADR-0002.

## Module Responsibilities

| Module | Responsibility |
|---|---|
| `domain/ports.py` | Định nghĩa `ContentPluginPort` (abstract: `plugin_id`, `name`, `supported_categories`, `classify(scene) -> ClassificationResult`) |
| `domain/models.py` | `Scene` (value object khớp schema trong `component-methods.md`), `ClassificationResult` (category, animation_template_id) |
| `application/list_plugins.py` | `ListPluginsUseCase(registry: ContentPluginRegistry)` — trả danh sách plugin đã đăng ký |
| `application/classify_scene.py` | `ClassifySceneUseCase(registry: ContentPluginRegistry)` — lấy plugin theo `plugin_id`, gọi `classify()`, xử lý `PluginNotFoundError` |
| `adapters/api/router.py` | FastAPI route `GET /v1/plugins` — gọi `ListPluginsUseCase` |
| `adapters/messaging/consumer.py` | Consume `classify_scenes` command từ `content_plugin.commands`, gọi `ClassifySceneUseCase` cho từng scene, publish event qua `producer.py` |
| `adapters/plugins/registry.py` | `ContentPluginRegistry` — implement discovery mechanism (Question 3): quét `adapters/plugins/`, `importlib`, `issubclass` check, dict `plugin_id -> instance` |
| `adapters/plugins/programming/plugin.py` | `ProgrammingPlugin` — plugin cụ thể cho miền lập trình (FR1.2): phân loại "algorithm" vs "concept" |
