# ADR-0023 — Dùng Google Cloud Text-to-Speech làm engine giọng đọc chính, giữ Piper làm fallback

## Date
2026-09-08

## Status
Accepted

## Context

ADR-0010 chọn Piper làm engine TTS với lý do chạy hoàn toàn local, miễn phí, không phụ thuộc mạng. Quyết định đó đúng cho mục tiêu lúc bấy giờ (công cụ cá nhân, chạy offline qua Docker).

Mục tiêu đã thay đổi: Creator muốn **bật kiếm tiền trên YouTube**. Điều này đưa vào một ràng buộc mà ADR-0010 không hề cân nhắc — **chính sách nội dung của nền tảng**.

Ba dữ kiện dẫn tới quyết định này:

1. **Chất lượng giọng Piper là rủi ro monetization.** Chính sách YouTube Partner Program về *inauthentic / mass-produced content* nhắm thẳng vào video "giọng đọc máy + nội dung sinh tự động". CR-001 §C1 đã ghi nhận kho `rhasspy/piper-voices` chỉ có 3 giọng tiếng Việt, giọng nam duy nhất là `vi_VN-vivos-x_low` ở mức chất lượng **x_low**.

2. **Free tier của Google phủ hết nhu cầu thực tế.** Tra cứu 2026-09-07: hạng WaveNet cho **4 triệu ký tự/tháng miễn phí**, không hết hạn. Đối chiếu với ~8.000 ký tự cho một video 10 phút (CR-005 §C6), con số này tương đương **~500 video/tháng**. Nhịp đăng 1 video/ngày chỉ dùng ~6% hạn mức. Chi phí khi vượt là ~$0.03/video 10 phút.

3. **Kiến trúc đã sẵn sàng.** ADR-0010 thiết kế `TTSEnginePort` chính là để thay engine mà không đụng domain. Thêm adapter là đúng ý đồ ban đầu, không phải phá vỡ nó.

## Decision

**Dùng Google Cloud Text-to-Speech (hạng WaveNet) làm engine mặc định, giữ Piper làm fallback.**

Cụ thể:

- Thêm `GoogleTTSAdapter` implement `TTSEnginePort` — không sửa domain/application.
- `voice_registry` mang thêm trường `engine`; danh mục giọng gộp cả hai engine.
- **Fallback tự động về Piper** khi thiếu credential hoặc lỗi mạng (CR-005 FR13.3), kèm cảnh báo rõ ràng — không im lặng.
- Credential qua `GOOGLE_APPLICATION_CREDENTIALS`, tách hoàn toàn với OAuth của Publisher Service (khác scope, khác vòng đời).

## Consequences

### Chấp nhận đánh đổi

**Phá vỡ nguyên tắc "chạy hoàn toàn local qua Docker" ghi trong README.** Đây là hệ quả trực tiếp và có ý thức. Lý do chấp nhận: nguyên tắc local-only phục vụ mục tiêu "công cụ cá nhân không phụ thuộc dịch vụ ngoài", nhưng mục tiêu hiện tại là **xuất bản có kiếm tiền**, mà chất lượng giọng lại là rào cản lớn nhất. Piper vẫn ở đó, nên **khả năng chạy offline không mất đi** — nó xuống hạng từ mặc định thành fallback.

README cần sửa để phản ánh đúng: hệ thống chạy local được, nhưng chất lượng tốt nhất cần credential Google.

### Rủi ro và cách xử lý

| Rủi ro | Xử lý |
|---|---|
| Vượt hạn mức miễn phí ngoài ý muốn | Đếm ký tự theo tháng + cảnh báo ở GUI (FR13.5). Ngưỡng 4M rất xa nên đây là "để yên tâm", không phải hàng rào |
| Mất mạng giữa lúc render | Fallback Piper + cảnh báo (FR13.3) |
| Google đổi giá / bỏ free tier | `TTSEnginePort` cho phép đổi engine lần nữa với chi phí thấp — đó là điểm mạnh của thiết kế này |
| Lộ credential | Service account chỉ cần scope `cloud-platform` cho TTS; không dùng chung với OAuth YouTube |
| Lãng phí quota khi render lại | Cache theo hash `(text, voice_id, engine)` (FR13.6) |

### Không chọn phương án nào khác, vì sao

- **ElevenLabs** — chất lượng cao nhất nhưng đắt nhất cho video dài; không có free tier đủ dùng.
- **Azure Neural** — tương đương Google về chất lượng, nhưng dự án đã có sẵn hệ sinh thái Google (OAuth YouTube), giảm số nhà cung cấp phải quản lý.
- **viXTTS / F5-TTS (local)** — giữ được nguyên tắc local-only và miễn phí thật sự, nhưng cần GPU để không quá chậm, model lớn, tích hợp phức tạp hơn nhiều. **Vẫn là phương án dự phòng tốt nếu sau này muốn quay lại local-only** — ghi nhận ở CR-005.

## Liên quan
- ADR-0010 (chọn Piper) — ADR này **bổ sung**, không thay thế: Piper vẫn trong hệ thống.
- ADR-0014 (TTS message-driven) — không đổi.
- CR-001 §C1/§C2 — đã ghi nhận giới hạn giọng Piper và liệt kê phương án mở rộng.
- CR-005 — yêu cầu đầy đủ.
