# NFR Design Patterns — Unit 7: Publisher Service

## CRUD vs CQRS
CRUD đơn giản trên `outbox_events`/`processed_messages` (ADR-0013) + `oauth_credentials` (1 row, upsert). Không phải CQRS.

## Resilience Pattern
Không retry nội bộ. Lỗi upload/timeout (sau `UPLOAD_TIMEOUT_SECONDS`) → `UploadError` → `publish_failed` ngay, để Orchestrator/GUI quyết định retry (transient). Lỗi validation (`InvalidPublishRequestError`) và `MissingCredentialError` cũng raise ngay, không retry nội bộ — permanent tại thời điểm đó (cần Creator/upstream sửa trước khi retry có ý nghĩa). OAuth callback (REST, ngoài Saga): không có "retry nội bộ" — Creator tự bấm lại `/v1/auth/youtube/start`.

## Idempotency Pattern
CHỈ message-level (Inbox, dedupe `message_id`) — KHÔNG có artifact-level idempotency (khác Unit 3/5/6), vì mỗi `publish_video` thành công luôn tạo video MỚI trên YouTube (không có "artifact" nào để idempotency-by-file như video/audio clip). `oauth_credentials` tự nhiên idempotent theo thiết kế (upsert, callback nhiều lần chỉ ghi đè cùng 1 row).

## Saga Pattern
Participant trực tiếp, bước cuối "Publish Video" trong Saga Render+Publish (sau "Assemble Video", không có bước nào sau nó). Không có compensating action — video đã đăng lên YouTube không thể tự động "hoàn tác" qua hệ thống.

## Event-Driven Design
Consumer `publish_video` (queue `publisher.commands`), publish đúng 1 trong 2 event: `video_published` HOẶC `publish_failed` — 1 Outbox row/command (mirror Unit 6, không có progress event trung gian).

## Inbox/Outbox Pattern
PostgreSQL-backed (`publisher-db`, ADR-0013), Outbox row (event kết quả) và Inbox mark (`processed_messages`) ghi CÙNG 1 transaction (mirror Unit 6). Bảng `oauth_credentials` là business data riêng biệt (không thuộc Inbox/Outbox pattern kỹ thuật) — cùng database instance nhưng khác mục đích.

## Security Pattern
Zero-trust validation (Functional Design Rule 1) cho `publish_video` payload trước khi gọi YouTube API. OAuth flow dùng `state` parameter chuẩn chống CSRF. Credential lưu plaintext trong Postgres nội bộ (ADR-0016, threat model single-user local). Không auth/rate-limit riêng cho AMQP hay REST OAuth endpoint.
