# Setup kênh YouTube ConceptFlow — cho khán giả Việt và quốc tế

Kênh đăng video giải thích khái niệm bằng animation Manim, lời dẫn **tiếng Việt và
tiếng Anh** (CR-008 cho phép chọn ngôn ngữ cho cả pipeline).

Phục vụ hai nhóm khán giả cùng lúc là quyết định có đánh đổi thật, nên phần đầu
tiên là chọn chiến lược — mọi bước sau phụ thuộc vào nó.

---

## 0. Quyết định trước: một kênh hay hai kênh?

| | **Một kênh, hai ngôn ngữ** | **Hai kênh riêng** |
|---|---|---|
| Công sức | Thấp — một nơi để chăm | Gấp đôi: 2 avatar, 2 banner, 2 lịch đăng |
| Thuật toán YouTube | **Bất lợi rõ** — YouTube học "ai thích kênh này" từ lịch sử xem; trộn ngôn ngữ làm tín hiệu nhiễu, đề xuất kém đi cho cả hai nhóm | Mỗi kênh một tín hiệu sạch |
| Người xem | Người Việt thấy video tiếng Anh không hiểu → lướt qua → giảm tỉ lệ giữ chân của cả kênh | Đúng thứ họ đăng ký |
| Lúc mới bắt đầu | Dồn được lượt xem vào một chỗ, qua ngưỡng kiếm tiền nhanh hơn | Chia đôi, lâu hơn |

**Khuyến nghị: bắt đầu bằng MỘT kênh, một ngôn ngữ chính.** Rồi dùng
**multi-language audio track** (mục 4) để phủ ngôn ngữ kia — cách này cho một video
mang nhiều bản lời dẫn, không cần kênh thứ hai và không làm loãng tín hiệu thuật toán.

Chọn ngôn ngữ chính theo mục tiêu:

- **Tiếng Việt**: ít cạnh tranh hơn nhiều ở mảng giáo dục Manim, dễ nổi bật, khán
  giả trung thành. Nhưng CPM thấp (~$0.5–2) và trần lượng người xem nhỏ.
- **Tiếng Anh**: thị trường lớn gấp bội, CPM cao (~$4–12), nhưng phải cạnh tranh
  trực tiếp với 3Blue1Brown, Veritasium… ở đúng thể loại animation toán.

> Nếu chưa chắc: bắt đầu tiếng Việt. Nổi bật ở hồ nhỏ rồi mở rộng dễ hơn là chìm
> ở hồ lớn. Kênh này đã có sẵn hạ tầng đa ngôn ngữ nên mở rộng sau không tốn nhiều.

---

## 1. Tạo kênh

1. Đăng nhập YouTube → avatar góc phải → **Tạo kênh / Create channel**
2. Cân nhắc dùng **Brand Account** thay vì kênh cá nhân:
   - Tách khỏi tên thật trên tài khoản Google
   - Thêm được người quản lý khác mà không chia sẻ mật khẩu
   - **Đổi chủ sở hữu được** sau này
   - Một tài khoản Google tạo được nhiều Brand Account
3. Tên kênh: **ConceptFlow** — đặt được cho cả hai nhóm khán giả (không dấu, dễ
   đọc với người nước ngoài, dễ tìm)

## 2. Ảnh nhận diện

| Mục | Kích thước | File có sẵn trong repo |
|---|---|---|
| Ảnh đại diện | 800 × 800 | `docs/brand/conceptflow-avatar-800.png` |
| Banner | 2560 × 1440 | `docs/brand/conceptflow-banner-2560.jpg` (dựng lại bằng `make-banner.py`) |
| Hình mờ video | 150 × 150, nền trong | cắt từ `conceptflow-mark-1024.png` |

Upload: **YouTube Studio → Customization → Branding**

## 3. Cấu hình cho khán giả quốc tế

Đây là phần hay bị bỏ qua nhất, mà lại quyết định video có ra khỏi Việt Nam không.

### 3.1 Ngôn ngữ kênh
**Settings → Channel → Advanced settings**

- **Language**: đặt đúng ngôn ngữ chính bạn chọn ở mục 0
- **Country of residence**: Vietnam (ảnh hưởng thanh toán và tuân thủ, không giới
  hạn ai xem được)

### 3.2 Ngôn ngữ từng video
**Mỗi lần upload**, trong Details → **Show more**:

- **Video language**: ngôn ngữ lời dẫn thật của video đó
- **Subtitles/CC**: tải lên phụ đề

Khai sai ngôn ngữ khiến YouTube đề xuất video cho nhóm người không hiểu nó — hại
hơn là không khai.

### 3.3 Phụ đề — đòn bẩy lớn nhất cho khán giả quốc tế

Pipeline **đã sinh sẵn file phụ đề** khớp timeline thật (CR-002). Dùng nó:

- Upload phụ đề **ngôn ngữ gốc** trước → giúp YouTube hiểu nội dung, cải thiện SEO
- Rồi thêm phụ đề **ngôn ngữ thứ hai** → video xuất hiện trong kết quả tìm kiếm
  của ngôn ngữ đó

Đừng dùng phụ đề tự động của YouTube cho tiếng Việt — độ chính xác kém, nhất là với
thuật ngữ toán.

### 3.4 Tiêu đề và mô tả song ngữ

Cách gọn cho một kênh phục vụ hai nhóm:

```
Tiêu đề:  Định lý Pythagoras giải thích trực quan | Pythagorean Theorem Visualized

Mô tả:
[VI] Giải thích định lý Pythagoras bằng animation...

[EN] A visual explanation of the Pythagorean theorem...
```

Tiêu đề đừng dài quá — YouTube cắt sau khoảng 60 ký tự trong hầu hết giao diện.

## 4. Multi-language audio (cách phủ hai ngôn ngữ mà không cần kênh thứ hai)

YouTube cho phép **nhiều bản lời dẫn trên cùng một video**; người xem tự chọn, hoặc
YouTube chọn theo cài đặt ngôn ngữ của họ.

Đây là thứ hợp với dự án này đến mức gần như được thiết kế sẵn: cùng một video đã
render, chỉ cần chạy lại TTS ở ngôn ngữ khác rồi upload track âm thanh thứ hai.

Cách làm:
1. Tạo video với `content_language = vi` → xuất bản
2. Tạo lại **cùng lời dẫn đó** với `content_language = en` → lấy file audio
3. YouTube Studio → video → **Subtitles** → **Add language** → thêm audio track

> Tính năng này trước đây giới hạn, nay đã mở rộng cho hầu hết kênh. Không thấy tuỳ
> chọn trong Studio thì kênh bạn chưa được bật — cứ dùng phụ đề (3.3), hiệu quả gần
> bằng mà không cần chờ.

## 5. Kiếm tiền (YPP)

Ngưỡng: **1.000 người đăng ký** + **4.000 giờ xem công khai trong 12 tháng**, hoặc
**10 triệu lượt xem Shorts trong 90 ngày**.

Với kênh này có hai điều đáng lưu ý:

- **Nội dung AI không bị cấm**, nhưng YouTube yêu cầu **giá trị gia tăng gốc**.
  Video giải thích khái niệm bằng animation Manim tự viết là nội dung gốc — khác
  hẳn kiểu đọc lại bài viết bằng giọng máy. Giữ ranh giới đó rõ ràng.
- **Khai báo nội dung tổng hợp**: khi upload, ở mục *Altered content*, khai nếu
  video dùng giọng đọc AI. Không khai mà bị phát hiện thì rủi ro lớn hơn nhiều so
  với khai.
- **CPM phụ thuộc nơi người xem ở**, không phải nơi bạn ở. Kênh tiếng Việt có khán
  giả Mỹ vẫn được CPM Mỹ cho lượt xem đó.

## 6. Checklist trước video đầu tiên

- [ ] Chọn xong ngôn ngữ chính (mục 0)
- [ ] Avatar 800×800 đã upload
- [ ] Banner đã kiểm tra bằng `banner_mobile_preview.png` (xem `youtube-avatar-prompts.md`)
- [ ] Channel language khai đúng
- [ ] Mô tả kênh có cả hai ngôn ngữ
- [ ] Bật **2-Step Verification** cho tài khoản Google — YouTube bắt buộc để mở
      khoá thumbnail tuỳ chỉnh, mà pipeline đã sinh sẵn thumbnail (CR-006)
- [ ] Nối kênh vào ConceptFlow: xem [`docs/setup/youtube-oauth-client.md`](../setup/youtube-oauth-client.md)

> **Thumbnail tuỳ chỉnh cần xác minh số điện thoại.** Chưa xác minh thì bước upload
> thumbnail của pipeline sẽ thất bại lặng lẽ — code ghi log rồi bỏ qua, vì video đã
> lên rồi thì không đáng fail cả lần đăng (xem `youtube_publisher.py`).

## 7. Nhịp đăng

Đều đặn quan trọng hơn tần suất. Một video mỗi tuần vào cùng một ngày tốt hơn ba
video một tuần rồi im lặng một tháng — YouTube đọc tín hiệu này khi quyết định có
đẩy video mới của bạn đi không.

Với công suất pipeline hiện tại: **~6 video/ngày cho mỗi OAuth client** (giới hạn
quota YouTube API, xem `docs/setup/youtube-oauth-client.md`). Trần đó cao hơn nhu
cầu của một lịch đăng đều đặn rất nhiều, nên nó không phải thứ giới hạn bạn.
