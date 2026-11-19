# CR-117 · Bước 11 Storyboard — yêu cầu

**Trạng thái:** đã duyệt 2026-11-20 (gồm hai đề xuất ở "Điểm cần Creator xác nhận") · **Nhánh:** `feature/cr-117-storyboard` · **Câu hỏi:** [cr-117-requirement-questions.md](cr-117-requirement-questions.md)

## Mục tiêu
Biến visual plan (bước 9) và visual sequence (bước 10) thành các **shot** cụ thể, mỗi shot có một khung hình thật (ảnh PNG). Kèm theo là **tài sản dự án** (`AST-…`): nhân vật, vật thể, phông nền được vẽ một lần theo đúng chủ đề của dự án rồi dùng lại ở nhiều shot. Output là `storyboard.json`.

## Phụ thuộc
- Không có CR hay nhánh nào đang mở, nên không chồng lấn.
- Cần CR-116 (đã vào `main`): thư viện hình ở rendering, kit nằm ở phong cách tham chiếu, `/v1/stills` nhận kit.

## Đã chốt
- Cấp bậc: Beat → Unit (bước 9) → Shot (bước 11). Một unit có 1 hoặc nhiều shot; mỗi shot có đúng một `unit_id` và một `beat_id`.
- Duyệt: duyệt từng shot; mọi shot của một unit đã duyệt thì unit tự được duyệt; bước 11 có trạng thái chung (`in_review`…).
- Lưu: bảng mới trong DB `rendering`; ảnh khung ở `/shared/{project_id}/storyboard/SH-xxx.png`, `frame_path` tương đối, không nhúng base64. Google Drive để sau.
- Tài sản hai tầng: hình gốc trong thư viện hình (dùng lại) và tài sản dự án `AST-…` (gắn `project_id` + `concept_id`).

## Yêu cầu chức năng

**FR-1 · Hai pha, tài sản trước, shot sau (câu 1A)**
1. **Pha tài sản:** AI lập danh sách tài sản dự án từ visual concept và visual plan → Creator duyệt danh sách → vẽ từng tài sản → Creator duyệt.
2. **Pha shot:** AI viết shot theo từng unit, chỉ dùng tài sản đã duyệt → vẽ khung → Creator duyệt shot.
Pha shot chỉ bắt đầu khi tài sản cần cho unit đó đã duyệt. Unit cần tài sản chưa có thì hiện "thiếu tài sản" và nêu mô tả cần vẽ; không tự bịa mã `AST-…`.

**FR-2 · Tài sản dự án (`AST-…`)**
Mỗi tài sản lưu: `asset_id` (dạng `AST-<TÊN>-<số>`), `project_id`, `concept_id` (nối về visual concept), `kind` (figure/backdrop), `tier` trong dự án (hero/support/background, quyết định độ chi tiết), `portrayal` (mô tả tự do cảm giác cần gợi ra), `must_not_imply`, `palette_tokens` (lấy từ bảng màu của visual concept), `expressions`/`poses`, `base_illustration_id` (hình gốc trong thư viện, có thể trống), ảnh đã vẽ, trạng thái duyệt. Không có danh sách vai trò cố định; `portrayal` là văn bản tự do lấy từ visual concept.

**FR-3 · Tìm hình gốc (câu 6A)**
Khi lập danh sách tài sản, AI so mô tả từng tài sản với tên, tag, mô tả và `kind` của hình trong thư viện và đề xuất hình gốc. Creator chọn dùng hình gốc nào hoặc "vẽ mới".

**FR-4 · Đưa tài sản vào thư viện (câu 7, theo lời Creator)**
Mặc định **tick sẵn** ô "Đưa vào thư viện": tài sản đã duyệt được đưa vào thư viện hình. Creator có thể **bỏ tick** để không đưa. *(Xem "Điểm cần Creator xác nhận" bên dưới.)*

**FR-5 · Viết shot, mỗi unit một lượt AI (câu 3A)**
Mỗi unit một lượt AI: nhận unit (bước 9), mục sequence tương ứng (bước 10), luật bố cục và chữ của visual concept (bước 8), các tài sản đã duyệt, và các shot của unit trước (để giữ nhất quán). Trả 1 hoặc nhiều shot theo đúng cấu trúc mẫu: `shot_id`, `unit_id`, `beat_id`, `script_refs`, `shot_purpose`, `composition` (chủ thể, vị trí, vật chính, góc máy), `action_summary`, `on_screen_text`, `asset_refs`, `notes`. Chạy lại được từng unit. Số từ chữ trên màn hình không vượt luật typography của visual concept.

