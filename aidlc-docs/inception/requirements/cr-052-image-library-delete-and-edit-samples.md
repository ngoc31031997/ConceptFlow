# CR-052 — Thư viện hình: nút Xoá trên từng hình, cho phép thay đổi Hình mẫu

## Date
2026-09-29

## Stage
Requirements Analysis — mọi câu hỏi đã trả lời; **chờ Creator duyệt bản yêu cầu**. Chưa thiết kế, chưa viết code.

## Yêu cầu gốc (nguyên văn)
> "ở màn hình thư viện hình ko có button xoá hình, ở mục hình mẫu có thể cho phép thay đổi không"

Trả lời lượt 1 của Creator (nguyên văn):
> "1 chỉ không được xoá khi hình được dùng trong dự án đang trong quá trình render hoặc đó là hình minh hoạ (bước 5-11)
> 2 hiện tại ko có phân quyền trên hệ thống
> hình minh hoạ để promt tham chiếu đến sửa url tham chiếu là được mà đúng ko"

Trả lời lượt 2 của Creator (nguyên văn):
> Q1 "đúng". Q2 "tối đa 5 hình nhưng các hình mẫu sẽ được clone vào 1 folder và dc ref đến qua url hoặc 1 cách nào đó chứ ko viết cứng trong txt nữa". Q3 "chỉ có hình mẫu là ko xoá dc". Q4 "dúngd" (đúng).

Trả lời lượt 3 của Creator (nguyên văn):
> "F1 : không dc xoá / F2 làm theo đề xuất, hình mâux là bản clone / f3 đi theo B"

## Phân tích ý định
- **Loại**: Enhancement (bổ sung cho CR-044 thư viện hình và CR-049 luật style).
- **Phạm vi**: nhiều thành phần — `web-gui` (IllustrationLibraryPage, IllustrationTile) và `authoring-service` (use case xoá hình, thư mục Hình mẫu, prompt hoạ sĩ AI, seeder). Nếu F1 = C thì thêm `llm-service`.
- **Độ phức tạp**: Trung bình.

## Hiện trạng (đã đọc code)
### Xoá hình
- Đã có API `DELETE /v1/admin/illustrations/{id}` (`application/illustrations.go` `Delete`). API này chỉ chặn hình `builtin` (bộ kit và Hình mẫu).
- Trên màn hình, nút "Xoá" chỉ nằm **bên trong trình sửa code** (`IllustrationEditor`). Ô hình trong lưới không có nút Xoá.
- Xoá hiện **không kiểm tra hình có đang được dự án dùng hay không**. `project_illustrations.illustration_id` là `ON DELETE SET NULL`, nên nếu xoá giữa chừng thì dự án mất liên kết, và code Remotion đang gọi component đó sẽ lỗi khi render.

### Hình mẫu
- Ba Hình mẫu (SchoolBus, Cat, Microscope) là **code TSX** viết cứng trong `authoring-service/internal/domain/prompts/illustration_exemplars_vi.txt`, không phải URL, và được đóng gói vào image khi build.
- Prompt hoạ sĩ (`illustration_drawer.go` `references`) dán nguyên văn code của 3 Hình mẫu, cộng tối đa 2 hình Creator đã duyệt cùng thư mục. Model đọc code, không xem ảnh.
- Khi khởi động, seeder ghi đè các hàng `exemplar-*` từ file này.

### Bộ kit
- 34 hình kit (Tooth, Toothbrush, Airplane…) nằm trong image render (`conceptflow-mini/illustration.tsx`). Mọi video **luôn import toàn bộ kit** (`llm-service/app/pipeline/merger.py` `ILLUSTRATION_KIT`), và prompt Kỹ sư Remotion nói với AI rằng kit có sẵn. Các hàng kit trong bảng `illustrations` chỉ để hiện trong Thư viện và cho bước 5 chọn "dùng lại". Seeder thêm lại các hàng này mỗi lần khởi động.

