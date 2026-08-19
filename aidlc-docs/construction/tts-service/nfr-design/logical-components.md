# Logical Components — Unit 3: TTS Service

**Revision (2026-08-07, ADR-0014, ADR-0013)**: FastAPI/REST replaced by AMQP consumer/producer; PostgreSQL (Inbox/Outbox) added.

## Components
| Component | Type | Purpose |
|---|---|---|
| AMQP Consumer (`aio-pika`) | Message consumer | Nhận `synthesize_speech` (queue `tts.commands`), auto-reconnect built-in |
| `PiperTTSAdapter` | In-process TTS engine wrapper | Implement `TTSEnginePort`, chạy synthesis trong threadpool, 60s timeout nội bộ — không đổi |
| Voice model in-process cache | In-memory dict (`language -> model instance`) | Không đổi |
| Shared Docker Volume | File storage | Lưu file audio `.wav`, cơ chế idempotency artifact-level — không đổi |
| PostgreSQL (`tts-db`) | Database | `outbox_events`/`processed_messages` (ADR-0013) |
| `InboxRepository`/`OutboxRepository` | Persistence adapter | Dedupe message + enqueue event, transactional |
| `OutboxRelay` | Background poller | Publish event chưa gửi tới `orchestrator.events` |

## No External Infrastructure Components
Không cần cache ngoài (Redis), không cần circuit breaker riêng. **Đổi**: nay CÓ database (Postgres, ADR-0013) và CÓ tham gia RabbitMQ (ADR-0014) — trước đây cả hai đều "N/A".

## Diagram

```mermaid
flowchart TB
    subgraph Unit3["TTS Service (Python)"]
        CONSUMER["AMQP Consumer<br/>(synthesize_speech)"]
        UC["SynthesizeSpeechBatchUseCase"]
        ADAPTER["PiperTTSAdapter<br/>(threadpool, 60s timeout)"]
        CACHE["Voice Model Cache<br/>(in-memory, loaded at startup)"]
        INBOX["InboxRepository"]
        OUTBOX["OutboxRepository"]
        RELAY["OutboxRelay"]
    end

    ORCH["Orchestrator Service"]
    MQ[("RabbitMQ")]
    DB[("PostgreSQL: tts-db")]
    FS[("Shared Docker Volume<br/>/shared/{project_id}/audio/")]

    ORCH -->|command synthesize_speech| MQ
    MQ --> CONSUMER
    CONSUMER --> INBOX
    INBOX --> DB
    CONSUMER --> UC
    UC -->|check file exists| FS
    UC --> ADAPTER
    ADAPTER --> CACHE
    ADAPTER -->|write .wav| FS
    CONSUMER --> OUTBOX
    OUTBOX --> DB
    RELAY --> DB
    RELAY -->|speech_synthesized / synthesis_failed| MQ
    MQ --> ORCH

    style Unit3 fill:#C8E6C9,stroke:#2E7D32,stroke-width:2px,color:#000
    style MQ fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
    style DB fill:#BBDEFB,stroke:#1565C0,stroke-width:2px,color:#000
    style FS fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
```
