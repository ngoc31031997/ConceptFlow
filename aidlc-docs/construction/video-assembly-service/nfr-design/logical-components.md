# Logical Components — Unit 6: Video Assembly Service

## Components
| Component | Type | Purpose |
|---|---|---|
| AMQP Consumer (`aio-pika`) | Message consumer | Nhận `assemble_video` (queue `video_assembly.commands`) |
| `MediaFormatInspector` | ffprobe wrapper | Pre-check codec/resolution/framerate đồng nhất giữa các animation clip (Functional Design Question 3) |
| `FfmpegVideoAssembler` | Assembly engine wrapper | Implement `VideoAssemblerPort`, chạy ffmpeg (mux → concat → overlay nhạc nền) trong threadpool, `ASSEMBLY_TIMEOUT_SECONDS` |
| Shared Docker Volume | File storage | Đọc animation clip + audio clip input, ghi file trung gian `_tmp/` + video output `final.mp4`, cơ chế idempotency artifact-level |
| PostgreSQL (`video-assembly-db`) | Database | `outbox_events`/`processed_messages` (ADR-0013) |
| `InboxRepository`/`OutboxRepository` | Persistence adapter | Dedupe message + enqueue event (1 row/command, cùng transaction với Inbox mark) |
| `OutboxRelay` | Background poller | Publish event chưa gửi tới `orchestrator.events` |

## No External Infrastructure Components
Không cần cache ngoài (Redis), không cần circuit breaker riêng.

## Diagram

```mermaid
flowchart TB
    subgraph Unit6["Video Assembly Service (Python)"]
        CONSUMER["AMQP Consumer<br/>(assemble_video)"]
        UC["AssembleVideoUseCase"]
        INSPECTOR["MediaFormatInspector<br/>(ffprobe pre-check)"]
        ASM["FfmpegVideoAssembler<br/>(threadpool, ASSEMBLY_TIMEOUT_SECONDS)"]
        INBOX["InboxRepository"]
        OUTBOX["OutboxRepository"]
        RELAY["OutboxRelay"]
    end

    ORCH["Orchestrator Service"]
    MQ[("RabbitMQ")]
    DB[("PostgreSQL: video-assembly-db")]
    FS[("Shared Docker Volume<br/>/shared/{project_id}/video/")]

    ORCH -->|command assemble_video| MQ
    MQ --> CONSUMER
    CONSUMER --> INBOX
    INBOX --> DB
    CONSUMER --> UC
    UC -->|check final.mp4 exists| FS
    UC --> ASM
    ASM --> INSPECTOR
    INSPECTOR -->|read clips| FS
    ASM -->|mux + concat + overlay, write final.mp4| FS
    CONSUMER --> OUTBOX
    OUTBOX --> DB
    RELAY --> DB
    RELAY -->|video_assembled / assembly_failed| MQ
    MQ --> ORCH

    style Unit6 fill:#C8E6C9,stroke:#2E7D32,stroke-width:2px,color:#000
    style MQ fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
    style DB fill:#BBDEFB,stroke:#1565C0,stroke-width:2px,color:#000
    style FS fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
```
