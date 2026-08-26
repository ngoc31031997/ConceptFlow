# Tech Stack Decisions — Unit 7: Publisher Service

## Language/Runtime: Python 3.12
- **Rationale**: Nhất quán ADR-0009 (Python cho hầu hết service).

## Web Framework: FastAPI
- **Ecosystem**: Nhất quán Unit 2 (ADR-0003) — unit đầu tiên từ Unit 3 trở đi cần lại REST endpoint (OAuth flow, không thể message-driven vì phải tương tác trực tiếp với trình duyệt Creator + Google's redirect flow).
- **Scope**: Chỉ 2 route nghiệp vụ (`/v1/auth/youtube/start`, `/v1/auth/youtube/callback`) + `/health` — tối giản hơn nhiều so với Content Plugin Service.

## YouTube Integration: google-api-python-client + google-auth-oauthlib
- **Ecosystem**: Thư viện chính thức của Google cho OAuth 2.0 Authorization Code flow và YouTube Data API v3 (`technology-direction.md`).
- **Upload**: `videos().insert()` với `MediaFileUpload(resumable=True)` (Low-Level Design Question 5).
- **Performance**: Network I/O-bound, chạy trong `ThreadPoolExecutor` (không có async API chính thức từ Google's client library).

## Messaging Client: aio-pika
Nhất quán Unit 1/2/3/4/5/6.

## Database Client: asyncpg
Nhất quán Unit 2/3/4/5/6 (ADR-0013). Bảng bổ sung `oauth_credentials` (plaintext, ADR-0016) bên cạnh Outbox/Inbox chuẩn.

## Testing
- `pytest` + `pytest-asyncio` (`asyncio_mode = "auto"`) — nhất quán.
- `httpx` (FastAPI's `TestClient` dependency) cho test REST layer, mirror Unit 2.
- `ruff` cho lint/format.
- Test cho YouTube API/OAuth thực tế (integration, không phải unit test thuần) sẽ cần cân nhắc riêng ở Code Generation — unit test business logic dùng `FakeVideoPublisher`/`FakeCredentialStore`/fake OAuth flow, không gọi Google API thật.

Không cần ADR riêng cho các lựa chọn framework/testing — hệ quả trực tiếp của ADR-0003/ADR-0009/ADR-0013. Google API client library không phải trade-off cạnh tranh (đã xác định từ Inception — "Google API Python Client cho YouTube Integration", `technology-direction.md`), không cần ADR riêng.
