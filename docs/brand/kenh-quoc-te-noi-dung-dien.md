# Kênh quốc tế — nội dung điền vào YouTube Studio

**Dùng cho SAU NÀY**, khi mở kênh tiếng Anh riêng. Kênh đang ưu tiên là bản tiếng
Việt: [kenh-viet-noi-dung-dien.md](kenh-viet-noi-dung-dien.md).

Đường dẫn: **YouTube Studio → Customization → Profile** (của kênh tiếng Anh, không
phải kênh tiếng Việt).

---

## Đọc trước khi mở kênh này

Đừng mở kênh thứ hai quá sớm. Ba dấu hiệu cho thấy đã đến lúc:

1. Kênh tiếng Việt đã đều đặn (≥ 20 video, lịch đăng ổn định) — mở kênh thứ hai
   khi chưa vững sẽ làm hụt hơi cả hai
2. Analytics cho thấy lượng người xem nước ngoài đáng kể dù nội dung tiếng Việt
3. Bạn kham được **gấp đôi** khối lượng: 2 lịch đăng, 2 bộ metadata, 2 phần bình luận

**Trước khi tới lúc đó**, cách phủ khán giả quốc tế rẻ hơn nhiều là phụ đề tiếng
Anh + mô tả tiếng Anh ngay trên kênh Việt (đã có trong file bản Việt).

Lưu ý cạnh tranh: mảng animation toán tiếng Anh có 3Blue1Brown, Veritasium,
Numberphile. Kênh mới cần một góc riêng rõ ràng — chẳng hạn tập trung vào một
lĩnh vực hẹp, hoặc video ngắn hơn, hoặc trình độ nhập môn hơn.

---

## Picture

Upload: `docs/brand/conceptflow-avatar-800.png` — **dùng chung ảnh với kênh Việt**.
Cùng nhận diện thì người xem chuyển qua lại vẫn biết là một nhà.

## Name

```
ConceptFlow
```

Nếu YouTube báo trùng với kênh Việt của chính bạn, phân biệt bằng hậu tố ngôn ngữ
(đây là quy ước phổ biến, người xem hiểu ngay):

```
ConceptFlow EN
```

## Handle

```
@conceptflow.en
```

Phương án thay thế:

```
@conceptflowmath
@conceptflow.math
```

## Description

Hai dòng đầu là phần hiện trong kết quả tìm kiếm. Từ khoá tiếng Anh của mảng này
là *visual*, *intuition*, *explained* — người tìm gõ đúng những chữ đó.

```
Visual explanations of mathematics and science. Every video turns an abstract idea into something you can actually see and reason about.

What you'll find here:
• Mathematics, visualised — calculus, linear algebra, probability, geometry
• Scientific concepts shown as motion, not as walls of formulas
• Clear, concise explanations that respect your time

Animations are built with Manim, the open-source mathematical animation library.

New video every week. Subscribe so you don't miss one.

Business enquiries: <email của bạn>
```

## Add language (bản dịch mô tả)

Bấm **Add language → Vietnamese**, dán:

```
Giải thích các khái niệm toán học và khoa học bằng animation trực quan.

Lời dẫn của kênh này bằng tiếng Anh, có phụ đề tiếng Việt. Nếu bạn muốn nội dung lời dẫn tiếng Việt, xem kênh ConceptFlow tiếng Việt.

Video được dựng bằng Manim — thư viện animation toán học mã nguồn mở.
```

> Câu thứ hai làm hai việc: báo trước lời dẫn là tiếng Anh (tránh người Việt vào
> rồi thoát ngay), và trỏ họ sang kênh Việt. Trỏ chéo giữa hai kênh giữ người xem
> ở lại hệ sinh thái của bạn thay vì bỏ đi.

## Links

| Link title | URL |
|---|---|
| Vietnamese channel | `https://www.youtube.com/@conceptflow` ← trỏ chéo, quan trọng |
| GitHub | `https://github.com/ngoc31031997/ConceptFlow` |
| Website | *(nếu có)* |

Trên kênh Việt cũng thêm link ngược lại về kênh EN.

## Contact info

```
<email của bạn>
```

Dùng **chung một email** với kênh Việt — gộp liên hệ hợp tác về một chỗ.

---

## Sau khi Publish

### 1. Ngôn ngữ kênh
**Settings → Channel → Advanced settings**
- Language: **English** ← khác kênh Việt, đây là điểm mấu chốt
- Country of residence: **Vietnam** (nơi bạn ở, không giới hạn ai xem được)

Khai sai ô Language là lỗi nặng nhất với kênh quốc tế: YouTube sẽ đẩy kênh cho
người Việt thay vì người nói tiếng Anh.

### 2. Nối kênh vào ConceptFlow

Kênh này cần **cấp quyền OAuth riêng** — token YouTube gắn với đúng một kênh. Vào
màn kết quả trong app → **Thêm kênh** → chọn kênh EN trên màn hình của Google.

Sau đó app sẽ có hai kênh trong danh sách, chọn kênh khi đăng từng video.

> **Cân nhắc quota**: hai kênh nối qua **cùng một** file `client_secret` sẽ **dùng
> chung** hạn mức ~6 video/ngày. Nếu định đăng nhiều, tạo GCP project thứ hai và
> thả file client_secret thứ hai vào `secrets/` — xem
> [`docs/setup/youtube-oauth-client.md`](../setup/youtube-oauth-client.md).

### 3. Quy trình nội dung

Với mỗi video, `content_language = en` khi tạo trong app (CR-008) — toàn bộ
pipeline sẽ dùng tiếng Anh: lời dẫn, phụ đề, gợi ý metadata.

**Không** đăng lại y nguyên video tiếng Việt kèm phụ đề Anh lên kênh EN. Kênh EN
cần lời dẫn tiếng Anh thật, nếu không tỉ lệ giữ chân sẽ rất thấp.

## Chiến lược đằng sau

Xem [youtube-channel-setup.md](youtube-channel-setup.md).
