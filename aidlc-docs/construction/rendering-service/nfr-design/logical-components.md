# Logical Components — Unit 5: Rendering Service

## Components
| Component | Type | Purpose |
|---|---|---|
| AMQP Consumer (`aio-pika`) | Message consumer | Nhận `render_scenes` (queue `rendering.commands`) |
| `AnimationTemplateRegistry` | In-process registry | Dynamic discovery template plugin (ADR-0015), load 1 lần lúc khởi động |
| `ManimAnimationRenderer` | Animation engine wrapper | Implement `AnimationRendererPort`, chạy Manim trong threadpool, `RENDER_TIMEOUT_SECONDS` |
| Shared Docker Volume | File storage | Lưu animation clip `.mp4`, cơ chế idempotency artifact-level |
| PostgreSQL (`rendering-db`) | Database | `outbox_events`/`processed_messages` (ADR-0013) |
| `InboxRepository`/`OutboxRepository` | Persistence adapter | Dedupe message + enqueue event (per-scene commit riêng, xem `nfr-design-patterns.md`) |
| `OutboxRelay` | Background poller | Publish event chưa gửi tới `orchestrator.events` |

## No External Infrastructure Components
Không cần cache ngoài (Redis), không cần circuit breaker riêng.

## Diagram

```mermaid
flowchart TB
    subgraph Unit5["Rendering Service (Python)"]
        CONSUMER["AMQP Consumer<br/>(render_scenes)"]
        UC["RenderScenesBatchUseCase"]
        REG["AnimationTemplateRegistry<br/>(ADR-0015)"]
        RENDERER["ManimAnimationRenderer<br/>(threadpool, RENDER_TIMEOUT_SECONDS)"]
        INBOX["InboxRepository"]
        OUTBOX["OutboxRepository"]
        RELAY["OutboxRelay"]
    end

    ORCH["Orchestrator Service"]
    MQ[("RabbitMQ")]
    DB[("PostgreSQL: rendering-db")]
    FS[("Shared Docker Volume<br/>/shared/{project_id}/animations/")]

    ORCH -->|command render_scenes| MQ
    MQ --> CONSUMER
    CONSUMER --> INBOX
    INBOX --> DB
    CONSUMER --> UC
    UC -->|check .mp4 exists| FS
    UC --> RENDERER
    RENDERER --> REG
    RENDERER -->|write .mp4| FS
    CONSUMER --> OUTBOX
    OUTBOX --> DB
    RELAY --> DB
    RELAY -->|scene_render_started / scene_rendered / rendering_completed / rendering_failed| MQ
    MQ --> ORCH

    style Unit5 fill:#C8E6C9,stroke:#2E7D32,stroke-width:2px,color:#000
    style MQ fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
    style DB fill:#BBDEFB,stroke:#1565C0,stroke-width:2px,color:#000
    style FS fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
```
