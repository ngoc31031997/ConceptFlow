# CR-116 · Chuyển phần hình ảnh từ authoring-service sang rendering-service — câu hỏi yêu cầu

**Mục tiêu (đã chốt với Creator 2026-11-19):** authoring-service chỉ còn lo viết kịch bản. Thư viện hình minh hoạ, bộ vẽ AI, phong cách hình (visual language, kit, khung mẫu) chuyển sang rendering-service. Bước 11 Storyboard (CR-117) sẽ đọc thư viện hình từ rendering.

## Hiện trạng (kiểm kê trên main @ 4033706)

**Bảng trong DB `authoring`** (số dòng đang có trên máy Creator):

| Bảng | Dòng | Ghi chú |
|---|---|---|
| `illustration_folders` | 14 | thư mục thư viện |
| `illustrations` | 293 | code hình, kind, tier, kit_key, ảnh xem trước PNG/GIF lưu thẳng trong bảng (cột `preview_png`, `preview_gif`) |
| `project_illustrations` | 0 | danh sách hình AI lập cho từng dự án |
| `visual_languages` | 1 | phong cách hình của kênh |
| `visual_language_versions` | 1 | các phiên bản của phong cách |
| `visual_language_samples` | 4 | khung mẫu đã vẽ |
| `project_authoring.visual_language_id` | (cột) | dự án gắn với phong cách nào |

**Route của authoring (33 route):**
- thư viện hình: `/v1/illustration-folders`, `/v1/illustrations*`, `/v1/illustration-tries`, `/v1/illustration-style`, `/v1/admin/illustrations*` (vẽ AI, vẽ lại, xuất/nhập, hình mẫu);
- hình theo dự án: `/v1/projects/{id}/illustrations*`;
- phong cách hình: `/v1/visual-languages*`, `/v1/projects/{id}/visual-language`, `/v1/admin/visual-languages*`.

**Bên gọi:**
- web-gui gọi qua api-gateway (`routes/illustrations.js`, `routes/visualLanguages.js`);
- orchestrator hỏi authoring `kit_key` của dự án để gửi lệnh render;
- bộ vẽ AI trong authoring gọi model qua `LLMProviderPort`;
- authoring gọi rendering `/v1/illustrations/preview`, `/v1/kits`, `/v1/stills`, `/v1/stills/prune`.

**Điểm khó chính:** authoring viết bằng Go, rendering viết bằng Python.

Trả lời bằng cách điền sau `[Answer]:`.

---

## Câu 1
Code chuyển sang rendering viết thế nào?

A) Viết lại bằng Python trong rendering-service — gộp vào cùng process đang chạy HTTP `/v1/check`, `/v1/stills`.
   - ✅ Một service, một ngôn ngữ; gọi vẽ/kiểm tra hình là gọi hàm nội bộ, không qua HTTP.
   - ⚠️ Viết lại toàn bộ logic Go (thư viện, bộ vẽ AI, phong cách, xuất/nhập) và test; CR lớn. Process rendering vốn đang chạy render video nặng, thêm API quản trị vào cùng process.

B) Chuyển nguyên code Go thành một chương trình Go thứ hai chạy trong container rendering, dùng DB `rendering`.
   - ✅ Không viết lại logic, giữ nguyên test.
   - ⚠️ Một container chạy hai ngôn ngữ, Dockerfile phức tạp; vẫn là hai process gọi nhau qua HTTP.

C) 💡 Gợi ý: tách thành một service mới (ví dụ `asset-service`, Go) có DB riêng, dùng rendering chỉ để vẽ/kiểm tra.
   - ✅ Chuyển code Go gần như nguyên vẹn; rendering giữ đúng vai trò "vẽ và render"; bước 11 có chỗ rõ ràng cho tài sản dự án.
   - ⚠️ Thêm một service và một DB phải vận hành; khác với hướng "đưa về rendering" đã nói.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Câu 2
Chuyển 293 hình, 14 thư mục, phong cách hình và 4 khung mẫu đang có thế nào?

A) Script chuyển một lần: đọc thẳng từ DB `authoring`, ghi vào DB mới, kiểm đếm số dòng hai bên khớp rồi mới xoá bảng cũ.
   - ✅ Giữ nguyên id, trạng thái, hình mẫu, ảnh xem trước; Creator không phải làm gì.
   - ⚠️ Cần một lần chạy có kiểm soát; nếu lỗi giữa chừng phải chạy lại (script phải chạy lại được an toàn).

