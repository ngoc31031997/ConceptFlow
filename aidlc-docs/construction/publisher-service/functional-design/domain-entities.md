# Domain Entities — Unit 7: Publisher Service

## `OAuthCredential`
| Field | Type | Notes |
|---|---|---|
| `access_token` | `str` | Dùng cho YouTube API calls |
| `refresh_token` | `str` | Dùng để refresh `access_token` khi hết hạn (Rule 3) |
| `expires_at` | `datetime` | Thời điểm `access_token` hết hạn |
| `channel_id` | `str` | YouTube channel của Creator (từ token response) |

## `PublishRequest`
| Field | Type | Notes |
|---|---|---|
| `project_id` | `str` | Không rỗng |
| `video_path` | `str` | Không rỗng, phải tồn tại trên shared volume (Rule 1) |
| `title` | `str` | Không rỗng (Rule 1, 2) |
| `description` | `str \| None` | Optional |
| `tags` | `list[str]` | Optional, mặc định `[]` |
| `visibility` | `str` | Phải là `public`/`unlisted`/`private` (Rule 4), không có default |

## `PublishResult`
| Field | Type | Notes |
|---|---|---|
| `youtube_video_url` | `str` | KHÔNG bổ sung field khác (Rule 5) |

## Errors (Domain)
| Error | Trigger | Kênh phản hồi |
|---|---|---|
| `InvalidPublishRequestError` (mới, Functional Design) | Zero-trust validation thất bại: `video_path` thiếu/không tồn tại, `title` rỗng, `visibility` không hợp lệ (Rule 1, 2, 4) | `publish_failed` (AMQP) |
| `MissingCredentialError` | Chưa xác thực OAuth (Story E1 chưa hoàn thành) | `publish_failed` (AMQP) |
| `UploadError` | Lỗi mạng/YouTube API/timeout khi upload | `publish_failed` (AMQP) |
| OAuth token exchange failure (Google-side, không có domain error riêng — bắt tại `oauth_flow.py`, propagate lên `HandleOAuthCallbackUseCase`) | `code` sai/hết hạn/đã dùng (Rule 6) | REST `400` (KHÔNG publish AMQP event — ngoài Saga) |

## Relationships
```
PublishRequest
  └── validated by AssembleVideoUseCase-equivalent (PublishVideoUseCase)
        ├── requires → OAuthCredential (from CredentialStorePort.get())
        │     └── refreshed if expired (Rule 3) → re-saved via CredentialStorePort.save()
        └── produces → PublishResult (on success)
              or raises → InvalidPublishRequestError | MissingCredentialError | UploadError (on failure)
```
