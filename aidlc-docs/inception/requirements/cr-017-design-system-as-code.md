# CR-017 — Design system `conceptflow`: khóa bản sắc kênh vào code (P1)

## Date
2026-09-09

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: New Feature + Refactoring — thêm một thư viện, đổi bề mặt API mà script được phép dùng
- **Scope estimate**: 2 unit — `rendering` (thư viện + lint), `web-gui` (prompt và template)
- **Complexity estimate**: Moderate. Kỹ thuật đơn giản; phần khó là **chọn đúng bộ component** — chọn hẹp quá thì bóp nghẹt nội dung, rộng quá thì không khóa được style

## Bối cảnh — quan sát được
Mỗi video được sinh ra bởi một model bên ngoài, từ một prompt tự do. Đã xác minh:

- `scriptPrompts.ts::buildGenerationSystemPrompt` giao cho model toàn quyền:
  *"Bạn được toàn quyền sáng tạo về: cách ví von, ví dụ cụ thể, màu sắc, bố cục,
  thứ tự trình bày."*
- Bảng màu duy nhất tồn tại nằm trong `scriptTemplates.ts` — một **template
  khởi đầu** mà Creator có thể thay hoàn toàn, không phải một ràng buộc.
- `script_lint.py::INVALID_KWARGS_BY_CLASS` là **blacklist 5 class**, tự nhận
  trong docstring là "hand-maintained list... can never be exhaustive".
- Prompt phải dành hẳn một mục "LỖI API MANIM THƯỜNG GẶP — TUYỆT ĐỐI TRÁNH" để
  dạy model tránh `corner_radius` trên `Rectangle`. Tức là hệ thống đang dùng
  **văn xuôi trong prompt** làm cơ chế kiểm soát API.

## Vấn đề
1. **Không có bản sắc kênh.** Video số 5 và video số 20 dùng bảng màu khác,
   cỡ chữ khác, nhịp animation khác. Người xem không nhận ra đó là cùng một
   kênh. Đây là điểm phân biệt rõ nhất giữa một kênh chuyên nghiệp và một chuỗi
   video rời rạc.
2. **Kiểm soát API bằng blacklist là cuộc đua không thắng được.** Manim có hàng
   trăm class; danh sách tay sẽ luôn chạy sau lỗi mới.
3. **Prompt đang gánh việc của code.** Mọi ràng buộc style và API đều là câu chữ
   trong một chuỗi copy ra ngoài — không kiểm chứng được, không version được,
   không test được.

Tham chiếu: 3Blue1Brown ổn định style không nhờ viết prompt giỏi, mà nhờ có
`constants.py` và một thư viện scene cá nhân. **Style của kênh phải nằm trong
code.**

## Quyết định
Tạo package Python `conceptflow`, cài sẵn trong image của `rendering`. Script do
AI sinh ra mở đầu bằng `from conceptflow import *` thay cho `from manim import *`.

Thư viện gồm ba tầng:

| Tầng | Nội dung | Vai trò |
|---|---|---|
| `theme` | Palette, thang cỡ chữ, font, safe margin, hằng số nhịp animation | Một nơi duy nhất định nghĩa "trông như thế nào" |
| `ConceptFlowScene` | Base scene: tự set background/camera/font mặc định; helper bố cục | Mọi video kế thừa cùng một nền |
| Components | `TitleCard`, `Callout`, `CodePanel`, `StepList`, `ComparisonSplit`, `Recap` | Từ vựng dựng cảnh, thay cho việc ghép mobject thô |

**Lợi ích kép, và đây là lý do chính chọn hướng này:** một bề mặt API hẹp biến
lint từ **blacklist thành whitelist**. Thay cho việc liệt kê từng kwarg sai của
từng class Manim, luật trở thành một câu: *script chỉ được gọi API thuộc
namespace `conceptflow`*. Vừa khóa được style, vừa xoá gần hết class lỗi API mà
prompt đang phải cảnh báo dài dòng.

## Functional Requirements

### FR44 — Theme là nguồn sự thật duy nhất về hình thức
- **FR44.1**: PHẢI có module `conceptflow.theme` khai báo: bảng màu có tên
  (nền, chữ chính, nhấn, cảnh báo, mờ), thang cỡ chữ có tên (H1/H2/BODY/CAPTION),
  họ font, safe margin, và bộ hằng số thời lượng animation.
