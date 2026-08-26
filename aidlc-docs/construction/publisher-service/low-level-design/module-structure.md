# Module Structure — Unit 7: Publisher Service

## Layering (Hexagonal / Ports & Adapters — ADR-0002)

```
services/publisher/
├── domain/
│   ├── models.py                  # OAuthCredential, PublishRequest, PublishResult (value objects)
│   ├── errors.py                   # MissingCredentialError, UploadError, InvalidPublishRequestError (added at Functional Design, Rule 1/2/4)
│   └── ports.py                    # VideoPublisherPort, CredentialStorePort
├── application/
│   ├── publish_video.py            # PublishVideoUseCase
│   └── handle_oauth_callback.py    # HandleOAuthCallbackUseCase
├── adapters/
│   ├── api/
│   │   ├── router.py                # FastAPI router, /v1/auth/youtube/{start,callback} (ADR-0008)
│   │   └── schemas.py               # Pydantic response models (redirect has no body, but error responses do)
│   ├── messaging/
│   │   ├── consumer.py              # AMQP consumer for publish_video (queue publisher.commands)
│   │   └── producer.py              # Envelope builders: video_published, publish_failed
│   ├── persistence/
│   │   ├── db.py                     # PostgreSQL pool + schema bootstrap (ADR-0013, extends with oauth_credentials)
│   │   ├── inbox.py                  # InboxRepository (copied verbatim from Unit 2/3/4/5/6)
│   │   ├── outbox.py                 # OutboxRepository
│   │   ├── relay.py                  # OutboxRelay
│   │   └── credential_store.py       # PostgresCredentialStore implements CredentialStorePort
│   ├── youtube/
│   │   ├── oauth_flow.py             # OAuth 2.0 Authorization Code flow (google-auth-oauthlib)
│   │   └── youtube_publisher.py      # YouTubeVideoPublisher implements VideoPublisherPort
│   └── logging/
│       └── correlation.py            # saga_id (AMQP) / X-Request-ID (REST) injection vào log context
├── main.py                          # Composition root — FastAPI app + AMQP consumer + OutboxRelay startup
└── tests/
    ├── domain/
    ├── application/
    └── adapters/
```

## Dependency Direction
`adapters/` → `application/` → `domain/`. `domain/` không import Google API client/FastAPI/aio-pika/asyncpg. `application/` chỉ phụ thuộc `domain/` qua `VideoPublisherPort`/`CredentialStorePort` (abstraction) — không biết chi tiết YouTube Data API/Postgres cụ thể. `adapters/youtube/youtube_publisher.py` implement `domain/ports.py::VideoPublisherPort`; `adapters/persistence/credential_store.py` implement `domain/ports.py::CredentialStorePort` — cho phép đổi platform đăng video hoặc nơi lưu credential sau này mà không sửa business logic.

Unit này là unit ĐẦU TIÊN kể từ Unit 3 có cả REST (`api/`) VÀ AMQP (`messaging/`) layer — mirror `adapters/api/` từ Unit 2, `adapters/persistence/` (Inbox/Outbox) từ Unit 2/3/4/5/6, nhưng thiết kế Inbox/Outbox NGAY TỪ ĐẦU (không retrofit như Unit 2).

## Module Responsibilities

| Module | Responsibility |
|---|---|
| `domain/models.py` | `OAuthCredential` (`access_token`, `refresh_token`, `expires_at`, `channel_id`), `PublishRequest` (`project_id`, `video_path`, `title`, `description`, `tags`, `visibility`), `PublishResult` (`youtube_video_url`) |
| `domain/errors.py` | `MissingCredentialError` (chưa xác thực OAuth), `UploadError` (lỗi mạng/YouTube API/timeout khi upload) |
| `domain/ports.py` | `VideoPublisherPort` (abstract: `publish(request, credential) -> PublishResult`), `CredentialStorePort` (abstract: `get() -> OAuthCredential \| None`, `save(credential) -> None`) |
| `application/publish_video.py` | `PublishVideoUseCase(publisher: VideoPublisherPort, credential_store: CredentialStorePort)` — lấy credential đã lưu (raise `MissingCredentialError` nếu chưa có), gọi `publisher.publish(...)` |
| `application/handle_oauth_callback.py` | `HandleOAuthCallbackUseCase(oauth_flow, credential_store: CredentialStorePort)` — đổi `code` lấy token, lưu qua `credential_store.save(...)` |
| `adapters/api/router.py` | FastAPI routes `GET /v1/auth/youtube/start` (redirect tới Google consent screen), `GET /v1/auth/youtube/callback` (gọi `HandleOAuthCallbackUseCase`) |
| `adapters/messaging/consumer.py` | Consume `publish_video`; gọi `PublishVideoUseCase`; publish `video_published`/`publish_failed` (1 event/command, cùng transaction với Inbox mark — mirror Unit 6) |
| `adapters/youtube/oauth_flow.py` | Wrap `google-auth-oauthlib`'s `Flow` — tạo authorization URL (`start`), đổi `code` lấy token (`callback`) |
| `adapters/youtube/youtube_publisher.py` | `YouTubeVideoPublisher` — implement `VideoPublisherPort`; tự refresh `access_token` nếu hết hạn (dùng `refresh_token`); gọi `google-api-python-client`'s `videos().insert()` với `MediaFileUpload(resumable=True)` trong `ThreadPoolExecutor`, timeout `UPLOAD_TIMEOUT_SECONDS` |
| `adapters/persistence/credential_store.py` | `PostgresCredentialStore` — implement `CredentialStorePort`; đọc/ghi bảng `oauth_credentials` (1 row duy nhất, single-user, plaintext — ADR-0016) |
| `adapters/logging/correlation.py` | Gắn `saga_id` (AMQP) hoặc `X-Request-ID` (REST) vào log context |
