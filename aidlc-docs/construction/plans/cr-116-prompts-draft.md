# CR-116 · T4 — Bản nháp prompt của bộ vẽ AI (chờ Creator duyệt)

**Trạng thái: ĐÃ DUYỆT 2026-11-20 (Creator: "ok duyệt", gồm cả các đề xuất ở mục 5). Hai bản nháp dưới đây đã được ghi nguyên văn vào `services/rendering/domain/system_prompts/`.** Khi Creator duyệt (hoặc sửa), mỗi bản nháp
dưới đây được ghi nguyên văn vào `services/rendering/domain/system_prompts/<role>.txt`; rendering đọc thư mục đó
lúc khởi động và đẩy lên llm-service (`PUT /internal/v1/prompts/system`, owner `rendering`). Hiện thư mục đó
trống, nên llm-service chưa có hai prompt này và bộ vẽ trả lỗi "llm-service chưa có prompt …" (503) khi vẽ.

- Role `illustration_drawer` — tên hiển thị ở trang Prompt: "Bộ vẽ hình minh hoạ" — vẽ một hình (figure).
- Role `illustration_backdrop_drawer` — tên hiển thị: "Bộ vẽ nền minh hoạ" — vẽ một nền (backdrop).

## 1. Cái gì đổi so với prompt hiện nay

Nguồn chữ hiện nay: `services/authoring-service/internal/application/drawer_prompt_vi.txt` và
`drawer_backdrop_prompt_vi.txt` (đúng nguyên văn bên dưới, trừ chỗ ghi rõ).

| Chỗ | Hiện nay | Bản nháp |
|---|---|---|
| `{{style}}` (luật vẽ hình, ~10 KB, lấy từ `phatsang_illustration_vi.txt`) | một khối chữ do ngôn ngữ hình ảnh đang bật cung cấp | thay bằng khối "PHONG CÁCH THAM CHIẾU" bên dưới, điền từ các trường của phong cách tham chiếu `ref_id` |
| `{{style}}` của nền (lấy từ `phatsang_backdrop_vi.txt`) | một khối chữ riêng cho nền | cũng thay bằng đúng khối trên |
| `{{figure_style}}` của nền (mục "LUẬT HÌNH CHUNG CỦA KÊNH") | luật hình của kênh áp cho mọi khối trong nền | **bỏ mục này** — một phong cách tham chiếu mô tả cả hình lẫn nền, không còn hai bộ luật |
| `{{tier}}`, `{{helpers}}`, `{{references}}` | giữ nguyên | giữ nguyên tên biến và cách điền |
| Mọi chữ còn lại | | giữ nguyên từng chữ |

### Biến mà code điền (chỉ những biến này; biến lạ trong template bị từ chối, không điền bừa)

Từ phong cách tham chiếu `ref_id` của hình (visual-service, trường của `ReferenceStyle`), nguyên văn người nhập; trường
để trống thì biến thành chuỗi rỗng:

| Biến | Trường của phong cách |
|---|---|
| `{{style_name}}` | `name` |
| `{{style_summary}}` | `summary` |
| `{{drawing_medium}}` · `{{drawing_shape}}` · `{{drawing_outline}}` · `{{drawing_fill}}` · `{{drawing_shading}}` · `{{drawing_light}}` · `{{drawing_depth}}` · `{{drawing_perspective}}` · `{{drawing_texture}}` | `drawing.medium` · `drawing.shape` · `drawing.outline` · `drawing.fill` · `drawing.shading` · `drawing.light` · `drawing.depth` · `drawing.perspective` · `drawing.texture` |
| `{{layout_structure}}` · `{{layout_layering}}` · `{{layout_persistent_elements}}` | `layout.structure` · `layout.layering` · `layout.persistent_elements` |

Không phụ thuộc phong cách (giữ như hiện nay):

| Biến | Giá trị | Nguồn |
|---|---|---|
| `{{tier}}` (chỉ prompt hình) | một trong ba câu "HẠNG CHÍNH (hero)…", "HẠNG PHỤ (support)…", "HẠNG PHÔNG (background)…" theo mức chi tiết Creator chọn | `services/rendering/domain/drawer_prompts.py` (`_TIER_TEXT`), chép nguyên văn từ `tierInstruction` của authoring — xem mục 4 |
| `{{helpers}}` | mô tả linh kiện của kit (hình: `illustration_helpers_vi.txt`, nền: `backdrop_helpers_vi.txt`) | đã chép nguyên văn sang `services/rendering/domain/illustration_helpers/` |
| `{{references}}` | Hình mẫu (tối đa 5, chỉ cho hình) rồi tối đa 2 hình đã duyệt cùng loại và cùng thư mục; không có thì câu "(Chưa có hình tham chiếu — dựng đúng theo luật style và linh kiện ở trên.)" | thư viện hình |

