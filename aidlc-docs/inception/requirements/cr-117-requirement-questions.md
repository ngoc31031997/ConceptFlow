# CR-117 · Bước 11 Storyboard — câu hỏi yêu cầu

## Đã chốt (2026-11-19/20, trước khi mở CR)
- **Cấp bậc:** Beat → Unit (bước 9) → Shot (bước 11). Một unit có 1 hoặc nhiều shot; mỗi shot có đúng một `unit_id` và một `beat_id`.
- **Duyệt:** duyệt từng shot; mọi shot của unit đã duyệt → unit tự duyệt; bước 11 có trạng thái chung (`in_review`…) như mẫu `storyboard.json`.
- **Lưu:** bảng mới trong DB `rendering`; ảnh khung `/shared/{project_id}/storyboard/SH-xxx.png`, `frame_path` tương đối, không base64. Google Drive làm sau.
- **Vẽ khung:** rendering (`/v1/stills`) với kit của phong cách dự án chọn ở bước 8 (R12 "Glowing Flat Vector").
- **Tài sản hai tầng, tạo ngay trong bước 11:** hình gốc trong thư viện hình (dùng lại) + tài sản dự án `AST-…` gắn `project_id` + `concept_id`, có `portrayal` (mô tả tự do), `must_not_imply`, `palette_tokens`, `expressions`, tier trong dự án (hero/support/background), hình đã vẽ, trạng thái duyệt. `asset_refs` của shot trỏ vào `AST-…`. Thiếu tài sản thì đánh dấu thiếu, không bịa mã.
- **Input:** bản chốt bước 8 (visual concept), 9 (visual plan), 10 (visual sequence) đọc qua route nội bộ visual-service (`/internal/v1/visual/{id}/concept|plan|sequence`); kịch bản chốt qua authoring.
- **Mockup:** màn hình bước 11 phải có mockup được duyệt trước khi code.

Trả lời bằng cách điền sau `[Answer]:`.

---

## Câu 1
Thứ tự công việc bên trong bước 11?

A) Tài sản trước, shot sau: (1) AI lập danh sách tài sản dự án từ visual concept + plan → Creator duyệt danh sách → vẽ từng tài sản (dùng hình gốc nếu có) → duyệt; (2) AI viết shot theo từng unit, chỉ dùng tài sản đã duyệt → vẽ khung → duyệt shot.
   - ✅ Khung vẽ bằng hình thật đã duyệt, không phải vẽ lại khi tài sản đổi.
   - ⚠️ Chưa thấy shot đã phải chốt tài sản; dài hơn.

B) Shot trước, tài sản sau: AI viết shot kèm đề xuất tài sản; khung vẽ phác (hình khối thay thế) → duyệt shot → rồi mới vẽ tài sản và vẽ lại khung.
   - ✅ Thấy nhịp shot sớm.
   - ⚠️ Mỗi khung vẽ hai lần; tài sản có thể lệch với shot đã duyệt.

X) Other (please describe after [Answer]: tag below)

[Answer]: a

## Câu 2
Khung storyboard được dựng thế nào?

A) Một composition Remotion chung "khung storyboard": đặt các tài sản đã vẽ theo `composition` của shot (vị trí chủ thể/vật chính, góc máy wide/medium/close/split…), nền theo màu journey của beat, chữ trên màn hình. Không gọi AI để vẽ khung.
   - ✅ Nhanh, rẻ, ổn định, sửa vị trí là vẽ lại ngay; đúng kit.
   - ⚠️ Bố cục theo bộ luật cố định (ô lưới), không tự do như vẽ tay.

B) AI viết code Remotion riêng cho từng khung (như bộ vẽ hình), kiểm tra bằng `/v1/check/remotion`.
   - ✅ Bố cục tự do, gần khung phim thật hơn.
   - ⚠️ Mỗi shot một lượt AI (có khi 3 lượt sửa lỗi); chậm, tốn, kết quả dao động.

C) 💡 Gợi ý: A làm mặc định, cho phép bấm "Vẽ khung bằng AI" (B) cho shot nào cần.
   - ✅ Rẻ cho đa số, tự do khi cần.
   - ⚠️ Hai đường vẽ phải bảo trì.

X) Other (please describe after [Answer]: tag below)

[Answer]: a

## Câu 3
AI viết shot theo đơn vị nào?

A) Mỗi unit một lượt AI (giống bước 10): nhận unit + beat + sequence item + tài sản đã duyệt → trả 1..n shot của unit.
   - ✅ Lỗi một unit không làm hỏng cả bước; chạy lại từng unit.
   - ⚠️ Nhiều lượt gọi; cần giữ nhất quán giữa các unit (truyền shot của unit trước).

