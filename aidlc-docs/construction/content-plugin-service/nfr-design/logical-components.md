# Logical Components — Unit 2: Content Plugin Service

## Components
| Component | Type | Purpose |
|---|---|---|
| `ContentPluginRegistry` | In-memory registry | Lưu trữ plugin đã discover, tra cứu theo `plugin_id` |
| `IdempotencyStore` | In-memory `set[message_id]` + TTL cleanup task | Dedupe message AMQP đã xử lý |
| FastAPI app | HTTP server | Expose `GET /v1/plugins` |
| AMQP Consumer (`aio-pika`) | Message consumer | Nhận `classify_scenes`, auto-reconnect built-in |
| AMQP Producer (`aio-pika`) | Message producer | Publish `scenes_classified`/`classification_failed` |

## No External Infrastructure Components
Không cần cache ngoài (Redis), không cần database, không cần circuit breaker riêng — mọi state là in-memory, mọi resilience dựa vào cơ chế có sẵn của `aio-pika` + RabbitMQ (Unit 1).

## Diagram

```mermaid
flowchart TB
    subgraph Unit2["Content Plugin Service (Python/FastAPI)"]
        API["FastAPI: GET /v1/plugins"]
        CONSUMER["AMQP Consumer"]
        PRODUCER["AMQP Producer"]
        REG["ContentPluginRegistry<br/>(in-memory)"]
        IDEM["IdempotencyStore<br/>(in-memory set + TTL)"]
        UC["Use Cases<br/>(ListPlugins, ClassifyScene)"]
    end

    MQ[("RabbitMQ")]
    GW["API Gateway"]

    GW -->|REST| API
    API --> UC
    MQ -->|classify_scenes| CONSUMER
    CONSUMER --> IDEM
    CONSUMER --> UC
    UC --> REG
    CONSUMER --> PRODUCER
    PRODUCER -->|scenes_classified /<br/>classification_failed| MQ

    style Unit2 fill:#C8E6C9,stroke:#2E7D32,stroke-width:2px,color:#000
    style MQ fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
```
