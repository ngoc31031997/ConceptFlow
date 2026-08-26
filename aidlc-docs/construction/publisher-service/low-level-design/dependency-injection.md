# Dependency Injection — Unit 7: Publisher Service

## Mechanism
Constructor injection thủ công — không đổi so với Unit 2/3/4/5/6.

## What Gets Injected vs Constructed Directly
- **Injected (abstraction)**: `VideoPublisherPort` (`YouTubeVideoPublisher`) → `PublishVideoUseCase`; `CredentialStorePort` (`PostgresCredentialStore`) → cả `PublishVideoUseCase` và `HandleOAuthCallbackUseCase`. Cho phép thay platform đăng video hoặc nơi lưu credential mà không sửa `application/`.
- **Constructed directly**: `InboxRepository`/`OutboxRepository`/`OutboxRelay` (Postgres, ADR-0013, mirror Unit 2/3/4/5/6), `oauth_flow.py` helper (wrap `google-auth-oauthlib`, không cần abstraction riêng vì chỉ có 1 implementation khả dĩ — Google OAuth).

## Composition Root
`main.py`:
1. Kết nối PostgreSQL, khởi tạo `InboxRepository`, `OutboxRepository`, `PostgresCredentialStore`.
2. Khởi tạo `oauth_flow` (Google client secrets từ env var).
3. Khởi tạo `YouTubeVideoPublisher(timeout_seconds=env)`.
4. Khởi tạo `PublishVideoUseCase(publisher=youtube_publisher, credential_store=credential_store)`, `HandleOAuthCallbackUseCase(oauth_flow, credential_store)`.
5. Tạo FastAPI app, wire router `/v1/auth/youtube/{start,callback}` qua `Depends()`.
6. Kết nối RabbitMQ, wire `PublishVideoCommandHandler(publish_video_uc, pool, inbox, outbox)`, đăng ký consumer cho queue `publisher.commands`.
7. Khởi động `OutboxRelay` như background task (FastAPI lifespan event).
8. FastAPI serving (readiness qua `GET /health`, unversioned — mirror Content Plugin Service's health check, khác Unit 3/4/5/6's sentinel file vì unit này CÓ REST endpoint để serve health check chuẩn HTTP).

## Wiring Diagram
```
main.py
  ├── PostgresCredentialStore (Postgres, oauth_credentials table)
  │     ├── injected into → HandleOAuthCallbackUseCase
  │     └── injected into → PublishVideoUseCase
  ├── YouTubeVideoPublisher (implements VideoPublisherPort)
  │     └── injected into → PublishVideoUseCase
  ├── InboxRepository, OutboxRepository (Postgres, ADR-0013)
  │     └── injected into → PublishVideoCommandHandler (AMQP consumer)
  │           └── consumes command "publish_video" (queue publisher.commands)
  │           └── ghi event "video_published"/"publish_failed" (1 row/command) vào Outbox, cùng transaction với Inbox mark
  ├── FastAPI app
  │     └── router /v1/auth/youtube/start, /v1/auth/youtube/callback → HandleOAuthCallbackUseCase
  └── OutboxRelay (background task, FastAPI lifespan)
        └── poll Outbox chưa publish → publish qua producer.py → đánh dấu published_at
```

## Testability
`PublishVideoUseCase`/`HandleOAuthCallbackUseCase` chỉ phụ thuộc abstraction (`VideoPublisherPort`, `CredentialStorePort`) — unit test dùng `FakeVideoPublisher`/`FakeCredentialStore`, không cần Google API/Postgres thật.
