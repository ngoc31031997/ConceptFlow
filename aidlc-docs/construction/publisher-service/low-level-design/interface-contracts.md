# Interface Contracts — Unit 7: Publisher Service

## REST API (versioned per ADR-0008, ngoài Saga)

### `GET /v1/auth/youtube/start`
- **Called by**: Web GUI (via API Gateway, Unit 9)
- **Response**: `302 Found`, redirect tới Google's OAuth consent screen (authorization URL do `oauth_flow.py` tạo)
- **Correlation ID**: Header `X-Request-ID` — tự sinh nếu Gateway không gửi.

### `GET /v1/auth/youtube/callback`
- **Called by**: Google (redirect sau khi Creator cấp quyền)
- **Query params**: `code` (authorization code), `state` (CSRF protection token)
- **Response 200**: `{ "connected": true }` (đổi `code` lấy token thành công, đã lưu vào `oauth_credentials`)
- **Response 400**: `{ "connected": false, "error": "..." }` (code không hợp lệ/hết hạn)
- **Correlation ID**: Header `X-Request-ID`.

### `GET /health` (unversioned, mirror Content Plugin Service)
- **Response 200**: `{ "status": "ok" }`

## AMQP Interface (theo `component-methods.md`)

### Consumer: command `publish_video` (queue `publisher.commands`)
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "uuid",
  "schema_version": "1.0",
  "timestamp": "ISO8601",
  "payload": {
    "video_path": "/shared/{project_id}/video/final.mp4",
    "title": "string",
    "description": "string | null",
    "tags": ["string"] ,
    "visibility": "public | unlisted | private"
  }
}
```
- **Correlation ID**: `saga_id` trong envelope.

### Producer: event `video_published` (→ `orchestrator.events`, routing key `orchestrator`)
```json
{
  "...envelope...": "...",
  "payload": {
    "event_type": "video_published",
    "youtube_video_url": "string"
  }
}
```

### Producer: event `publish_failed`
```json
{
  "...envelope...": "...",
  "payload": {
    "event_type": "publish_failed",
    "error_message": "string"
  }
}
```

## Event Sequence Per Command
Đúng 1 Outbox row/command (mirror Unit 6) — `publish_video` là 1 thao tác duy nhất, không có progress event trung gian.

## Delivery Guarantee & Idempotency
At-least-once (kế thừa Unit 1). Inbox (`processed_messages`) dedupe `message_id` ở mức COMMAND, cùng transaction với Outbox row kết quả — mirror Unit 2/3/4/5/6. KHÔNG có artifact-level idempotency (Question 7) — mỗi `publish_video` thành công tạo 1 video MỚI trên YouTube; retry-với-metadata-giữ-nguyên là trách nhiệm của GUI/Orchestrator, không phải Publisher Service.

## Correlation ID
`saga_id` từ AMQP envelope cho luồng `publish_video` (thuộc Saga Publish). `X-Request-ID` cho luồng OAuth REST (ngoài Saga, tương tác trực tiếp Creator ↔ Google).

## Internal Port Contracts (domain/ports.py)
```python
class VideoPublisherPort(ABC):
    @abstractmethod
    def publish(self, request: PublishRequest, credential: OAuthCredential) -> PublishResult:
        """Uploads request.video_path to YouTube with the given metadata.

        Raises:
            domain.errors.UploadError: network/API failure or timeout.
        """

class CredentialStorePort(ABC):
    @abstractmethod
    def get(self) -> OAuthCredential | None:
        """Returns the stored credential, or None if never authenticated (Story E1)."""

    @abstractmethod
    def save(self, credential: OAuthCredential) -> None:
        """Persists (upserts) the single-user OAuth credential."""
```

## OAuth Credential Refresh (Question 3)
`YouTubeVideoPublisher.publish()` checks `credential.expires_at` before calling the YouTube API; if expired, refreshes via `refresh_token` (no Creator interaction needed — Story E1's AC) and persists the refreshed credential via `CredentialStorePort.save()` before proceeding.

## Upload Timeout (Question 6)
Đọc từ biến môi trường `UPLOAD_TIMEOUT_SECONDS` (mặc định 600s = 10 phút). Vượt timeout hoặc lỗi mạng → `UploadError` → `publish_failed`.