**FR-6 · Dựng khung bằng composition chung (câu 2A)**
Rendering có một composition Remotion "khung storyboard": đặt các tài sản đã vẽ theo `composition` của shot (vị trí, góc máy), nền theo màu journey của beat, chữ trên màn hình; dùng kit của phong cách dự án chọn ở bước 8. Không gọi AI để vẽ khung. Sửa bố cục thì vẽ lại ngay. Ảnh lưu ở `/shared/{project_id}/storyboard/SH-xxx.png`.

**FR-7 · Duyệt ba cấp**
- Shot: Creator duyệt từng shot (sửa tay các trường, vẽ lại khung, duyệt).
- Unit: tự duyệt khi mọi shot của unit đã duyệt; sửa lại một shot thì unit trở về chưa duyệt.
- Bước 11: có trạng thái chung theo mẫu (`in_review`, …); duyệt khi mọi unit đã duyệt.

**FR-8 · AI rà soát (câu 4A)**
Creator bấm "Rà soát": một lượt AI duyệt toàn bộ shot (đủ ý của plan, khớp `must_not_imply`, chữ trên màn hình đọc được, tài sản dùng nhất quán) và trả các ghi chú để Creator sửa. Kiểm tra bằng code (shot có unit/beat hợp lệ, `asset_refs` tồn tại và đã duyệt, chữ không quá số từ cho phép, ảnh khung tồn tại) luôn chạy, không cần bấm.

**FR-9 · Chốt phiên bản và xuất (câu 5A)**
Khi bước đã duyệt, Creator bấm "Chốt": tạo phiên bản bất biến (`storyboard-v1`, `-v2`…), ghi `/shared/{project_id}/storyboard/storyboard.json` cùng ảnh khung của phiên bản đó, và có route nội bộ để các bước sau đọc đúng bản đã chốt. `storyboard.json` theo mẫu: `schema_version`, `project_id`, `document_id`, `status`, `shots[]`.

**FR-10 · Giao diện (câu 8A)**
Một trang hai tab: **Tài sản** và **Shot**. Tab Shot nhóm theo beat → unit; mỗi shot là một thẻ gồm ảnh khung, mục đích, bố cục, hành động, chữ trên màn hình, tài sản, ghi chú, và nút duyệt / sửa / vẽ lại. Làm mockup và được Creator duyệt **trước khi code** giao diện, theo quy tắc của dự án.

## Yêu cầu phi chức năng
- Theo `docs/code-standards-rules.md` và `docs/ux-ui-design-rules.md`; test giả lời AI, không gọi model thật.
- Prompt AI mới (lập tài sản, viết shot, rà soát) đưa Creator duyệt nguyên văn trước khi đưa vào code.
- Cập nhật hợp đồng trong `docs/contracts/` và một ADR cho cách chia việc giữa visual-service và rendering.
- Dự án chưa chọn phong cách ở bước 8, hoặc phong cách chưa có kit: bước 11 báo lỗi rõ, không vẽ khung.

## Điểm cần Creator xác nhận
1. **Câu 7, ô "Đưa vào thư viện" tick sẵn:** tài sản dự án mang màu, biểu cảm và vai trò riêng của chủ đề (ví dụ virus "xấu" ở dự án B). Nếu đưa nguyên bản đó vào thư viện, dự án sau có thể lấy nhầm bản mang vai trò cũ. Mình đề xuất khi tick thì thư viện nhận bản đã vẽ, **ghi rõ tên dự án trong mô tả hình**, đặt vào một thư mục mặc định "Từ dự án" (hoặc Creator chọn thư mục ngay ở ô đó). Bạn đồng ý, hay muốn cách khác?
2. **Ai điều phối bước 11:** ADR-0049 giao visual-service sở hữu các bước tiền kỳ hình ảnh (AI, đơn vị duyệt, chốt phiên bản), còn bạn muốn dữ liệu output nằm ở DB rendering. Mình đề xuất: **visual-service điều phối** (gọi AI, quản lý duyệt, chốt), và gọi rendering để vẽ khung và **lưu bảng storyboard, shot, tài sản**. Chi tiết chốt ở bước thiết kế. Bạn có muốn phần điều phối nằm hẳn ở rendering không?

## Ngoài phạm vi
- Đồng bộ Google Drive.
- Bước 12 trở đi (chuyển động, animatic).
- Bỏ hẳn bước "Hình minh hoạ" cũ khỏi luồng của orchestrator (CR dọn luồng riêng).
