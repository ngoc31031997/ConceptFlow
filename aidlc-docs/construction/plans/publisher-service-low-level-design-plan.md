# Low-Level Design Plan — Unit 7: Publisher Service

## Unit Context
- **Responsibility**: Xác thực OAuth 2.0 với YouTube (Story E1, FR7.3), lưu credential; nhận command `publish_video` kèm metadata (Story E2, FR7.2) từ Orchestrator; upload video .mp4 lên YouTube (Story E3, FR7.1)
- **Architectural style**: Hexagonal/Ports & Adapters (ADR-0002), Python 3.12 + FastAPI (ADR-0003, ADR-0009) — unit đầu tiên từ Unit 3 trở đi có REST endpoint (mirror Content Plugin Service's `api/` layer)
- **Interfaces**: REST `GET /auth/youtube/start`, `GET /auth/youtube/callback` (OAuth, ngoài Saga) + AMQP consumer `publish_video` (queue `publisher.commands`) → publish `video_published`/`publish_failed`; PostgreSQL Inbox/Outbox (ADR-0013, mirror Unit 2/3/4/5/6) — khác Unit 2, thiết kế Inbox/Outbox NGAY TỪ ĐẦU thay vì retrofit sau
- **Depends on**: Unit 1 (RabbitMQ) — không phụ thuộc trực tiếp unit nào khác

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `module-structure.md`
- [ ] Tạo `dependency-injection.md`
- [ ] Tạo `interface-contracts.md`
- [ ] Tạo `sequence-flows.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Layering & Dependency Direction (BẮT BUỘC)
A) 💡 Suggested: `domain/` (`PublishRequest`/`PublishResult` model, `OAuthCredential` value object, `MissingCredentialError`/`UploadError` errors, `VideoPublisherPort` interface — không import Google API client/FastAPI/AMQP/Postgres cụ thể) → `application/` (`PublishVideoUseCase`: điều phối domain + port; `HandleOAuthCallbackUseCase`: xử lý callback OAuth) → `adapters/` (`api/` FastAPI router cho OAuth flow, `messaging/`, `persistence/` giống Unit 2/3/4/5/6 (Inbox/Outbox) + thêm bảng `oauth_credentials`, `youtube/` chứa `YouTubeVideoPublisher` implement `VideoPublisherPort`, `logging/`)
   - ✅ Strengths: nhất quán toàn hệ thống, tách Google API client cụ thể khỏi domain/application (có thể đổi platform đăng video sau này mà không sửa business logic)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Dependency Injection (BẮT BUỘC)
A) 💡 Suggested: Constructor injection thủ công — `PublishVideoUseCase` nhận `VideoPublisherPort` qua constructor; composition root `main.py` wire `YouTubeVideoPublisher` cụ thể, khớp FastAPI's `Depends()` cho REST layer (mirror Unit 2)
   - ✅ Strengths: nhất quán, cho phép test độc lập (fake publisher)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: OAuth 2.0 Flow & Credential Storage (BẮT BUỘC — quyết định mới quan trọng)
Story E1 yêu cầu: xác thực 1 lần, credential được lưu để dùng lại (không cần đăng nhập lại). Hệ thống chỉ có 1 Creator (1 user) — không multi-tenant.

A) 💡 Suggested: Dùng `google-auth-oauthlib`'s flow chuẩn (Authorization Code flow): `GET /auth/youtube/start` redirect Creator tới Google's consent screen; `GET /auth/youtube/callback` nhận `code`, đổi lấy `access_token`+`refresh_token`, lưu vào bảng Postgres `oauth_credentials` (1 row duy nhất — hệ thống single-user, không cần `user_id` khóa ngoại phức tạp). Trước mỗi lần upload, `YouTubeVideoPublisher` tự refresh `access_token` nếu hết hạn (dùng `refresh_token`, không cần Creator xác thực lại — đúng AC của Story E1)
   - ✅ Strengths: đúng chuẩn OAuth 2.0, đáp ứng AC "không cần đăng nhập lại", đơn giản hóa vì chỉ 1 user (không cần multi-tenant credential lookup)
   - ⚠️ Trade-offs: `access_token`/`refresh_token` lưu dạng gì (plaintext hay mã hóa) cần quyết định riêng — xem Question 4

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Credential Encryption at Rest
`access_token`/`refresh_token` là dữ liệu nhạy cảm (cho phép đăng video thay Creator). Hệ thống chạy hoàn toàn local trên máy cá nhân (không multi-tenant, không public-facing).

A) 💡 Suggested: Lưu plaintext trong Postgres `oauth_credentials` (không mã hóa thêm) — vì: (1) Postgres container chỉ accessible trong Docker network nội bộ, không expose ra host; (2) đây là dữ liệu cục bộ trên máy cá nhân của chính Creator, không phải multi-tenant SaaS; (3) mã hóa thêm (vd. Fernet với key riêng) tạo thêm bề mặt quản lý key mà không giảm rủi ro thực tế ở threat model này (nếu máy bị compromise, key cũng bị lộ theo)
   - ✅ Strengths: đơn giản, đúng threat model (single-user local tool, không phải multi-tenant cloud service)
   - ⚠️ Trade-offs: nếu dự án mở rộng sang multi-user/cloud sau này, cần bổ sung mã hóa — ghi nhận là follow-up, không phải rủi ro ở MVP hiện tại

B) Mã hóa `access_token`/`refresh_token` bằng Fernet (symmetric encryption), key đọc từ biến môi trường (`CREDENTIAL_ENCRYPTION_KEY`)
   - ✅ Strengths: phòng thủ theo chiều sâu (defense in depth) — an toàn hơn nếu Postgres data volume bị truy cập trực tiếp (vd. backup file bị lộ)
   - ⚠️ Trade-offs: thêm độ phức tạp quản lý key, cần cơ chế xoay key (key rotation) nếu key bị lộ — over-engineering cho MVP 1 người dùng local

C) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: YouTube Upload Mechanism
A) 💡 Suggested: Dùng `google-api-python-client`'s `videos().insert()` với `MediaFileUpload(resumable=True)` — resumable upload chuẩn của YouTube Data API v3, phù hợp với file video có thể khá lớn (không bị lỗi network giữa chừng làm mất toàn bộ tiến trình upload). Metadata (title, description, tags, visibility) map trực tiếp từ `publish_video` command payload vào YouTube API's `snippet`/`status` object
   - ✅ Strengths: đúng API chính thức, resumable upload là best practice cho file lớn
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: Execution Model — YouTube Upload (Network I/O, có thể chậm)
A) 💡 Suggested: Upload chạy trong `ThreadPoolExecutor` (mirror Unit 3/5/6's pattern — `google-api-python-client` không có async API chính thức), timeout đọc từ env var `UPLOAD_TIMEOUT_SECONDS` (mặc định 600 giây = 10 phút, dư dả cho video giáo dục thông thường qua kết nối cá nhân). Vượt timeout hoặc lỗi mạng → `UploadError` → `publish_failed`
   - ✅ Strengths: nhất quán pattern threadpool+timeout, không block event loop (FastAPI vẫn phục vụ OAuth callback trong lúc upload chạy nền)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 7: Idempotency & Retry (Story E3's AC — "thử lại mà không phải cấu hình lại metadata")
A) 💡 Suggested: KHÔNG idempotent theo file như Unit 3/5/6 (mỗi lần `publish_video` thành công tạo 1 video MỚI trên YouTube — không có khái niệm "video đã tồn tại, dùng lại" vì YouTube không có natural key trùng lặp cho việc này). Message-level: Inbox dedupe `message_id` như thường lệ (chặn xử lý trùng lặp redelivery). Story E3's AC "thử lại không cần cấu hình lại metadata" là trách nhiệm của GUI/Orchestrator (giữ lại metadata đã nhập, gửi lại command `publish_video` MỚI với `message_id` mới) — KHÔNG phải trách nhiệm của Publisher Service tự động retry upload đã thất bại
   - ✅ Strengths: đúng ranh giới trách nhiệm — Publisher Service chỉ thực hiện 1 lần upload/command, retry logic là quyết định của Orchestrator/GUI
   - ⚠️ Trade-offs: nếu Orchestrator gửi lại `publish_video` sau khi upload thất bại giữa chừng, có thể tạo video trùng lặp trên YouTube nếu phần upload trước đó thực ra đã thành công nhưng response bị mất — chấp nhận được ở MVP (rủi ro thấp, Creator có thể xóa video trùng thủ công trên YouTube nếu xảy ra)

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 8: API Versioning (nhất quán Unit 2, ADR-0008)
A) 💡 Suggested: URI versioning `/v1/auth/youtube/start`, `/v1/auth/youtube/callback` — nhất quán ADR-0008 (đã áp dụng cho mọi REST endpoint hệ thống, Unit 2's LLD Question 4)
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 9: Distributed Tracing & Correlation ID
A) 💡 Suggested: `saga_id` từ AMQP envelope cho luồng `publish_video` (thuộc Saga Publish). OAuth flow (`/auth/youtube/start`/`/callback`) KHÔNG thuộc Saga — dùng header `X-Request-ID` (mirror Unit 2's Question 5), vì đây là luồng tương tác trực tiếp Creator ↔ Google, không qua Orchestrator
   - ✅ Strengths: nhất quán Unit 2, đúng phân loại luồng nào thuộc Saga
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 10: Error Classification & Compensating Action
A) 💡 Suggested: `MissingCredentialError` (chưa xác thực OAuth — Creator cần thực hiện Story E1 trước) và `UploadError` (lỗi mạng/YouTube API/timeout) đều publish `publish_failed`, coi là transient (Orchestrator/Creator có thể yêu cầu retry theo Story E3's AC — Question 7). Không có compensating action rollback (không có gì để "hoàn tác" nếu upload thất bại — video đơn giản chưa tồn tại trên YouTube)
   - ✅ Strengths: đơn giản, đúng bản chất (Publisher Service không giữ state trung gian cần rollback)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 11: State Management
A) 💡 Suggested: Postgres chứa `outbox_events`/`processed_messages` (ADR-0013) + `oauth_credentials` (1 row, Question 3/4) — 2 loại dữ liệu khác bản chất (kỹ thuật vs. business credential) nhưng cùng 1 database-per-service instance (`publisher-db`), không cần tách riêng
   - ✅ Strengths: đơn giản, database-per-service vẫn giữ nguyên (1 DB/unit), chỉ thêm 1 bảng business bên cạnh Inbox/Outbox kỹ thuật
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