B) Dùng chức năng xuất/nhập thư viện đã có (`/v1/admin/illustrations/export` → `import`).
   - ✅ Dùng lại thứ đã có và đã test.
   - ⚠️ Chỉ phủ thư viện hình; phong cách hình, khung mẫu, cột gắn dự án phải chuyển cách khác; ảnh xem trước phải vẽ lại.

X) Other (please describe after [Answer]: tag below)

[Answer]: xoá toàn bộ đi chúng ta sẽ vẽ lại từ đầu

## Câu 3
`project_illustrations` (danh sách hình AI lập cho từng dự án, hiện 0 dòng) trùng ý với "tài sản dự án `AST-…`" của bước 11. Xử lý thế nào?

A) Không chuyển: xoá bảng, route và màn hình này; CR-117 làm tài sản dự án từ đầu theo mô hình hai tầng đã chốt.
   - ✅ Không có hai khái niệm trùng nhau; không mất dữ liệu (đang 0 dòng).
   - ⚠️ Từ lúc CR-116 xong đến lúc CR-117 xong sẽ không có màn hình lập hình theo dự án.

B) Chuyển nguyên sang, rồi CR-117 mở rộng nó thành tài sản dự án.
   - ✅ Màn hình hiện tại vẫn chạy liên tục.
   - ⚠️ Chuyển code mà sắp phải sửa lớn; CR-117 bị ràng bởi cấu trúc cũ.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Câu 4
Dự án gắn với phong cách hình nào (`project_authoring.visual_language_id`) nằm ở đâu? Orchestrator đang hỏi authoring `kit_key` khi gửi lệnh render.

A) Chuyển việc gắn sang service mới chứa phong cách hình; orchestrator hỏi `kit_key` ở đó.
   - ✅ Authoring không còn biết gì về hình.
   - ⚠️ Sửa orchestrator; thêm một chỗ phải biết dự án tồn tại (xoá/nhân bản dự án phải báo sang).

B) Giữ cột gắn ở authoring (chỉ là một id), phong cách thì nằm ở service mới.
   - ✅ Ít đổi nhất; nhân bản/xoá dự án vẫn như cũ.
   - ⚠️ Authoring vẫn giữ một mẩu liên quan tới hình; id trỏ sang DB khác nên không còn khoá ngoại.

X) Other (please describe after [Answer]: tag below)

[Answer]: cái này nằm ở bước 8 ngôn ngữ hình có chỗ tham chiếu phong cách rồi mà

## Câu 5
URL mà web-gui gọi có đổi không?

A) Giữ nguyên các URL `/v1/illustrations*`, `/v1/visual-languages*`…; chỉ api-gateway đổi đích sang service mới.
   - ✅ web-gui không phải sửa; ít rủi ro.
   - ⚠️ Không có dịp dọn lại tên route.

B) Đổi tên/đặt lại route cho gọn, sửa cả web-gui.
   - ✅ API sạch hơn.
   - ⚠️ Sửa nhiều màn hình; CR lớn hơn mà Creator không thấy khác gì.

X) Other (please describe after [Answer]: tag below)

[Answer]:  A

## Câu 6
Bộ vẽ AI (hiện gọi model qua `LLMProviderPort` của authoring) gọi model thế nào sau khi chuyển?

A) Gọi llm-service giống các bước AI khác của visual-service (prompt nằm trong thư viện prompt của llm-service, có nhật ký cuộc gọi).
   - ✅ Một đường gọi AI duy nhất, có log, đổi model ở một chỗ.
   - ⚠️ Phải chuyển hai prompt `drawer_prompt_vi.txt`, `drawer_backdrop_prompt_vi.txt` (và prompt lập hình nếu giữ) sang thư viện prompt; nội dung prompt sẽ đưa Creator duyệt trước khi sửa.

B) Giữ nguyên cách gọi hiện tại, chỉ mang theo code.
   - ✅ Ít đổi.
   - ⚠️ Có thêm một đường gọi AI riêng ngoài llm-service.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Câu 7
Ảnh xem trước (PNG/GIF) đang lưu trong cột của bảng `illustrations`. Sau khi chuyển lưu ở đâu?

A) Giữ trong DB như hiện tại.
   - ✅ Sao lưu DB là có đủ ảnh; không phải đổi gì.
   - ⚠️ DB phình theo số hình.

B) Lưu file ở `/shared/illustrations/{id}.png|gif`, DB chỉ giữ đường dẫn — giống cách bước 11 lưu khung storyboard.
   - ✅ Thống nhất một cách lưu ảnh; sau này đồng bộ Google Drive cùng một chỗ.
   - ⚠️ Sao lưu phải gồm cả volume `/shared`; thêm bước chuyển ảnh ra file.

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