B) Mỗi beat một lượt AI.
   - ✅ Nhất quán trong beat, ít lượt hơn.
   - ⚠️ Beat dài → trả lời dài, dễ hỏng.

X) Other (please describe after [Answer]: tag below)

[Answer]: a

## Câu 4
Có AI rà soát (reviewer) cả storyboard như bước 8–10 không?

A) Có: một lượt rà soát toàn bộ shot (đủ ý của plan, khớp `must_not_imply`, chữ trên màn hình đọc được, tài sản dùng nhất quán), trả các ghi chú để Creator sửa.
   - ✅ Cùng quy trình với các bước trước.
   - ⚠️ Thêm một prompt và một lượt AI.

B) Không, chỉ kiểm tra bằng code (shot có unit/beat hợp lệ, `asset_refs` tồn tại và đã duyệt, chữ ≤ số từ tối đa của typography, ảnh khung tồn tại).
   - ✅ Đơn giản, không tốn AI.
   - ⚠️ Không bắt được lỗi về nghĩa.

C) Cả A và B (kiểm tra code luôn chạy; AI rà soát khi Creator bấm).

X) Other (please describe after [Answer]: tag below)

[Answer]: a

## Câu 5
Chốt phiên bản và xuất `storyboard.json`?

A) Giống bước 8–10: khi mọi unit đã duyệt, Creator bấm "Chốt" → tạo phiên bản bất biến (`storyboard-v1`, `-v2`…), ghi `/shared/{project_id}/storyboard/storyboard.json` + ảnh khung của phiên bản đó, và route nội bộ cho bước sau đọc.
   - ✅ Bước 12+ đọc đúng bản đã chốt; có lịch sử.
   - ⚠️ Mỗi phiên bản giữ một bộ ảnh (tốn chỗ).

B) Chỉ một bản sống, ghi đè mỗi lần chốt.
   - ✅ Gọn.
   - ⚠️ Không lùi lại được; bước sau có thể đọc bản đang sửa.

X) Other (please describe after [Answer]: tag below)

[Answer]: a

## Câu 6
Tìm hình gốc trong thư viện cho tài sản dự án thế nào?

A) AI đề xuất (so mô tả tài sản với tên/tag/mô tả hình trong thư viện, cùng `kind`) → Creator chọn dùng hình gốc nào hoặc "vẽ mới".
   - ✅ Creator quyết; tận dụng thư viện.
   - ⚠️ Thêm một lượt AI khi lập danh sách tài sản.

B) Creator tự tìm và chọn trong thư viện, AI không đề xuất.
   - ✅ Không tốn AI.
   - ⚠️ Chậm khi thư viện lớn.

X) Other (please describe after [Answer]: tag below)

[Answer]: a

## Câu 7
Tài sản vẽ mới cho dự án có tự đưa bản trung tính vào thư viện để dự án sau dùng lại không?

A) Không tự đưa; trên thẻ tài sản có nút "Đưa vào thư viện" (Creator chọn thư mục).
   - ✅ Thư viện chỉ chứa thứ Creator muốn giữ.
   - ⚠️ Phải nhớ bấm.

B) Tự đưa mọi tài sản đã duyệt vào thư viện.
   - ✅ Thư viện lớn dần.
   - ⚠️ Lẫn hình mang màu/vai trò riêng của dự án.

X) Other (please describe after [Answer]: tag below)

[Answer]: default là đưa vào. user cho thể chọn bỏ bằng cách untick checkbox

## Câu 8
Giao diện bước 11 đặt thế nào? (sẽ làm mockup sau khi trả lời)

A) Một trang với hai tab: "Tài sản" và "Shot". Tab Shot nhóm theo beat → unit, mỗi shot là một thẻ: ảnh khung, mục đích, bố cục, hành động, chữ trên màn hình, tài sản, ghi chú, nút duyệt/sửa/vẽ lại.
   - ✅ Giống bố cục các bước 9–10.
   - ⚠️ Trang dài khi nhiều shot.

B) Dạng lưới khung (contact sheet) xem cả phim một lượt, bấm vào khung mở bảng sửa bên cạnh.
   - ✅ Nhìn nhịp cả phim; đúng tinh thần storyboard.
   - ⚠️ Khác các bước trước.

C) 💡 Gợi ý: A làm chính, thêm nút "Xem dạng lưới" (B, chỉ xem).

X) Other (please describe after [Answer]: tag below)

[Answer]: A
