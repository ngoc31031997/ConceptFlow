# Logical Components — Unit 7: Publisher Service

## Components
| Component | Type | Purpose |
|---|---|---|
| FastAPI app | REST server | `/v1/auth/youtube/{start,callback}`, `/health` |
| AMQP Consumer (`aio-pika`) | Message consumer | Nhận `publish_video` (queue `publisher.commands`) |
| `oauth_flow.py` | OAuth 2.0 wrapper | `google-auth-oauthlib`'s Authorization Code flow |
| `YouTubeVideoPublisher` | Upload engine wrapper | Implement `VideoPublisherPort`, resumable upload trong threadpool, `UPLOAD_TIMEOUT_SECONDS`, proactive token refresh |
| `PostgresCredentialStore` | Persistence adapter | Implement `CredentialStorePort`, đọc/ghi `oauth_credentials` (1 row, plaintext — ADR-0016) |
| Shared Docker Volume | File storage | Đọc video .mp4 output (từ Unit 6), read-only cho unit này |
| PostgreSQL (`publisher-db`) | Database | `outbox_events`/`processed_messages` (ADR-0013) + `oauth_credentials` |
| `InboxRepository`/`OutboxRepository` | Persistence adapter | Dedupe message + enqueue event (1 row/command, cùng transaction với Inbox mark) |
| `OutboxRelay` | Background poller | Publish event chưa gửi tới `orchestrator.events` |

## No External Infrastructure Components
Không cần cache ngoài (Redis), không cần circuit breaker riêng.

## Diagram

```mermaid
flowchart TB
    subgraph Unit7["Publisher Service (Python/FastAPI)"]
        API["FastAPI<br/>/v1/auth/youtube/{start,callback}"]
        FLOW["oauth_flow.py"]
        CONSUMER["AMQP Consumer<br/>(publish_video)"]
        UC["PublishVideoUseCase"]
        PUB["YouTubeVideoPublisher<br/>(threadpool, UPLOAD_TIMEOUT_SECONDS)"]
        STORE["PostgresCredentialStore"]
        INBOX["InboxRepository"]
        OUTBOX["OutboxRepository"]
        RELAY["OutboxRelay"]
    end

    CREATOR["Creator (browser)"]
    GOOGLE["Google OAuth / YouTube Data API"]
    ORCH["Orchestrator Service"]
    MQ[("RabbitMQ")]
    DB[("PostgreSQL: publisher-db")]
    FS[("Shared Docker Volume<br/>/shared/{project_id}/video/")]

    CREATOR-->|GET /v1/auth/youtube/start|API
    API-->FLOW
    FLOW-->|redirect + token exchange|GOOGLE
    API-->|save credential|STORE
    STORE-->DB

    ORCH-->|command publish_video|MQ
    MQ-->CONSUMER
    CONSUMER-->INBOX
    INBOX-->DB
    CONSUMER-->UC
    UC-->|get credential|STORE
    UC-->PUB
    PUB-->|refresh if expired|GOOGLE
    PUB-->|read video|FS
    PUB-->|resumable upload|GOOGLE
    CONSUMER-->OUTBOX
    OUTBOX-->DB
    RELAY-->DB
    RELAY-->|video_published / publish_failed|MQ
    MQ-->ORCH

    style Unit7 fill:#C8E6C9,stroke:#2E7D32,stroke-width:2px,color:#000
    style MQ fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
    style DB fill:#BBDEFB,stroke:#1565C0,stroke-width:2px,color:#000
    style FS fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
    style GOOGLE fill:#FFE0B2,stroke:#E65100,stroke-width:2px,color:#000
```
