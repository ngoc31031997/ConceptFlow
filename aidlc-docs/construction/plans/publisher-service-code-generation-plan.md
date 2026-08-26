# Code Generation Plan — Unit 7: Publisher Service

## Unit Context
- **Stories**: E1, E2 (consume only), E3
- **Dependencies**: Unit 1 (RabbitMQ) — đã hoàn tất; KHÔNG phụ thuộc trực tiếp unit nào khác
- **Interfaces**: REST `GET /v1/auth/youtube/{start,callback}` + `/health`; AMQP consumer `publish_video` (queue `publisher.commands`) → 1 event/command (`video_published`/`publish_failed`) qua Outbox (ADR-0013)
- **Owned entities**: `outbox_events`/`processed_messages` + `oauth_credentials` (Postgres `publisher-db`)

## Coding Standards (nhất quán Unit 2/3/4/5/6)
- **Naming**: Python snake_case cho function/variable, PascalCase cho class
- **SOLID**: Bắt buộc — `PublishVideoUseCase` phụ thuộc `VideoPublisherPort`/`CredentialStorePort` abstraction
- **Documentation**: Python docstring (Google style), giải thích WHY
- **Linting**: `ruff`

## Steps

- [x] **Step 1 — Project Structure Setup**: Tạo `services/publisher/` (domain/, application/, adapters/{api,messaging,persistence,youtube,logging}/, main.py, tests/), `requirements.txt`, `requirements-dev.txt`, `pyproject.toml`, `Dockerfile`
- [x] **Step 2 — Business Logic Generation (domain/)**: `models.py` (`OAuthCredential`, `PublishRequest`, `PublishResult`), `errors.py` (`MissingCredentialError`, `UploadError`, `InvalidPublishRequestError`), `ports.py` (`VideoPublisherPort`, `CredentialStorePort`)
- [x] **Step 3 — Business Logic Generation (application/)**: `publish_video.py` (`PublishVideoUseCase` — zero-trust validate, credential lookup, delegate to publisher), `handle_oauth_callback.py` (`HandleOAuthCallbackUseCase` — exchange code, save credential)
- [x] **Step 4 — Business Logic Unit Testing**: `tests/application/` — `FakeVideoPublisher`, `FakeCredentialStore`, `FakeOAuthFlow`, cover Business Rule 1-6
- [x] **Step 5 — OAuth Flow Adapter Generation**: `adapters/youtube/oauth_flow.py` (`GoogleOAuthFlow` — wrap `google-auth-oauthlib`, `build_authorization_url()`, `exchange_code()`)
- [x] **Step 6 — YouTube Publisher Adapter Generation**: `adapters/youtube/youtube_publisher.py` (`YouTubeVideoPublisher` implements `VideoPublisherPort` — proactive token refresh, resumable upload via `google-api-python-client`, threadpool + `UPLOAD_TIMEOUT_SECONDS`)
- [x] **Step 7 — Publisher/OAuth Unit Testing**: `tests/adapters/test_youtube_publisher.py` — mock Google API client calls, không gọi YouTube thật
- [x] **Step 8 — Persistence Adapter Generation**: `adapters/persistence/{db,inbox,outbox,relay}.py` (copy nguyên mẫu Unit 2/3/4/5/6, ADR-0013, `db.py` mở rộng thêm bảng `oauth_credentials`), `adapters/persistence/credential_store.py` (`PostgresCredentialStore` implements `CredentialStorePort`)
- [x] **Step 9 — Logging Adapter Generation**: `adapters/logging/correlation.py` (copy nguyên mẫu Unit 2 — hỗ trợ cả `saga_id` AMQP và `X-Request-ID` REST)
- [x] **Step 10 — REST API Layer Generation**: `adapters/api/router.py` (FastAPI routes `/v1/auth/youtube/{start,callback}`, `/health`), `adapters/api/schemas.py` (Pydantic response models)
- [x] **Step 11 — REST API Layer Unit Testing**: `tests/adapters/test_router.py` — FastAPI `TestClient`, mock use cases
- [x] **Step 12 — Messaging Layer Generation**: `adapters/messaging/consumer.py` (`PublishVideoCommandHandler` — inbox check → use case → 1 event + inbox mark cùng transaction → ack), `adapters/messaging/producer.py` (2 envelope builders: `video_published`, `publish_failed`)
- [x] **Step 13 — Messaging Layer Unit Testing**: `tests/adapters/test_messaging.py` — mock AMQP + `FakePool`
- [x] **Step 14 — Persistence/Relay/CredentialStore Unit Testing**: `tests/adapters/test_persistence.py`, `test_relay.py` (copy nguyên mẫu Unit 2/3/4/5/6), `test_credential_store.py` (mới)
- [x] **Step 15 — Composition Root**: `main.py` — `create_app()` factory (FastAPI + AMQP consumer + OutboxRelay trong lifespan, mirror Unit 2)
- [x] **Step 16 — Documentation Generation**: Cập nhật `README.md` gốc (bao gồm hướng dẫn đăng ký Google OAuth Client — Infrastructure Design Question 5), tạo `aidlc-docs/construction/publisher-service/code/README.md`
- [x] **Step 17 — Deployment Artifacts**: Thêm service `publisher` + `publisher-db` + volume vào `docker-compose.yml` gốc, thêm biến `GOOGLE_OAUTH_*` vào `.env.example`

**Không áp dụng**: Repository Layer riêng ngoài Outbox/Inbox/credential_store, Frontend Components, Database Migration Scripts.

**Note về Google API thật**: Unit test dùng mock cho `google-api-python-client`/`google-auth-oauthlib` calls (`FakeVideoPublisher`/`FakeOAuthFlow`) nên KHÔNG cần Google OAuth Client thật hay kết nối mạng để chạy test suite. Việc xác thực/upload thật (integration/manual test) cần Creator tự đăng ký OAuth Client và cấu hình `.env`, ngoài phạm vi test tự động ở bước này.