- **FR44.2**: Đổi bản sắc kênh PHẢI chỉ cần sửa module này, không sửa script nào.
- **FR44.3**: Theme PHẢI chọn được theo project (ít nhất một theme mặc định),
  để một Creator vận hành nhiều kênh không phải fork thư viện.
- **FR44.4**: Font PHẢI hỗ trợ tiếng Việt đầy đủ. Đây là ràng buộc đã ghi ở
  CR-006 §C3 và vẫn còn hiệu lực.

### FR45 — Base scene và thư viện component
- **FR45.1**: PHẢI có `ConceptFlowScene(Scene)` tự áp dụng theme trong
  `construct` mà script không phải làm gì.
- **FR45.2**: PHẢI cung cấp component cho các khuôn hình lặp lại nhiều nhất
  trong video giáo dục: thẻ tiêu đề, khối code, danh sách bước, so sánh hai cột,
  chú thích nhấn mạnh, màn tóm tắt.
- **FR45.3**: Component PHẢI tự đảm bảo nằm trong safe margin và tự chọn cỡ chữ
  theo thang của theme — Creator/AI KHÔNG truyền toạ độ tuyệt đối hay cỡ chữ thô.
- **FR45.4**: PHẢI cung cấp từ vựng chuyển cảnh chuẩn (`reveal`, `swap`,
  `emphasize`, `dismiss`) với `run_time` lấy từ hằng số theme, để nhịp phim đồng
  nhất giữa các video.
- **FR45.5**: Thư viện PHẢI cho phép "thoát hiểm" — truy cập Manim thô khi
  component không đủ diễn đạt. Không có đường thoát thì thư viện sẽ chặn đúng
  những video tham vọng nhất. Đường thoát này PHẢI bị lint cảnh báo (FR46.3),
  không bị cấm.

### FR46 — Lint chuyển từ blacklist sang whitelist
- **FR46.1**: `script_lint` PHẢI báo lỗi khi script gọi mobject/animation không
  thuộc bề mặt API công khai của `conceptflow`.
- **FR46.2**: `INVALID_KWARGS_BY_CLASS` PHẢI được gỡ bỏ khi whitelist đã phủ —
  giữ cả hai là giữ hai nguồn sự thật.
- **FR46.3**: Dùng đường thoát ở FR45.5 PHẢI sinh cảnh báo có nêu tên API thô
  được dùng, không chặn render.
- **FR46.4**: Màu hex viết thẳng và `font_size` là số thô nằm ngoài thang theme
  PHẢI bị cảnh báo — đây là hai cách phổ biến nhất để style trôi khỏi chuẩn.
- **FR46.5**: PHẢI cảnh báo khi một beat (CR-019) chỉ chứa `Text`/`MathTex` mà
  không có bất kỳ đối tượng hình học nào. Đây là cách biến yêu cầu "video nhiều
  ví dụ minh hoạ trực quan, không phải text đơn điệu" từ lời dặn trong prompt
  thành một kiểm tra thực sự chạy. Cảnh báo, không chặn: có beat (ví dụ `cta`)
  vốn dĩ chỉ là chữ.

### FR47 — Prompt và template đổi theo
- **FR47.1**: Prompt sinh script PHẢI được viết lại quanh API `conceptflow`:
  liệt kê component sẵn có thay cho việc giao "toàn quyền sáng tạo về màu sắc,
  bố cục".
- **FR47.2**: Mục "LỖI API MANIM THƯỜNG GẶP" trong prompt PHẢI được gỡ — nó tồn
  tại để bù cho việc không có whitelist.
- **FR47.3**: Template khởi đầu trong `scriptTemplates.ts` PHẢI viết lại theo
  API mới, và tiếp tục là ví dụ chuẩn mực cho model bắt chước.

## Non-goals
- Không fork Manim. `conceptflow` là một lớp mỏng bọc bên trên Manim CE 0.18.
- Không tự động chuyển đổi script cũ. Script viết theo API cũ vẫn render được
  (whitelist cảnh báo, không chặn) cho đến khi CR-018 buộc phải migrate.
