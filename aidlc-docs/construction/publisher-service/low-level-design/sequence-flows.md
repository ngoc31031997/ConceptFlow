# Sequence Flows — Unit 7: Publisher Service

## Flow 1: OAuth Authentication (REST, ngoài Saga, Story E1)

```mermaid
sequenceDiagram
    participant CREATOR as Creator (GUI)
    participant API as adapters/api/router.py
    participant FLOW as adapters/youtube/oauth_flow.py
    participant GOOGLE as Google OAuth
    participant UC as HandleOAuthCallbackUseCase
    participant STORE as PostgresCredentialStore

    CREATOR->>API: GET /v1/auth/youtube/start
    API->>FLOW: build_authorization_url()
    FLOW-->>API: authorization_url
    API-->>CREATOR: 302 redirect to Google consent screen
    CREATOR->>GOOGLE: grants permission
    GOOGLE-->>CREATOR: redirect to /v1/auth/youtube/callback?code=...

    CREATOR->>API: GET /v1/auth/youtube/callback?code=...
    API->>UC: handle(code)
    UC->>FLOW: exchange_code(code)
    FLOW->>GOOGLE: token exchange
    GOOGLE-->>FLOW: access_token, refresh_token, expires_in
    FLOW-->>UC: OAuthCredential
    UC->>STORE: save(credential)
    STORE-->>UC: OK
    UC-->>API: success
    API-->>CREATOR: 200 { connected: true }
```

## Flow 2: Successful Publish (AMQP, trong Saga, Story E3)

```mermaid
sequenceDiagram
    participant ORCH as Orchestrator
    participant MQ as RabbitMQ
    participant CONSUMER as adapters/messaging/consumer.py
    participant INBOX as InboxRepository
    participant UC as PublishVideoUseCase
    participant STORE as PostgresCredentialStore
    participant PUB as YouTubeVideoPublisher
    participant YT as YouTube Data API
    participant OUTBOX as OutboxRepository
    participant RELAY as OutboxRelay

    ORCH->>MQ: command publish_video (video_path, title, description, tags, visibility)
    MQ->>CONSUMER: deliver
    CONSUMER->>INBOX: has_processed(message_id)?
    INBOX-->>CONSUMER: no

    CONSUMER->>UC: publish(request)
    UC->>STORE: get()
    STORE-->>UC: OAuthCredential (found)
    UC->>PUB: publish(request, credential) — trong ThreadPoolExecutor
    PUB->>PUB: refresh access_token if expired (persists via STORE.save())
    PUB->>YT: videos().insert() — resumable upload
    YT-->>PUB: video_id
    PUB-->>UC: PublishResult(youtube_video_url)

    CONSUMER->>OUTBOX: enqueue video_published (cùng transaction với Inbox mark)
    CONSUMER->>INBOX: mark_processed(message_id)
    CONSUMER->>MQ: ack

    RELAY->>MQ: publish video_published (poll định kỳ)
    MQ-->>ORCH: deliver video_published
```

## Flow 3: Publish Failure — Missing Credential

```mermaid
sequenceDiagram
    participant CONSUMER as adapters/messaging/consumer.py
    participant UC as PublishVideoUseCase
    participant STORE as PostgresCredentialStore
    participant OUTBOX as OutboxRepository

    CONSUMER->>UC: publish(request)
    UC->>STORE: get()
    STORE-->>UC: None (chưa xác thực OAuth)
    UC-->>CONSUMER: raise MissingCredentialError
    CONSUMER->>OUTBOX: enqueue publish_failed (error_message: "not authenticated, complete Story E1 first")
    Note over CONSUMER: cùng transaction với Inbox mark, ack như bình thường
```

## Flow 4: Publish Failure — Upload Error (Network/Timeout)

```mermaid
sequenceDiagram
    participant CONSUMER as adapters/messaging/consumer.py
    participant UC as PublishVideoUseCase
    participant PUB as YouTubeVideoPublisher
    participant OUTBOX as OutboxRepository

    CONSUMER->>UC: publish(request)
    UC->>PUB: publish(request, credential)
    PUB-->>UC: raise UploadError (non-2xx response / timeout sau UPLOAD_TIMEOUT_SECONDS)
    UC-->>CONSUMER: propagate UploadError
    CONSUMER->>OUTBOX: enqueue publish_failed
    Note over CONSUMER: transient — Orchestrator/GUI có thể gửi lại publish_video<br/>(metadata giữ nguyên ở GUI, Story E3's AC), có thể tạo video trùng<br/>trên YouTube nếu upload trước đó thực ra đã thành công (Question 7, rủi ro chấp nhận được)
```

## Flow 5: Idempotent Redelivery (Message-Level, Inbox)
Giống Unit 2/3/4/5/6 — `message_id` đã xử lý (requeue) → `InboxRepository.has_processed()` trả `true` → ack ngay, không publish lại event, không upload lại.