# Câu hỏi làm rõ (sau lượt trả lời đầu)

**Đã ghi nhận:** 1A viết lại bằng Python trong rendering · 2 xoá toàn bộ dữ liệu, vẽ lại từ đầu · 3A bỏ `project_illustrations` · 4 phong cách lấy từ bước 8 · 5A giữ URL · 6A gọi AI qua llm-service · 7B ảnh lưu file ở `/shared`.

## Câu 8
Câu 2 "xoá toàn bộ" là thao tác không hoàn tác được. Phạm vi xoá gồm những gì?

A) Xoá hết cả 6 bảng: 293 hình, 14 thư mục, phong cách hình (`visual_languages`, `visual_language_versions`), 4 khung mẫu, `project_illustrations`. Trước khi xoá, xuất một bản sao lưu JSON (dùng chức năng xuất đã có) cất ở `/shared/backup/` phòng khi cần xem lại.
   - ✅ Sạch hoàn toàn; vẫn còn đường lùi nếu đổi ý.
   - ⚠️ Bản sao lưu chỉ để xem/nhập tay, hệ thống mới không tự đọc nó.

B) Như A nhưng không sao lưu.
   - ✅ Gọn nhất.
   - ⚠️ Mất hẳn 293 hình.

C) Chỉ xoá hình và thư mục; giữ phong cách hình để chuyển sang rendering.
   - ✅ Giữ luật vẽ của kênh đang dùng.
   - ⚠️ Mâu thuẫn với câu 4 nếu phong cách chuyển hẳn sang bước 8.

X) Other (please describe after [Answer]: tag below)

[Answer]: A nhưng không sao lưu (B) — xoá, không cần hoàn tác

## Câu 9
Câu 4: bước 8 (phong cách tham chiếu ở visual-service) chỉ là **mô tả bằng chữ** (cách vẽ, màu, bố cục). Còn **kit** là bộ thành phần Remotion mà rendering dùng để vẽ thật (`kits.json`); lệnh render bắt buộc có `kit_key`, hiện orchestrator lấy nó từ phong cách hình mà dự án gắn ở authoring. Bỏ phong cách ở authoring thì `kit_key` lấy ở đâu?

A) Bước 8 thêm lựa chọn kit: khi chốt phong cách tham chiếu, Creator chọn kit (danh sách lấy từ rendering `GET /v1/kits`); orchestrator đọc `kit_key` từ visual-service.
   - ✅ Phong cách và kit nằm cùng một chỗ, đúng ý "phong cách nằm ở bước 8".
   - ⚠️ Sửa visual-service (bước 8) và orchestrator trong CR này.

B) Rendering tự chọn kit mặc định khi lệnh không có `kit_key` (hiện chỉ có một kit dùng thật).
   - ✅ Ít sửa nhất.
   - ⚠️ Khi có kit thứ hai thì phải làm lại việc chọn kit.

X) Other (please describe after [Answer]: tag below)

[Answer]: Chuyển thông tin kit `phatsang` vào phong cách tham chiếu R12 "Glowing Flat Vector" trong thư viện tham chiếu (visual-service). Dự án chọn Glowing Flat Vector ở bước 8 thì bước 11 (và render) đọc lại từ đó.

## Câu 10
Luật vẽ của kênh (hiện là các file `phatsang_*` gắn với phong cách hình: luật hình, luật phông nền, luật cảnh) mà bộ vẽ AI đọc khi vẽ. Sau khi chuyển, bộ vẽ AI lấy luật vẽ ở đâu?

A) Từ phong cách tham chiếu bước 8 + visual concept bước 7 của dự án; luật kỹ thuật cố định của kit (giới hạn thành phần, bộ xương) nằm trong prompt vẽ ở llm-service. Nội dung prompt đưa Creator duyệt trước khi sửa.
   - ✅ Không còn luật vẽ nằm rải ở authoring; hình theo đúng phong cách từng dự án.
   - ⚠️ Thư viện hình dùng chung (không gắn dự án) cần một phong cách mặc định để vẽ hình gốc.

B) Chuyển nguyên các file luật vẽ sang rendering làm luật của kit.
   - ✅ Giữ nguyên chất lượng vẽ hiện tại.
   - ⚠️ Hai nguồn luật (kit và bước 8) có thể mâu thuẫn.

X) Other (please describe after [Answer]: tag below)

[Answer]: Chỉ bước 8 (phong cách/ngôn ngữ hình); bỏ phần nhắc bước 7
