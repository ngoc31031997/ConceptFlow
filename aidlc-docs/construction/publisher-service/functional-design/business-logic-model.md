# Business Logic Model — Unit 7: Publisher Service

## Business Process 1: OAuth Authentication (FR7.3 — Story E1)

**Trigger**: Creator bấm "Kết nối tài khoản YouTube" trong GUI → `GET /v1/auth/youtube/start`.

**Processing Steps**:
1. Build Google OAuth authorization URL, redirect Creator.
2. Creator cấp quyền trên Google's consent screen.
3. Google redirect về `GET /v1/auth/youtube/callback?code=...`.
4. `HandleOAuthCallbackUseCase` đổi `code` lấy `access_token`+`refresh_token`+`expires_in` qua Google's token endpoint.
5. Nếu đổi thành công: lưu `OAuthCredential` vào `oauth_credentials` (upsert — ghi đè credential cũ nếu có, single-user).
6. Nếu đổi thất bại (code sai/hết hạn/đã dùng): trả `400` cho GUI, không lưu gì (Business Rule 6).

**Output**: `oauth_credentials` có 1 row hợp lệ; các lần đăng video sau không cần Creator xác thực lại (đúng AC Story E1).

## Business Process 2: Publish Video (FR7.1, FR7.2 — Story E3)

**Trigger**: Command `publish_video` từ Orchestrator (sau khi Creator cấu hình metadata ở GUI — Story E2 — và bấm "Đăng lên YouTube").

**Processing Steps** (không có batch — 1 command = 1 video, mirror Unit 6):
1. **Idempotency check**: KHÔNG áp dụng — mỗi `publish_video` thành công luôn tạo video mới (Business Rule 7 kế thừa từ Low-Level Design Question 7).
2. **Zero-trust input validation** (Business Rule 1): `video_path` tồn tại trên shared volume, `title` không rỗng, `visibility` hợp lệ (`public`/`unlisted`/`private`).
3. **Credential lookup**: lấy `OAuthCredential` đã lưu — thiếu → `MissingCredentialError` → `publish_failed`.
4. **Token refresh** (nếu cần, Business Rule 3): kiểm tra `expires_at`, refresh proactive nếu gần/đã hết hạn, lưu credential mới.
5. **Upload**: gọi YouTube Data API's resumable upload với metadata (`title`, `description`, `tags`, `visibility`).
6. **Publish kết quả**: `video_published` (thành công, kèm `youtube_video_url`) hoặc `publish_failed` (bất kỳ bước 2-5 lỗi).

## Business Process Boundary
Publisher Service KHÔNG chịu trách nhiệm:
- Thu thập/validate metadata từ Creator ở GUI (Story E2's AC "ngăn bấm nút nếu thiếu tiêu đề" là trách nhiệm GUI — Publisher Service chỉ có lớp phòng thủ zero-trust thứ 2, Business Rule 2).
- Quyết định khi nào dispatch `publish_video` — đó là Orchestrator (Unit 8), sau khi Saga tới bước "Publish Video".
- Tự động retry sau khi upload thất bại — GUI/Orchestrator giữ metadata và gửi lại command mới nếu Creator yêu cầu (Story E3's AC).
