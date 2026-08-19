# Logical Components — Unit 4: Script Processing Service

## Components
| Component | Type | Purpose |
|---|---|---|
| AMQP Consumer (`aio-pika`) | Message consumer | Nhận `parse_script` (queue `script_processing.commands`), auto-reconnect built-in |
| `MarkdownScriptParser` | In-process parser | Implement `ScriptParserPort`, parse cú pháp Markdown (ADR-0011) |
| PostgreSQL (`script-processing-db`) | Database | `outbox_events`/`processed_messages` (ADR-0013) |
| `InboxRepository`/`OutboxRepository` | Persistence adapter | Dedupe message + enqueue event, transactional |
| `OutboxRelay` | Background poller | Publish event chưa gửi tới `orchestrator.events` |

## No External Infrastructure Components
Không cần cache (không có gì để cache), không cần circuit breaker riêng (chỉ 2 dependency hạ tầng: RabbitMQ, PostgreSQL — cả hai đều có cơ chế resilience built-in tương ứng của `aio-pika`/`asyncpg`).

## Diagram

```mermaid
flowchart TB
    subgraph Unit4["Script Processing Service (Python)"]
        CONSUMER["AMQP Consumer<br/>(parse_script)"]
        UC["ParseScriptUseCase"]
        PARSER["MarkdownScriptParser"]
        INBOX["InboxRepository"]
        OUTBOX["OutboxRepository"]
        RELAY["OutboxRelay"]
    end

    ORCH["Orchestrator Service"]
    MQ[("RabbitMQ")]
    DB[("PostgreSQL: script-processing-db")]

    ORCH -->|command parse_script| MQ
    MQ --> CONSUMER
    CONSUMER --> INBOX
    INBOX --> DB
    CONSUMER --> UC
    UC --> PARSER
    CONSUMER --> OUTBOX
    OUTBOX --> DB
    RELAY --> DB
    RELAY -->|script_parsed / parse_failed| MQ
    MQ --> ORCH

    style Unit4 fill:#C8E6C9,stroke:#2E7D32,stroke-width:2px,color:#000
    style MQ fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
    style DB fill:#BBDEFB,stroke:#1565C0,stroke-width:2px,color:#000
```
