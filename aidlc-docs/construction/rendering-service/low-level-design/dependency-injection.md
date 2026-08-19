# Dependency Injection — Unit 5: Rendering Service

## Mechanism
Constructor injection thủ công — không đổi so với Unit 2/3/4.

## What Gets Injected vs Constructed Directly
- **Injected (abstraction)**: `AnimationRendererPort` — `RenderSceneUseCase` nhận instance implement `AnimationRendererPort` (`ManimAnimationRenderer`) qua constructor. Cho phép thay animation engine sau này (không phải Manim) mà không sửa `application/`.
- **Constructed directly**: `AnimationTemplateRegistry` (kết quả `discover()`, truyền vào `ManimAnimationRenderer`), `InboxRepository`/`OutboxRepository`/`OutboxRelay` (Postgres, ADR-0013, mirror Unit 2/3/4), `artifact_paths.py` helper thuần túy.
- **Template plugin**: mỗi `AnimationTemplatePort` implementation tự đăng ký qua discovery (ADR-0015) — không inject thủ công từng template, giống `ContentPluginRegistry`.

## Composition Root
`main.py`:
1. `AnimationTemplateRegistry.discover()` — scan `adapters/rendering/templates/`.
2. Khởi tạo `ManimAnimationRenderer(registry=template_registry, timeout_seconds=env)`.
3. Khởi tạo `RenderSceneUseCase(renderer=manim_renderer)`, sau đó `RenderScenesBatchUseCase(single=render_scene_use_case)`.
4. Kết nối PostgreSQL, khởi tạo `InboxRepository`, `OutboxRepository`.
5. Kết nối RabbitMQ, wire `RenderScenesCommandHandler(batch_use_case, pool, inbox, outbox)`, đăng ký consumer cho queue `rendering.commands`.
6. Khởi động `OutboxRelay` như background task.
7. Ghi sentinel `/tmp/ready`.

## Wiring Diagram
```
main.py
  ├── AnimationTemplateRegistry.discover() (ADR-0015)
  │     └── injected into → ManimAnimationRenderer (implements AnimationRendererPort)
  │           └── injected into → RenderSceneUseCase
  │                 └── injected into → RenderScenesBatchUseCase
  ├── InboxRepository, OutboxRepository (Postgres, ADR-0013)
  │     └── injected into → RenderScenesCommandHandler (AMQP consumer)
  │           └── consumes command "render_scenes" (queue rendering.commands)
  │           └── ghi event "scene_render_started"/"scene_rendered" (per scene) + "rendering_completed"/"rendering_failed" (cuối batch) vào Outbox
  └── OutboxRelay (background task)
        └── poll Outbox chưa publish → publish qua producer.py → đánh dấu published_at
```
