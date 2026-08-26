# Business Rules — Unit 7: Publisher Service

## Rule 1: Zero-Trust Input Validation (Question 1)
Không tin tưởng dữ liệu từ Orchestrator dù metadata đã qua GUI validate — validate toàn bộ trước khi gọi YouTube API:
- `video_path` không rỗng và file thực sự tồn tại trên shared volume.
- `title` không rỗng (Rule 2).
- `visibility` phải là 1 trong `public`/`unlisted`/`private` (Rule 4).
- `description`/`tags` optional — không validate nội dung (YouTube tự validate độ dài/ký tự cấm khi upload).
- Vi phạm bất kỳ điều nào → lỗi rõ ràng, không gọi YouTube API, publish `publish_failed`.

## Rule 2: Missing `title` — Defense in Depth (Question 2)
`title` rỗng/thiếu → validation failure ngay tại Publisher Service (không phụ thuộc hoàn toàn vào GUI's Story E2 validation). Đây là lớp phòng thủ thứ 2, không thay thế UX validation ở GUI.

## Rule 3: Proactive OAuth Token Refresh (Question 3)
Trước mỗi lần upload, so sánh `now()` với `credential.expires_at` (buffer 60 giây để tránh race condition giữa lúc check và lúc gọi API thật). Nếu `now() >= expires_at - 60s`: refresh `access_token` bằng `refresh_token` TRƯỚC khi gọi YouTube API, lưu credential mới qua `CredentialStorePort.save()`. KHÔNG dùng reactive refresh (thử trước, refresh khi gặp 401) — tránh lãng phí 1 lần gọi API chắc chắn thất bại.

## Rule 4: `visibility` — Không Có Default (Question 4)
`visibility` thiếu/không hợp lệ → validation failure (Rule 1), KHÔNG tự động default về bất kỳ giá trị nào (kể cả `private` — giá trị "an toàn nhất" theo trực giác). Đăng sai chế độ hiển thị ngoài ý muốn của Creator là rủi ro nghiêm trọng hơn việc từ chối xử lý và yêu cầu dữ liệu đầy đủ.

## Rule 5: Domain Entity Scope — `PublishResult` Tối Giản (Question 5)
`PublishResult` chỉ chứa `youtube_video_url` — không tính/truyền thêm `youtube_video_id` hay `published_at` qua event. GUI hiển thị link cho Creator; YouTube tự hiển thị thời gian đăng trên trang video.

## Rule 6: OAuth Callback Error Handling — REST Response, Không Publish Event (Question 6)
Lỗi khi đổi `code` lấy token (code sai/hết hạn/đã dùng) → `HandleOAuthCallbackUseCase` trả lỗi qua REST response (`400`), KHÔNG publish AMQP event — luồng OAuth nằm ngoài Saga, không có `saga_id` để gắn vào event. GUI hiển thị thông báo, Creator tự bấm lại `/v1/auth/youtube/start` nếu muốn thử lại (không cần retry logic phức tạp).

## Rule 7: No Artifact-Level Idempotency (Low-Level Design Question 7, không đổi)
Mỗi `publish_video` thành công luôn tạo 1 video MỚI trên YouTube — không có khái niệm "video đã tồn tại, dùng lại". Message-level idempotency (Inbox dedupe `message_id`) vẫn áp dụng để chặn redelivery trùng lặp.

## Rule 8: Error Classification — Tất Cả Transient (Low-Level Design Question 10, không đổi)
`MissingCredentialError` và `UploadError` đều publish `publish_failed`, coi là transient. Không có compensating action rollback.
