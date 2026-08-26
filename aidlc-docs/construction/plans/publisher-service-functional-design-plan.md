# Functional Design Plan — Unit 7: Publisher Service

## Unit Context
- **Stories**: E1 (xác thực OAuth), E2 (cấu hình metadata — GUI/Orchestrator gán, Publisher chỉ tiêu thụ), E3 (tự động đăng video)
- **Scope**: Xác thực OAuth 2.0 với YouTube + lưu credential (FR7.3); upload video .mp4 kèm metadata lên YouTube (FR7.1, FR7.2 phần tiêu thụ metadata). KHÔNG chịu trách nhiệm thu thập/validate metadata từ Creator (đó là GUI, Unit 10) hay quyết định khi nào dispatch `publish_video` (đó là Orchestrator, Unit 8).

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `business-logic-model.md`
- [ ] Tạo `business-rules.md`
- [ ] Tạo `domain-entities.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Business Rule — Input Validation Strategy (zero trust, nhất quán Unit 5/6)
A) 💡 Suggested: Zero trust — validate toàn bộ `publish_video` payload trước khi upload: (1) `video_path` không rỗng VÀ file thực sự tồn tại trên shared volume; (2) `title` không rỗng (YouTube API yêu cầu bắt buộc, khớp Story E2's AC "tiêu đề là bắt buộc"); (3) `visibility` phải là 1 trong `public`/`unlisted`/`private`; (4) `description`/`tags` optional, không validate nội dung (YouTube tự validate độ dài/ký tự cấm khi upload). Vi phạm bất kỳ điều nào → lỗi rõ ràng, không gọi YouTube API
   - ✅ Strengths: nhất quán Unit 5/6, phát hiện lỗi sớm và rõ ràng hơn để YouTube API tự báo lỗi mơ hồ
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Business Rule — Missing/Invalid `title` (Story E2's AC)
Story E2's AC quy định: thiếu tiêu đề → GUI ngăn việc bấm nút đăng tải. Nhưng Publisher Service nhận command trực tiếp từ Orchestrator (không phải trực tiếp từ GUI) — cần rule riêng cho Publisher Service nếu (do lỗi ở tầng trên) `title` rỗng vẫn lọt tới đây?

A) 💡 Suggested: Publisher Service validate lại `title` không rỗng (Question 1, zero trust) — nếu rỗng, raise lỗi validation (không gọi YouTube), publish `publish_failed` với `error_message` rõ ràng ("title is required"). Đây là lớp phòng thủ thứ 2 (defense in depth), không thay thế validation ở GUI (Story E2's AC vẫn là GUI chặn trước khi gửi) — chỉ đảm bảo Publisher Service không bao giờ gọi YouTube API với dữ liệu thiếu bắt buộc
   - ✅ Strengths: đúng tinh thần zero-trust đã thiết lập từ Unit 5/6, không phụ thuộc hoàn toàn vào validation ở tầng khác
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Business Logic — OAuth Token Refresh Timing (Story E1's AC)
LLD đã xác định: refresh trước mỗi lần upload nếu hết hạn. Cần rule cụ thể: refresh dựa vào `expires_at` đã lưu, hay luôn thử dùng token hiện tại trước rồi refresh khi gặp lỗi 401?

A) 💡 Suggested: Kiểm tra `expires_at` TRƯỚC khi gọi YouTube API (proactive refresh) — nếu `now() >= expires_at` (hoặc gần hết hạn, buffer 60s để tránh race condition giữa lúc check và lúc gọi API thật), refresh trước khi upload. KHÔNG dùng reactive refresh (thử trước, refresh khi 401) vì sẽ lãng phí 1 lần gọi API chắc chắn thất bại nếu token đã biết hết hạn
   - ✅ Strengths: tránh gọi YouTube API biết trước sẽ thất bại, đơn giản để implement (so sánh timestamp)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 4: Business Rule — `visibility` Default Value
Nếu `visibility` không có trong payload (dù `component-methods.md` liệt kê là field bắt buộc), Publisher Service xử lý sao?

A) 💡 Suggested: KHÔNG có default — coi đây là zero-trust validation failure (Question 1) giống thiếu `title`, publish `publish_failed`. `visibility` ảnh hưởng trực tiếp tới quyền riêng tư video của Creator (đăng nhầm "public" khi Creator muốn "private" là rủi ro nghiêm trọng hơn nhiều so với các field khác) — không nên tự ý default để tránh đăng sai chế độ hiển thị ngoài ý muốn
   - ✅ Strengths: an toàn — không bao giờ tự đoán ý định privacy của Creator
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Domain Entity — `PublishResult` có cần thêm field không?
LLD đã định nghĩa `PublishResult(youtube_video_url)`. Có cần thêm field khác (vd. `youtube_video_id`, `published_at`) không?

A) 💡 Suggested: KHÔNG bổ sung — giữ nguyên như LLD, nhất quán tinh thần "event contract chỉ chứa dữ liệu thực sự cần dùng ở bước sau" (Unit 3/5/6's Functional Design). `youtube_video_url` đủ để GUI hiển thị link cho Creator (không có story nào yêu cầu hiển thị `video_id` riêng hay `published_at` — YouTube tự hiển thị thời gian đăng trên trang video)
   - ✅ Strengths: entity gọn, đúng nhu cầu thực tế các story đã xác nhận
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: Business Scenario — OAuth Callback với `code` không hợp lệ/hết hạn
A) 💡 Suggested: `HandleOAuthCallbackUseCase` bắt lỗi từ Google's token exchange (code sai/hết hạn/đã dùng), trả về lỗi rõ ràng cho GUI qua REST response (`400`, không phải publish event AMQP vì đây là luồng REST ngoài Saga) — GUI hiển thị thông báo và cho phép Creator thử lại luồng `/v1/auth/youtube/start` từ đầu
   - ✅ Strengths: đúng ranh giới luồng REST vs AMQP, đơn giản (không cần retry logic phức tạp — Creator tự bấm lại nút "Kết nối")
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
