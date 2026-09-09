# Prompt tạo avatar kênh YouTube ConceptFlow

Kênh đăng video giải thích khái niệm bằng animation Manim (toán/khoa học), lời dẫn
Việt + Anh. Avatar nên gợi: chuyển động toán học, mạch lạc, sạch.

## Ràng buộc kỹ thuật (quyết định luôn phần thiết kế)

| | |
|---|---|
| Kích thước upload | **800 × 800 px**, PNG/JPEG, < 4 MB |
| Hiển thị | **crop tròn** — 4 góc luôn bị cắt mất |
| Nơi nhỏ nhất | ~32 px (bình luận, gợi ý video) |

Ba hệ quả, và đây là lý do prompt bên dưới viết như vậy:

1. **Chủ thể phải nằm giữa, chừa lề rộng.** Bất cứ gì sát mép đều mất khi crop tròn.
2. **Một hình khối duy nhất, tương phản mạnh.** Ở 32 px mọi chi tiết nhỏ biến thành
   nhiễu. Thử thu nhỏ ảnh xuống 32 px, còn nhận ra được thì đạt.
3. **Đừng bắt AI vẽ chữ.** Mọi công cụ ảnh AI đều viết chữ méo, và ở 32 px thì chữ
   cũng không đọc nổi. Cần chữ thì thêm sau bằng Canva/Figma.

Màu lấy từ chính app (`--accent` / `--accent2` trong web-gui):
**`#2f8fe0`** (xanh dương) → **`#6dd5fa`** (xanh cyan), nền xanh đen đậm.

---

## Đường A — ImageFX / Gemini (khuyến nghị: ra thẳng ảnh vuông)

Vào https://labs.google/fx/tools/image-fx, chọn tỉ lệ **1:1**, dán prompt:

### Phương án 1 — Dải ruy băng ánh sáng (an toàn nhất, dễ nhận ở cỡ nhỏ)

```
A single glowing ribbon of light twisting into a smooth Möbius loop, centered
on a deep navy background. Gradient from #2f8fe0 blue to #6dd5fa cyan, soft
volumetric glow, subtle particle dust. Minimal, geometric, high contrast,
symmetrical composition, large empty margin around the subject. Flat vector
illustration style, no text, no letters, square 1:1.
```

### Phương án 2 — Sóng sin hoá thành vòng tròn (đúng tinh thần "concept flow")

```
An elegant sine wave curving around and closing into a perfect circle, drawn as
a single luminous stroke on a deep navy background. Cyan-to-blue gradient
(#6dd5fa to #2f8fe0), thin precise linework, faint grid in the background at very
low opacity. Mathematical, minimal, centered with generous padding, high
contrast. Vector style, no text, square 1:1.
```

### Phương án 3 — Đồ thị nút sáng (hợp nếu nội dung thiên về khái niệm/liên hệ)

```
A small constellation of glowing nodes connected by thin light edges, arranged
in a balanced circular cluster on a dark navy background. Nodes glow cyan
#6dd5fa, edges fade to blue #2f8fe0. Clean, symmetrical, centered, lots of
negative space around the cluster. Minimal scientific diagram aesthetic,
no text, square 1:1.
```

### Phương án 4 — Khối 3D tối giản (nổi bật nhất ở 32 px)

```
A single torus knot rendered in smooth glossy material, floating centered on a
deep navy background, lit with cyan and blue rim light (#6dd5fa, #2f8fe0).
Soft studio lighting, subtle reflection, dramatic contrast, generous empty space
around the object. Clean 3D render, minimal, no text, square 1:1.
```

## Đường B — Google Flow (nếu muốn dùng đúng công cụ này)

https://labs.google/fx/tools/flow — Flow xuất **video**, nên quy trình dài hơn:

1. Chọn tỉ lệ **16:9**
2. Dán prompt bên dưới (đã thêm mô tả chuyển động, vì Flow cần biết cảnh động thế nào)
3. Tạo clip ~4–8 giây
4. **Tạm dừng đúng khung hình đẹp nhất** → chụp màn hình, hoặc tải video về rồi trích frame
5. **Crop vuông vào chính giữa** → resize 800×800

```
A single luminous ribbon of light slowly twisting into a Möbius loop, rotating
gently in place at the exact center of the frame. Deep navy background, gradient
from #2f8fe0 blue to #6dd5fa cyan, soft volumetric glow, slow drifting particle
dust. Locked-off static camera, no camera movement, subject perfectly centered
with wide empty margins on all sides. Minimal geometric motion-graphics style,
seamless loop, no text, no letters, no people.
```

> Hai câu quan trọng nhất trong prompt trên là **"locked-off static camera"** và
> **"perfectly centered with wide empty margins"**. Thiếu chúng, Veo sẽ đẩy camera
> chạy và chủ thể trôi khỏi khung — lúc crop vuông là mất đầu mất đuôi.

Trích frame bằng ffmpeg (đã có sẵn trong dự án):

```bash
# Lấy frame tại giây thứ 2, crop vuông giữa, resize 800x800
ffmpeg -i flow_output.mp4 -ss 2 -vframes 1 \
  -vf "crop=ih:ih:(iw-ih)/2:0,scale=800:800" avatar.png
```

## Nên tránh trong prompt

Những thứ này làm hỏng avatar ở cỡ nhỏ, dù ảnh xem full trông đẹp:

- `detailed`, `intricate`, `complex` — thành nhiễu ở 32 px
- `text`, `logo`, `typography`, tên kênh — AI viết chữ méo
- `close-up`, `dynamic camera`, `dutch angle` — chủ thể lệch khỏi tâm
- Nhiều hơn 2–3 màu — mất tương phản khi thu nhỏ
- Mặt người / nhân vật — biến dạng ở cỡ nhỏ, và không hợp nội dung

## Kiểm tra trước khi upload

1. Thu ảnh xuống **32 × 32 px** — còn nhận ra hình không?
2. Crop tròn thử (Figma/Canva, hoặc xem preview của YouTube) — có mất phần nào quan trọng?
3. Xem trên **nền trắng và nền tối** — YouTube dùng cả hai
4. Upload: YouTube Studio → **Customization** → **Branding** → **Picture**

---

# Prompt tạo banner kênh (channel art)

> **Đã có banner dựng sẵn** trong `docs/brand/banner-*-2560x1440.png`, ghép từ ảnh
> torus và đã căn đúng vùng an toàn. Phần dưới dành cho khi bạn muốn làm bản mới.

Banner khó hơn avatar nhiều, vì YouTube cắt nó **ba kiểu khác nhau** tuỳ thiết bị
từ cùng một file. Hiểu điều này trước rồi hãy viết prompt.

## Kích thước và ba vùng cắt

| | |
|---|---|
| File upload | **2560 × 1440 px**, tỉ lệ 16:9, < 6 MB |
| **Vùng an toàn TV** | toàn bộ 2560 × 1440 — chỉ TV mới thấy hết |
| Máy tính | 2560 × 423 (dải ngang giữa) |
| **Điện thoại** | **1546 × 423 ở chính giữa** ← chỗ đông người xem nhất |

Nghĩa là: **mọi thứ quan trọng phải nằm trong ô 1546 × 423 ở giữa.** Phần còn lại
chỉ là nền loang ra hai bên. Đây là lỗi phổ biến nhất — thiết kế đẹp trên máy tính,
lên điện thoại mất chữ.

Với ảnh AI, cách chắc ăn: sinh ảnh **16:9 với chủ thể dồn hết vào giữa và hai bên
để trống**, rồi thêm chữ sau bằng Canva/Figma đúng trong ô an toàn.

## Prompt banner — đồng bộ với avatar torus

### Phương án 1 — Torus trôi trong không gian tối (khớp avatar nhất)

```
An ultra-wide cinematic banner, 16:9. A glossy liquid-mercury torus floating at
the exact center, chrome surface with flowing ripples, lit by cyan #6dd5fa and
blue #2f8fe0 rim light. Deep navy background fading to near-black toward the far
left and right edges, faint drifting particles, soft volumetric glow. The torus
occupies only the central third of the frame; the outer thirds are empty dark
gradient. Cinematic, minimal, high contrast, no text, no letters.
```

### Phương án 2 — Dòng chảy ngang (gợi chuyển động của kênh)

```
An ultra-wide cinematic banner, 16:9. Streams of glowing liquid light flowing
horizontally across a deep navy void, converging into a bright luminous ring at
the exact center of the frame, then dispersing again. Cyan #6dd5fa to blue
#2f8fe0 gradient, soft bloom, faint particles. Dark and empty toward the left
and right edges, all detail concentrated in the middle. Cinematic, minimal,
high contrast, no text, no letters.
```

### Phương án 3 — Lưới toán học mờ (thiên về giáo dục)

```
An ultra-wide cinematic banner, 16:9. A vast dark navy space with a faint
mathematical grid receding into depth, and a single glowing chrome torus resting
at the exact center, lit cyan #6dd5fa and blue #2f8fe0. Thin luminous curve
lines sweeping through the grid. Empty and dark at the far left and right,
detail concentrated centrally. Cinematic, minimal, no text, no letters.
```

> Câu quyết định thành bại là **"the torus occupies only the central third of the
> frame; the outer thirds are empty dark gradient"**. Thiếu nó, AI sẽ rải chi tiết
> khắp khung, và bản cắt trên điện thoại sẽ xén mất đúng phần đẹp nhất.

## Sau khi có ảnh

Ảnh AI ra 16:9 nhưng chủ thể chiếm gần hết khung — cắt thẳng sang 2560×1440 thì
trên điện thoại chỉ còn một lát ngang của vòng xuyến. Script dưới đây làm đúng
việc đó cho tử tế: thu nhỏ vòng xuyến vừa dải an toàn, thổi nền mờ loang ra hai
bên, rồi đặt tên kênh cạnh nó — tất cả nằm gọn trong ô 1546 × 423.

```bash
pip install Pillow
python3 docs/brand/make-banner.py
```

Xuất ra hai file:

| File | Dùng để |
|---|---|
| `docs/brand/conceptflow-banner-2560.jpg` | **mang đi upload** (2560 × 1440, ~210 KB) |
| `docs/brand/conceptflow-banner-mobile-preview.png` | 1546 × 423 — đúng phần điện thoại hiển thị |

Sửa tên kênh, câu tagline, cỡ chữ hay vị trí vòng xuyến ngay trong
`make-banner.py` rồi chạy lại. Script tự kiểm tra cụm chữ có tràn khỏi ô an toàn
không và báo lỗi nếu tràn.

Mở `conceptflow-banner-mobile-preview.png` trước khi upload. Những gì bạn thấy ở
đó **là toàn bộ** thứ đa số người xem sẽ thấy.

Upload: YouTube Studio → **Customization** → **Branding** → **Banner image**.