- Không làm trình chọn theme trực quan ở GUI. FR44.3 chỉ cần chọn được bằng tên.

## Rủi ro
- **Bộ component chọn sai sẽ bóp nghẹt nội dung.** Đây là rủi ro lớn nhất và
  không giải quyết được bằng suy luận. Giảm thiểu: rà 5–10 script đã sản xuất,
  rút component từ khuôn hình **thực sự lặp lại**, không từ tưởng tượng.
- **Model chưa từng thấy API này.** Manim thì model biết rất rõ; `conceptflow`
  thì không. Prompt phải kèm đặc tả API đầy đủ, và chất lượng script sinh ra có
  thể giảm ở giai đoạn đầu. Cần đo trước khi kết luận.
- **Whitelist quá chặt sẽ đẩy mọi script vào đường thoát**, làm cả cơ chế thành
  hình thức. Nếu tỉ lệ cảnh báo FR46.3 cao, đó là tín hiệu component thiếu chứ
  không phải Creator sai.

## Quyết định đã chốt (2026-09-10)
1. **Bảng màu**: nền lấy từ bộ nhận diện đã có — `#080E1C` (chính là nền của
   `make-banner.py` và của logo) — dùng chung cho **cả** intro/outro lẫn thân
   video. Màu nhấn lấy từ **Catppuccin Mocha**. Lý do chọn cách lai này: Creator
   muốn giữ màu brand cho intro/outro (CR-023), nhưng hai bảng màu nền khác nhau
   sẽ giật màu ở đúng điểm nối; dùng chung nền thì mạch liền, mà nội dung vẫn
   mang bảng màu Catppuccin.
2. **Font**: `Cormorant Garamond` cho tiêu đề, `Be Vietnam Pro` cho thân và mọi
   thứ dưới cỡ H2, `JetBrains Mono` cho code. Cormorant chỉ dùng ở cỡ lớn — nét
   thanh của serif tương phản cao sẽ biến mất khi chữ nhỏ bị nén qua encode.
3. **Font PHẢI được kiểm dấu tiếng Việt trước khi chốt vào image**: render thử
   chuỗi có đủ dấu khó (`ỗ ự ằ ẩ ợ ữ ẵ ọ`) và xác nhận không có ô vuông hay dấu
   đặt sai. Không giả định theo mô tả trên Google Fonts.
4. **Một theme, nhưng kiến trúc cho nhiều theme ngay từ đầu** (FR44.3): theme là
   dữ liệu chọn được theo project, và theme mới tạo được bằng cách kế thừa rồi
   ghi đè, không phải chép lại toàn bộ.
5. **`conceptflow` nằm trong repo này**, cài vào image `rendering`.
6. **Công thức toán dùng mặc định của Manim** (`MathTex`, Computer Modern). Không
   ràng buộc phải giống 3Blue1Brown; đổi sau nếu thấy lệch với Cormorant.

## Câu hỏi cần Creator chốt
1. Bảng màu và font cụ thể của kênh — đã có bộ nhận diện chưa, hay lấy tạm bảng
   màu trong `scriptTemplates.ts` (Catppuccin) làm chuẩn đầu tiên?
2. Có cần nhiều theme ngay từ đầu (FR44.3) không, hay một theme là đủ?
3. `conceptflow` nằm trong repo này (thư mục dùng chung, cài vào image
   `rendering`) hay tách thành package riêng?

## Kiểm chứng
- Unit test: component nào cũng nằm trong safe margin ở mọi tỉ lệ khung được hỗ trợ.
- Unit test: lint bắt được API ngoài whitelist, hex thô, font_size thô; và
  KHÔNG báo nhầm trên template chuẩn.
- Thủ công: dựng lại một video đã có bằng API mới, so sánh cạnh nhau; đối chiếu
  hai video khác chủ đề để xác nhận chúng trông như cùng một kênh.

## Liên quan
- CR-006 §C3 (font tiếng Việt)
- CR-018 (`narrate()` — component chỉ mang được narration sau khi CR đó xong)
- CR-019 (beat sheet — dùng component của CR này để dựng hook/CTA)
- CR-020 (pre-flight gate — nơi lint whitelist được thực thi sớm)
- ADR sẽ cần: "Design system as code thay cho ràng buộc bằng prompt"
