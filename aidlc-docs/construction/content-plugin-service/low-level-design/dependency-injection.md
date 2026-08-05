# Dependency Injection — Unit 2: Content Plugin Service

## Mechanism
Constructor injection thủ công (manual constructor injection) — không dùng DI container/framework, theo `architectural-style.md` (HLD).

## What Gets Injected vs Constructed Directly
- **Injected (abstraction)**: `ContentPluginRegistry` (implements domain interface truy vấn plugin) được inject vào `ListPluginsUseCase` và `ClassifySceneUseCase` qua constructor.
- **Constructed directly**: `Scene`, `ClassificationResult` (value objects bất biến — không cần inject, tạo trực tiếp khi cần).

## Composition Root
`main.py`:
```python
def create_app() -> FastAPI:
    registry = ContentPluginRegistry.discover("adapters/plugins/")  # ADR-0006 dynamic loading
    list_plugins_uc = ListPluginsUseCase(registry)
    classify_scene_uc = ClassifySceneUseCase(registry)

    app = FastAPI()
    app.include_router(create_router(list_plugins_uc))  # api/router.py, /v1/plugins

    amqp_consumer = create_consumer(classify_scene_uc)  # messaging/consumer.py
    app.state.amqp_consumer = amqp_consumer
    return app
```

## FastAPI Wiring
`GET /v1/plugins` dùng FastAPI `Depends()` để lấy `ListPluginsUseCase` instance đã wire sẵn từ `app.state` (không tạo instance mới mỗi request — registry chỉ discover 1 lần lúc khởi động, theo Question 6: stateless nhưng registry là in-memory cache sống suốt vòng đời service).

## Testability
Vì `ListPluginsUseCase`/`ClassifySceneUseCase` chỉ phụ thuộc `ContentPluginRegistry` (abstraction có thể mock), unit test cho application layer không cần FastAPI/AMQP thật — dùng `FakeContentPluginRegistry` trong test.