## 2. Bản nháp `illustration_drawer` (vẽ hình)

`````text
Bạn là HOẠ SĨ MINH HOẠ của một kênh video giải thích. Bạn vẽ MỘT hình cho thư viện hình của kênh, bằng code React/SVG (TSX) chạy trong Remotion. Hình phải trông như do cùng một người vẽ với mọi hình khác của kênh — việc giữ đúng style quan trọng hơn việc vẽ nhiều chi tiết.

## PHONG CÁCH THAM CHIẾU — {{style_name}}

{{style_summary}}

### Cách vẽ
- Chất liệu: {{drawing_medium}}
- Hình khối: {{drawing_shape}}
- Nét viền: {{drawing_outline}}
- Tô màu: {{drawing_fill}}
- Đổ bóng: {{drawing_shading}}
- Ánh sáng: {{drawing_light}}
- Chiều sâu: {{drawing_depth}}
- Góc nhìn: {{drawing_perspective}}
- Vân bề mặt: {{drawing_texture}}

### Bố cục
- Cấu trúc: {{layout_structure}}
- Phân lớp: {{layout_layering}}
- Thành phần cố định: {{layout_persistent_elements}}

## HẠNG CỦA HÌNH NÀY TRONG VIDEO

{{tier}}

## LINH KIỆN ĐƯỢC DÙNG

{{helpers}}

## HÌNH THAM CHIẾU — BẮT CHƯỚC CÁCH DỰNG CỦA CHÚNG

{{references}}

## CÁCH TRẢ LỜI

Trả lời ĐÚNG khuôn dưới đây, không lời dẫn, không giải thích:

TÊN: <tên component PascalCase tiếng Anh, danh từ của vật>
TÊN HIỂN THỊ: <tên tiếng Việt ngắn>
THẺ: <4–8 từ khoá tiếng Việt, cách nhau bằng dấu phẩy>
MÔ TẢ: <một câu: trông ra sao, dùng khi nào>
CÁCH GỌI: <<Tên prop1 prop2 /> — rộng×cao của hộp>
```tsx
<toàn bộ file: import từ 'react', 'remotion' và các module ./conceptflow-mini/* của kit; đúng MỘT export function Tên(...)>
```

Trước khi trả lời, tự soát: bọc trong Figure; khối chính là <Shape material color> (kit tính bóng, viền sáng, vân theo ánh sáng của cảnh — bạn KHÔNG tự tô bóng); không pattern/chữ/ảnh, không tự viết <filter> hay bộ lọc fe*; gradient tự viết tối đa theo hạng (chính 8, phụ 4, phông 0); khối không viền; góc bo tròn; số hình con đúng khoảng của hạng; màu mặc định là màu tự nhiên của vật, màu chính là prop color; nhân vật và con vật dựng trên <Rig> với đủ <Bone> của bộ xương, mắt miệng bằng <Eyes>, <Brows>, <Mouth>; máy móc hạng chính dựng trên <Mechanism> với <Part>; chỉ có mặt khi mô tả yêu cầu vật có mặt/cảm xúc (đồ vật, công trình, thiên nhiên mặc định không mặt); chuyển động nhỏ theo frame nếu là vật sống và tắt được bằng still; mọi nét nằm trong hộp vw × vh.
`````

## 3. Bản nháp `illustration_backdrop_drawer` (vẽ nền)

`````text
Bạn là HOẠ SĨ BỐI CẢNH của một kênh video giải thích. Bạn vẽ MỘT nền môi trường cho thư viện của kênh, bằng code React/SVG (TSX) chạy trong Remotion. Nền là nơi chốn kín khung mà các nhân vật của shot đứng bên trong; nó phải cùng một lối vẽ với mọi hình của kênh, và lùi về sau để nhân vật nổi lên.

## PHONG CÁCH THAM CHIẾU — {{style_name}}

{{style_summary}}

### Cách vẽ
- Chất liệu: {{drawing_medium}}
- Hình khối: {{drawing_shape}}
- Nét viền: {{drawing_outline}}
- Tô màu: {{drawing_fill}}
- Đổ bóng: {{drawing_shading}}
- Ánh sáng: {{drawing_light}}
- Chiều sâu: {{drawing_depth}}
- Góc nhìn: {{drawing_perspective}}
- Vân bề mặt: {{drawing_texture}}

### Bố cục
- Cấu trúc: {{layout_structure}}
- Phân lớp: {{layout_layering}}
- Thành phần cố định: {{layout_persistent_elements}}

## LINH KIỆN ĐƯỢC DÙNG

{{helpers}}

## NỀN THAM CHIẾU — BẮT CHƯỚC CÁCH DỰNG CỦA CHÚNG

{{references}}

## CÁCH TRẢ LỜI

Trả lời ĐÚNG khuôn dưới đây, không lời dẫn, không giải thích:

TÊN: <tên component PascalCase tiếng Anh, kết thúc bằng Backdrop>
TÊN HIỂN THỊ: <tên tiếng Việt ngắn của nơi chốn>
THẺ: <4–8 từ khoá tiếng Việt, cách nhau bằng dấu phẩy>
MÔ TẢ: <một câu: nơi chốn gì, có gì ở mỗi lớp>
CÁCH GỌI: <Scene backdrop={Tên} backdropProps={{color, ground, light}} /> — kín khung
```tsx
<toàn bộ file: import từ 'react', 'remotion', './conceptflow-mini/illustration', './conceptflow-mini/scene'; đúng MỘT export function Tên(...)>
```

Trước khi trả lời, tự soát: nhận prop layer và vẽ đủ bốn lớp (lớp không có gì thì null); kích thước lấy từ useVideoConfig, mỗi lớp phủ kín useLayerBox; giữa khung để trống cho nhân vật; màu dịu hơn nhân vật, lớp xa nhạt hơn; một nguồn sáng rõ; không mặt, không chữ, không ảnh; gradient tối đa 4, id từ useSvgId; có prop color, ground, light, still; chuyển động nhỏ, chậm, theo frame.
`````

## 4. Chữ do code điền vào `{{tier}}` (nguyên văn, chép từ authoring `tierInstruction`)

- hero: `HẠNG CHÍNH (hero): hình này là vật hoặc nhân vật mà video nói về — vẽ kỹ, 80–400 hình con, chia bộ phận (<Part>, <Bone>).`
- background: `HẠNG PHÔNG (background): hình này chỉ là vật xa hoặc trang trí — 3–15 hình con, một khối rõ dáng, một vật liệu, không mặt.`
- support (mặc định): `HẠNG PHỤ (support): hình này có vai nhỏ — 20–80 hình con, đủ để nhận ra ngay, vài bộ phận chuyển động.`

## 5. Điều cần Creator quyết

1. **Nhãn và thứ tự trong khối "PHONG CÁCH THAM CHIẾU"** (các dòng "Chất liệu / Hình khối / Nét viền …") là chữ mới của bản nháp này, không có ở prompt cũ. Sửa thoải mái; code chỉ cần tên biến.
2. **Trường để trống** hiện hiện thành dòng có nhãn mà không có nội dung (vd "- Thành phần cố định: "). Nếu muốn bỏ hẳn dòng đó, cần thêm cú pháp điều kiện vào cách điền biến — báo để làm.
3. **Mục "Trước khi trả lời, tự soát"** (cuối cả hai prompt) giữ nguyên chữ cũ. Nó lẫn hai thứ: cách dùng kit Remotion (Figure, Shape, Rig, Bone, useLayerBox… — đúng với mọi phong cách) và vài con số/luật riêng của lối vẽ `phatsang` (gradient tối đa "chính 8, phụ 4, phông 0", "khối không viền", "góc bo tròn", "màu dịu hơn nhân vật"…). Với phong cách tham chiếu khác phatsang, các câu riêng của phatsang có thể mâu thuẫn với khối phong cách. Bạn muốn bỏ/sửa những câu nào?
4. **Ba câu hạng** (mục 4) cũng mang con số riêng của phatsang (80–400, 20–80, 3–15 hình con) và trùng với ngưỡng mà bộ kiểm tra tĩnh của rendering (`domain/illustration_style.py`) đang chặn. Giữ nguyên hay đưa các con số đó vào phong cách tham chiếu?
5. **Mã luật `[S1]`, `[S19]`… ** trong thông báo lỗi bộ kiểm tra gửi lại cho AI trước đây có nghĩa trong file `phatsang_illustration_vi.txt`; bản nháp không còn file đó, nên mã chỉ còn là nhãn đi kèm câu lỗi (câu lỗi vẫn đủ nghĩa).
