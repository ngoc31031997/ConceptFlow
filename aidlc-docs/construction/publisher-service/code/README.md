# Unit 7: Publisher Service — Code Summary

## Overview
Xác thực OAuth 2.0 với YouTube (FR7.3, Story E1) và đăng video .mp4 kèm metadata lên YouTube (FR7.1, FR7.2 phần tiêu thụ — Story E3). Unit hybrid: REST (`/v1/auth/youtube/{start,callback}`, ngoài Saga) + AMQP consumer `publish_video` (queue `publisher.commands`, trong Saga) → publisher 1 event/command (`video_published`/`publish_failed`) qua PostgreSQL Outbox (ADR-0013).

## Structure (Hexagonal / Ports & Adapters, ADR-0002)
```
services/publisher/
├── domain/            # OAuthCredential, PublishRequest, PublishResult, errors, VideoPublisherPort, CredentialStorePort
├── application/        # PublishVideoUseCase, HandleOAuthCallbackUseCase
├── adapters/
│   ├── api/              # FastAPI router — /v1/auth/youtube/{start,callback}, /health
│   ├── youtube/          # GoogleOAuthFlow (google-auth-oauthlib), YouTubeVideoPublisher (google-api-python-client)
│   ├── persistence/      # Inbox/Outbox/Relay (ADR-0013, copied verbatim), credential_store.py (ADR-0016)
│   ├── messaging/        # AMQP consumer/producer
│   └── logging/          # correlation.py (saga_id / X-Request-ID)
├── main.py              # Composition root — create_app() factory (FastAPI + AMQP consumer + OutboxRelay in lifespan)
└── tests/
```

## Key Implementation Notes
- **First hybrid unit since Unit 2**: FastAPI REST layer (OAuth flow) alongside an AMQP consumer, both from one process — mirrors Content Plugin Service's composition root shape.
- **Sync/async boundary**: `PublishVideoUseCase`/`HandleOAuthCallbackUseCase` are fully synchronous (mirror Unit 6's `AssembleVideoUseCase`), called from async code via `asyncio.to_thread`. `CredentialStorePort` is therefore also synchronous — `PostgresCredentialStore` uses `psycopg2` (a blocking driver) rather than `asyncpg` for the single `oauth_credentials` table, since a worker thread has no event loop to await the asyncpg pool on. The Inbox/Outbox tables stay on `asyncpg` (only ever accessed from async consumer/relay code).
- **Proactive token refresh**: `YouTubeVideoPublisher.publish()` checks `credential.expires_at` (60s buffer) before every upload and refreshes via `refresh_token` if needed, persisting the refreshed credential through `CredentialStorePort.save()` — no reactive 401-triggered refresh.
- **No artifact-level idempotency**: unlike Unit 3/5/6, each successful `publish_video` creates a distinct new YouTube video — only message-level (Inbox) dedupe applies.
- **OAuth callback errors never enter the Saga**: a failed code exchange returns REST `400`, not an AMQP event (Business Rule 6) — Creator retries by revisiting `/v1/auth/youtube/start`.
- **Credential storage**: plaintext in Postgres (ADR-0016), justified by the single-user/local-only threat model.

## Test Coverage
- `tests/application/test_publish_video.py`, `test_handle_oauth_callback.py`: zero-trust validation (Rule 1, 2, 4), missing-credential handling, OAuth exchange error propagation
- `tests/adapters/test_youtube_publisher.py`: proactive refresh, upload success/failure — mocks `googleapiclient.discovery.build`, no real Google API call
- `tests/adapters/test_router.py`: FastAPI `TestClient`, OAuth start redirect + callback success/failure
- `tests/adapters/test_credential_store.py`: mocked `psycopg2.connect`
- `tests/adapters/test_messaging.py`, `test_persistence.py`, `test_relay.py`: Inbox/Outbox/consumer wiring, mirrors Unit 2/3/4/5/6

## Traceability
- Stories: E1, E2 (consume only), E3
- Requirements: FR7.1, FR7.2, FR7.3
- Design: `aidlc-docs/construction/publisher-service/{low-level-design,functional-design,nfr-requirements,nfr-design,infrastructure-design}/`
- ADR: `aidlc-docs/decisions/ADR-0016-oauth-credential-storage-plaintext.md`