### Phân quyền
- Chưa có phân quyền. Mọi thao tác trong CR này ai dùng cũng làm được.

### Bước wizard
Bước 5 Hình minh hoạ → 6 Kỹ sư (code) → 7–11 kiểm tra / duyệt / xử lý / render → 12 Kết quả → 13 Đăng.

## Quyết định đã chốt
- **Q1 — Khi nào chặn xoá**: hình "đang dùng" khi nó nằm trong `project_illustrations` của một dự án, HOẶC code Remotion của dự án gọi tới tên component của hình, **và** dự án đó chưa tới bước 12 (Kết quả). Dự án đã xong thì không chặn.
- **Q2 — Hình mẫu**: tối đa **5**. Hình mẫu là **bản clone** của một hình đã duyệt, nằm trong thư mục hệ thống "Hình mẫu". Prompt hoạ sĩ lấy Hình mẫu từ DB, tham chiếu theo thư mục / ID, không dùng URL vì model cần code chứ không cần ảnh. Bỏ file `illustration_exemplars_vi.txt`. 3 Hình mẫu hiện có được chuyển vào thư mục này một lần làm dữ liệu ban đầu, và seeder không ghi đè chúng nữa.
- **Q3 + F1 — Hình không xoá được**: Hình mẫu và bộ kit (34 hình trong image render). Mọi hình khác xoá được theo luật Q1.
- **Q4 — Khi bị chặn xoá**: nút Xoá vẫn bấm được, bấm thì hiện "Đang được dùng trong dự án …".
- **F2 — Bỏ làm mẫu**: Hình mẫu có nút "Bỏ làm mẫu", tách riêng với "Xoá". Với bản clone: bỏ bản clone, hình gốc vẫn ở thư mục cũ. Với 3 Hình mẫu gốc (không có bản gốc nào khác): chuyển về thư mục chủ đề (Phương tiện / Động vật / Khoa học) thành hình thường đã duyệt, không mất.
- **F3 — Hình mẫu chỉ đọc**: không sửa code / Vẽ lại trên Hình mẫu. Muốn đổi thì bỏ mẫu cũ, rồi đặt mẫu mới từ hình đã sửa.
- **Phân quyền**: chưa có trong hệ thống, nên mọi thao tác ai dùng cũng làm được.

## Yêu cầu chức năng
- **FR1 — Nút Xoá trên ô hình**: trong lưới Thư viện, mỗi ô hình (trừ Hình mẫu và kit) có nút "Xoá", kèm hộp xác nhận. Nút "Xoá" trong trình sửa vẫn giữ và theo cùng luật.
- **FR2 — Chặn xoá khi đang dùng**: `DELETE /v1/admin/illustrations/{id}` trả **409**, nêu tên (các) dự án đang dùng, khi hình vi phạm luật Q1. Hình mẫu và kit trả lỗi "chỉ đọc" như hiện nay. Luật nằm ở backend, nên mọi đường xoá đều bị chặn, kể cả nút "xoá hình" ở bước 5 của dự án (`project_illustrations.go`).
- **FR3 — Thư mục Hình mẫu**: có thư mục hệ thống "Hình mẫu". Thư mục này không xoá được và không thêm hình vào được bằng cách thường (tạo mới, tải SVG, AI vẽ, đổi thư mục).
- **FR4 — Đặt làm mẫu**: hình **đã duyệt**, không phải kit, không phải Hình mẫu, có nút "Đặt làm mẫu". Bấm thì clone code, tiêu đề, mô tả, thẻ và cách dùng sang một hàng mới trong thư mục Hình mẫu. Bản clone phải mang tên component riêng, vì tên hình là duy nhất (`illustrations_name_key`); cách đặt tên quyết định ở bước thiết kế. Đủ 5 thì từ chối kèm thông báo. Không clone một hình hai lần.
- **FR5 — Bỏ làm mẫu**: theo F2. Nếu bản clone (hoặc Hình mẫu gốc) đang được dự án dùng theo luật Q1, thì cũng chặn bỏ làm mẫu, với cùng thông báo như FR2. Hình mẫu gốc được chuyển thư mục, nên không bị chặn.
- **FR6 — Hình mẫu chỉ đọc**: Hình mẫu không sửa, không Vẽ lại, không Bỏ duyệt, không đổi thư mục được. Vẫn "Dựng lại" (ảnh xem trước) được.
- **FR7 — Prompt AI vẽ đọc Hình mẫu từ DB**: `references` (`illustration_drawer.go`) lấy các hình trong thư mục Hình mẫu (tối đa 5), rồi thêm tối đa 2 hình Creator đã duyệt cùng thư mục, tức tối đa 7 hình. `/v1/illustration-style` trả `exemplar_ids` từ DB. Mục "Luật style của kênh" trên màn hình hiện các Hình mẫu đang có, kèm nút "Bỏ làm mẫu".
- **FR8 — Chuyển dữ liệu**: 3 Hình mẫu gốc giữ nguyên ID và tên (`exemplar-SchoolBus`, `exemplar-Cat`, `exemplar-Microscope`), chuyển vào thư mục Hình mẫu **một lần**. Sau đó seeder không tạo lại và không ghi đè chúng, kể cả khi đã bị "Bỏ làm mẫu". Bỏ `illustration_exemplars_vi.txt` và `ExemplarIllustrations()`.
- **FR9 — Sao lưu / khôi phục** (`illustration_backup.go`): bản sao lưu phải giữ được Hình mẫu (thư mục và trạng thái mẫu). Khi khôi phục, không vượt quá 5. Chi tiết quyết định ở bước thiết kế.
- **FR10**: Hình AI vẽ sau khi đổi mẫu sẽ theo mẫu mới. Hình đã vẽ trước đó không tự đổi.

