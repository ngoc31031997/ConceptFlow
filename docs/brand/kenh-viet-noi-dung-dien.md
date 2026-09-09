# Kênh tiếng Việt — nội dung điền vào YouTube Studio

**Dùng cho kênh chính, ưu tiên hiện tại.** Bản quốc tế: [kenh-quoc-te-noi-dung-dien.md](kenh-quoc-te-noi-dung-dien.md).

Đường dẫn: **YouTube Studio → Customization → Profile**
(https://studio.youtube.com/channel/UCRBW1xbNU3Lg_dOZ5ITAMnA/editing/profile)

Copy từng ô bên dưới, dán vào đúng ô cùng tên, xong bấm **Publish** góc trên phải.

---

## Picture (ảnh đại diện)

Upload file: `docs/brand/conceptflow-avatar-800.png`

## Name

```
ConceptFlow
```

> ⚠️ Ô này đang là **"Concert Flow"** — sai chính tả và sai nghĩa (*concert* =
> buổi hoà nhạc). Sửa ngay bây giờ, trước khi có người đăng ký: đổi tên về sau
> làm mất nhận diện và làm khán giả cũ bối rối.
>
> YouTube giới hạn **đổi tên 3 lần / 14 ngày**, nên đừng thử qua lại nhiều.

## Handle

```
@conceptflow
```

Nếu đã có người lấy, thử theo thứ tự — vẫn giữ được thương hiệu:

```
@conceptflow.vn
@conceptflowvn
@hocconceptflow
```

Tránh thêm số ngẫu nhiên (`@conceptflow2847`) — trông như tài khoản rác.

## Description

Ô này bị cắt sau khoảng **150 ký tự đầu** trong kết quả tìm kiếm, nên hai dòng
đầu phải nói hết mọi thứ quan trọng. Dán nguyên khối:

```
Giải thích các khái niệm toán học và khoa học bằng animation trực quan. Mỗi video biến một ý tưởng trừu tượng thành hình ảnh bạn thật sự nhìn thấy được.

Ở đây có gì:
• Toán học được hình dung — giải tích, đại số tuyến tính, xác suất, hình học
• Khái niệm khoa học mô phỏng bằng chuyển động, không phải công thức khô khan
• Lời giải thích ngắn gọn bằng tiếng Việt, không vòng vo

Video được dựng bằng Manim — thư viện animation toán học mã nguồn mở.

Đăng video mới mỗi tuần. Đăng ký để không bỏ lỡ.

Liên hệ hợp tác: <email của bạn>
```

**Thay `<email của bạn>`** bằng email thật trước khi dán.

> Vì sao viết vậy: câu đầu chứa đúng cụm người Việt hay gõ khi tìm — "giải thích",
> "khái niệm toán học", "trực quan". Đừng mở đầu bằng "Chào mừng đến với kênh
> của mình!" — vừa tốn mất phần hiển thị quý nhất, vừa không chứa từ khoá nào.

## Add language (bản dịch mô tả)

**Bấm "Add language" → chọn English**, dán bản dưới. Người dùng YouTube đặt ngôn
ngữ tiếng Anh sẽ thấy bản này thay vì bản tiếng Việt — vẫn giữ được kênh tiếng
Việt thuần mà không đóng cửa với người nước ngoài tình cờ ghé qua.

```
Visual explanations of mathematics and science. Every video turns an abstract idea into something you can actually see.

Narration is in Vietnamese; English subtitles are available on every video.

Animations are built with Manim, the open-source mathematical animation library.

New video every week.

Business enquiries: <email của bạn>
```

> Câu thứ hai là câu quan trọng nhất ở đây — nói thẳng lời dẫn là tiếng Việt.
> Không nói, người nước ngoài bấm vào rồi thoát ngay, và YouTube ghi nhận đó là
> tín hiệu xấu cho video.

## Links

Bấm **Add link** cho từng dòng (dòng đầu hiện đè lên banner, nên xếp quan trọng nhất trước):

| Link title | URL |
|---|---|
| Website | *(để trống tới khi có trang thật)* |
| GitHub | `https://github.com/ngoc31031997/ConceptFlow` |
| Facebook | *(trang kênh, nếu có)* |

Chưa có gì thì bỏ trống — link chết hại hơn là không có link.

## Contact info

```
<email của bạn>
```

Nên dùng email riêng cho kênh (vd. `conceptflow.contact@gmail.com`), không dùng
email cá nhân — địa chỉ này **hiển thị công khai** ở tab About.

---

## Sau khi Publish — ba việc còn lại, không nằm ở trang này

### 1. Ngôn ngữ kênh
**Settings → Channel → Advanced settings**
- Language: **Vietnamese**
- Country of residence: **Vietnam**

Khai đúng để YouTube biết đẩy kênh cho ai.

### 2. Banner
**Customization → Branding → Banner image**

Upload sẵn có, chọn một trong hai:

| File | Nội dung |
|---|---|
| `docs/brand/conceptflow-banner-2560.jpg` | torus + "ConceptFlow" + "Toán · Khoa học · kể bằng hình" — **khuyến nghị**, chữ đậm nên đọc rõ hơn ở cỡ nhỏ |
| `docs/brand/banner-b-wordmark-2560x1440.png` | bản thay thế, chữ mảnh hơn |
| `docs/brand/banner-a-logo-2560x1440.png` | chỉ torus, không chữ |

Dựng lại bản khuyến nghị: `python3 docs/brand/make-banner.py` (cần Pillow).

Cả hai đã đặt toàn bộ nội dung trong vùng an toàn 1546×423, nên hiển thị đúng trên
điện thoại, máy tính và TV. Muốn tự làm bản khác thì xem
[youtube-avatar-prompts.md](youtube-avatar-prompts.md) (mục banner) — nhớ kiểm tra
vùng điện thoại trước khi upload.

### 3. Xác minh số điện thoại
https://www.youtube.com/verify

**Bắt buộc** để dùng thumbnail tuỳ chỉnh. Pipeline có sinh thumbnail sẵn (CR-006),
nhưng chưa xác minh thì bước upload thumbnail thất bại **âm thầm** — code chỉ ghi
log rồi bỏ qua, vì video đã lên rồi thì không đáng huỷ cả lần đăng. Nghĩa là bạn
sẽ không thấy báo lỗi, chỉ thấy thumbnail không đổi.

### 4. Nối kênh vào ConceptFlow
Xem [`docs/setup/youtube-oauth-client.md`](../setup/youtube-oauth-client.md).

> Kênh mới này (`UCRBW1xbNU3Lg_dOZ5ITAMnA`) **khác** kênh cũ đang nằm trong database
> (`UCJmC-gKc58_suFBi7ja9ncw`, nối bằng OAuth client cũ đã bị gỡ). Vào màn kết quả
> trong app, **Ngắt** kênh cũ rồi **Thêm kênh** mới.

## Chiến lược đằng sau các lựa chọn này

Xem [youtube-channel-setup.md](youtube-channel-setup.md) — vì sao một kênh một
ngôn ngữ, vì sao tiếng Việt trước, và cách phủ tiếng Anh sau bằng multi-language
audio thay vì mở kênh thứ hai.
