# NFR Requirements — Unit 7: Publisher Service

## Performance
Upload chạy trong `ThreadPoolExecutor` (Low-Level Design Question 6), timeout `UPLOAD_TIMEOUT_SECONDS` (mặc định 600s = 10 phút, đọc từ env var). FastAPI event loop vẫn phục vụ OAuth callback bình thường trong lúc upload chạy nền (threadpool tách biệt).

## Resource Constraints — Network Bandwidth
Không giới hạn cứng ở tầng ứng dụng — kết nối mạng cá nhân của Creator tự nhiên giới hạn tốc độ. Xử lý TUẦN TỰ (RabbitMQ consumer prefetch=1, không upload song song nhiều video cùng lúc trong 1 process, nhất quán Unit 2/3/4/5/6).

## Availability
Chấp nhận unavailability tạm thời — không multi-instance/failover (nhất quán toàn hệ thống). Message ở lại queue `publisher.commands` cho tới khi service khởi động lại; REST endpoint OAuth đơn giản không khả dụng trong lúc service down, Creator thử lại sau.

## Security
Zero-trust validation (Functional Design Rule 1) cho `publish_video` payload. OAuth flow dùng `state` parameter chuẩn (CSRF protection). Credential lưu plaintext trong Postgres nội bộ (ADR-0016, threat model single-user local). Không auth/rate-limit riêng cho AMQP (chỉ Orchestrator gửi command nội bộ); REST OAuth endpoint không cần auth riêng (gọi trực tiếp bởi trình duyệt Creator qua redirect).

## Messaging & Event Participation
Consumer `publish_video` (queue `publisher.commands`), producer `video_published`/`publish_failed` (qua Outbox → `orchestrator.events`, 1 event/command — mirror Unit 6). Delivery guarantee: at-least-once, dedupe qua Inbox theo `message_id` (ADR-0013).

## Distributed Transaction Participation (Saga)
**Vai trò**: Participant trực tiếp, bước "Publish Video" — bước cuối của Saga Publish. **Compensating action**: Không cần rollback — video đã đăng lên YouTube không thể "hoàn tác" tự động qua hệ thống (Creator tự xóa thủ công trên YouTube nếu cần, ngoài phạm vi hệ thống).

## Caching Requirements
`OAuthCredential` đọc từ Postgres mỗi lần `publish_video` (không cache in-memory) — token có thể được refresh bất kỳ lúc nào (Business Rule 3), tần suất publish rất thấp nên không cần tối ưu latency đọc DB.

## Tech Stack Consistency
Python 3.12 (ADR-0009) + FastAPI (ADR-0003, mirror Unit 2) + `google-api-python-client` + `google-auth-oauthlib` (YouTube Data API) + PostgreSQL Inbox/Outbox + `oauth_credentials` (ADR-0013, ADR-0016, asyncpg) + RabbitMQ (aio-pika). Xem `tech-stack-decisions.md` cho chi tiết đầy đủ.