## Yêu cầu phi chức năng
- Tuân theo `docs/ux-ui-design-rules.md`: Neubrutalism, thông báo trượt ra / trượt vào theo lớp `.reveal` dùng chung.
- Kiểm tra "đang dùng" chạy ở backend trong cùng giao dịch với lệnh xoá, để tránh lỗi đua giữa lúc kiểm tra và lúc xoá.
- Không đổi contract giữa các service. `llm-service` và `rendering` không đổi.
- Test: use case xoá / đặt mẫu / bỏ mẫu (Go), repository với Postgres, prompt hoạ sĩ (golden test có thể phải cập nhật), và component web-gui (Vitest).
- Rebuild `authoring-service` và `web-gui` sau khi code xong (CLAUDE.md).

## Tiêu chí chấp nhận
1. Ô hình thường có nút Xoá. Xoá được hình không ai dùng.
2. Xoá một hình mà dự án ở bước 5–11 đang dùng: hiện "Đang được dùng trong dự án …", hình vẫn còn.
3. Xoá hình mà chỉ dự án đã xong (bước 12–13) dùng: xoá được.
4. Hình mẫu và kit không có nút Xoá. Gọi API xoá thẳng thì bị từ chối.
5. "Đặt làm mẫu" tạo bản clone trong thư mục Hình mẫu. Làm lần thứ 6 thì bị từ chối kèm thông báo.
6. "Bỏ làm mẫu" bản clone: bản clone biến mất, hình gốc còn. "Bỏ làm mẫu" Hình mẫu gốc: hình chuyển về thư mục chủ đề. Khởi động lại authoring-service thì hình đó không quay lại thư mục Hình mẫu.
7. Hình mẫu không có nút Sửa code / Vẽ lại.
8. Prompt AI vẽ chứa code của đúng các Hình mẫu đang có, không còn đọc file txt.

## Ngoài phạm vi
- Xoá hoặc ẩn hình kit (F1 = không).
- Gửi ảnh cho model (tham chiếu bằng ảnh / URL). Model vẫn chỉ đọc code.
- Phân quyền.

## Extensions
Security Baseline và Property-Based Testing đã quyết định **No** ở cấp dự án (`aidlc-state.md`), nên không hỏi lại.
