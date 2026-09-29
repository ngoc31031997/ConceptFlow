# CR-052 — Thiết kế: nút Xoá trên ô hình, thư mục Hình mẫu

Trạng thái: **DESIGN — chờ Creator duyệt**. Yêu cầu đã duyệt: `aidlc-docs/inception/requirements/cr-052-image-library-delete-and-edit-samples.md` (FR1–FR10).

Service bị ảnh hưởng: `authoring-service`, `api-gateway` (2 route mới), `web-gui`. `llm-service`, `rendering`, `orchestrator` không đổi.

## 1. Kiểm tra "đang dùng" (FR2)

Mỗi service có DB riêng. Trạng thái dự án (`status`) nằm ở **orchestrator**. Liên kết hình ↔ dự án và code dự án nằm ở **authoring**. Thiết kế:

1. **Tìm dự án ứng viên trong DB authoring** (1 truy vấn):
   - `project_illustrations.illustration_id = <id>`, hoặc
   - `project_authoring.code_content` gọi tới tên component, so khớp nguyên từ bằng regex Postgres: `code_content ~ ('\m' || <name> || '\M')`. Tên hình là PascalCase `\w+`, không có ký tự đặc biệt regex.
   - Trả về `project_id` và `topic`, để hiện tên dự án trong thông báo.
2. **Hỏi trạng thái từng ứng viên** qua client có sẵn `orchestrator.Client.GetStatus` (`GET /internal/v1/projects/{id}/status`).
   - `ready_to_publish`, `publishing`, `published` (bước 12–13): **không chặn**.
   - Mọi trạng thái khác, kể cả `draft` và `failed_at_*` (dự án còn có thể chạy lại): **chặn**.
   - Orchestrator trả 404 (dự án đã bị xoá, còn sót hàng authoring): **không chặn**.
   - Orchestrator lỗi hoặc không trả lời: **từ chối xoá** (503, "Không kiểm tra được hình có đang được dùng không, thử lại sau"). Không đoán, không bỏ qua.
3. Có dự án chặn: trả `ErrIllustrationInUse{Projects: [{ID, Topic}]}`, HTTP **409** `{"error":"illustration_in_use","message":"Đang được dùng trong dự án: …","projects":[…]}`.

**Cổng mới** trong application: `IllustrationUsagePort` (`FindUsers(ctx, id, name) ([]ProjectRef, error)`) do repo Postgres cài, và `ProjectStatusPort` (`GetStatus`) do orchestrator client cài. Hàm "đang dùng" thuần (`blocking(statuses)`) nằm ở domain để test riêng.

**Đua dữ liệu**: không có giao dịch xuyên service. Lỗi đua còn lại là một dự án liên kết hình này **đúng** trong khoảng vài ms giữa bước kiểm tra và bước xoá. Để thu hẹp khoảng đó: xoá bằng `DELETE … WHERE id=$1 AND NOT EXISTS (SELECT 1 FROM project_illustrations WHERE illustration_id=$1 AND project_id <> ALL($ignored))`, gộp kiểm tra liên kết và xoá trong một câu SQL. Nếu 0 hàng bị xoá thì chạy lại bước 1–3. Phần code-reference không khoá được, nhưng chỉ xảy ra khi bước code chạy đúng lúc đó; chấp nhận và ghi lại ở đây.

**Nút "xoá hình" ở bước 5** (`ProjectIllustrationsUseCase` → `library.Delete`): chỉ xoá bản nháp vừa vẽ cho chính dự án đó. Nên hàm xoá nhận thêm `ignoreProject`, để dự án đang thao tác không tự chặn chính nó. Các dự án khác vẫn chặn theo luật.

## 2. Mô hình Hình mẫu (FR3–FR8)

### Dữ liệu
| Cột / đối tượng | Mới? | Ý nghĩa |
|---|---|---|
| thư mục `hinh-mau` "Hình mẫu" | mới, `is_system` | thêm vào `SystemIllustrationFolders()`, đứng đầu danh sách |
| `illustrations.exemplar` | có sẵn | `true` = đang là Hình mẫu (luôn nằm trong `hinh-mau`) |
| `illustrations.source_id TEXT NULL` | mới, `ON DELETE SET NULL` | bản clone: ID hình gốc (chống clone 2 lần; gốc bị xoá thì bản clone vẫn còn) |
| `illustrations.home_folder_id TEXT NULL` | mới | chỉ cho 3 Hình mẫu gốc: thư mục chủ đề để trả về khi "Bỏ làm mẫu" |

`builtin` từ nay **chỉ còn nghĩa là kit** (hình có trong image render). Hình mẫu đọc từ DB, nên `builtin=false, exemplar=true`. Luật chỉ đọc là `builtin || exemplar` (hàm `ReadOnly()` trên domain), thay cho các chỗ đang kiểm `Builtin` để chặn sửa, xoá, đổi trạng thái hay vẽ lại.

### Chuyển dữ liệu (FR8), chạy trong `db.go` khi khởi động, idempotent
```sql
ALTER TABLE illustrations ADD COLUMN IF NOT EXISTS source_id TEXT REFERENCES illustrations(id) ON DELETE SET NULL;
ALTER TABLE illustrations ADD COLUMN IF NOT EXISTS home_folder_id TEXT;
```
Sau khi seeder tạo thư mục `hinh-mau`, repo chạy bước chuyển một lần:
```sql
UPDATE illustrations SET home_folder_id = folder_id, folder_id = 'hinh-mau', builtin = false
WHERE exemplar AND builtin;   -- chỉ khớp hàng seed cũ; lần sau builtin=false nên không chạy lại
```
Seeder **không còn** chèn hay ghi đè Hình mẫu. `ExemplarIllustrations()` và `illustration_exemplars_vi.txt` bị xoá. Hình mẫu gốc đã "Bỏ làm mẫu" sẽ không quay lại khi khởi động lại (tiêu chí 6).

**DB mới hoàn toàn** (không có hàng seed cũ): thư mục Hình mẫu bắt đầu **rỗng**, và AI vẽ chỉ theo luật style cộng hình đã duyệt cùng thư mục, cho tới khi Creator "Đặt làm mẫu". Chỉ có một môi trường đang chạy, đã có 3 hàng seed, nên dữ liệu thật không bị mất. *(Phương án khác: giữ file txt làm seed ban đầu cho DB mới. Mình không chọn vì Creator yêu cầu bỏ file cứng; cho mình biết nếu muốn giữ.)*

### Đặt làm mẫu (FR4): `POST /v1/admin/illustrations/{id}/exemplar`
Điều kiện: hình `approved`, không phải kit, không phải Hình mẫu, chưa có bản clone (`source_id`); tổng Hình mẫu < 5. Vi phạm thì trả 409 hoặc 422 với thông báo tiếng Việt.
- **Tên bản clone**: `<Tên>Mau`; nếu trùng thì `<Tên>Mau2`, `<Tên>Mau3`… Code được đổi tên **chỉ ở dòng `export function <Tên>`**, dùng `exportNameRe` sẵn có. Tên component phải khớp tên hàng, như mọi hình trong thư viện.
- Chạy `uc.render` với code đã đổi tên (phải qua được renderer như mọi hình) và lưu ảnh xem trước. Bản clone: `folder_id='hinh-mau', exemplar=true, status=approved, source_id=<gốc>`; tiêu đề, mô tả, thẻ, cách dùng chép từ gốc.
- Giới hạn 5 kiểm trong giao dịch: `SELECT count(*) … FOR UPDATE` trên hàng thư mục `hinh-mau`, để 2 lần bấm cùng lúc không thành 6 hình.

### Bỏ làm mẫu (FR5): `DELETE /v1/admin/illustrations/{id}/exemplar`
- Bản clone (`source_id` có hoặc đã NULL vì gốc bị xoá, `home_folder_id` NULL): **xoá hàng**, dùng cùng luật "đang dùng" ở mục 1.
- Hình mẫu gốc (`home_folder_id` có): chuyển về `home_folder_id`, `exemplar=false`, `home_folder_id=NULL`, thành hình thường đã duyệt, sửa được. Không cần kiểm "đang dùng" vì tên và ID giữ nguyên.

### Bản clone không vào danh sách dùng cho video
Bản clone là bản sao để dạy AI vẽ, hình gốc vẫn nằm trong thư viện. Nếu cho vào danh sách chọn ở bước 5 (`project_illustrations.go` catalog) và danh sách của Kỹ sư Remotion (`ForCode`), AI sẽ thấy 2 hình giống hệt nhau với 2 tên. Vì vậy **loại hàng có `source_id` hoặc (`exemplar` và không có `home_folder_id`)** khỏi 2 danh sách đó. 3 Hình mẫu gốc vẫn dùng được cho video như hiện nay. Hệ quả: bản clone gần như không bao giờ "đang dùng", nên "Bỏ làm mẫu" bản clone hầu như không bị chặn.

### Prompt AI vẽ (FR7)
`references()` lấy Hình mẫu từ repo (`ListIllustrations{FolderID:"hinh-mau"}`, theo `created_at`, tối đa 5), rồi thêm tối đa 2 hình Creator đã duyệt cùng thư mục (không phải Hình mẫu). `maxDrawReferences` 5 → 7. Không Hình mẫu nào thì phần tham chiếu chỉ còn hình cùng thư mục, có thể rỗng (prompt vẫn hợp lệ). `/v1/illustration-style` trả `exemplar_ids` từ DB.

Test golden của prompt hoạ sĩ (`illustration_drawer_test.go`) được cập nhật theo nguồn mới.

### Chặn đưa hình vào `hinh-mau` bằng đường thường (FR3)
Tạo, sửa, AI vẽ, tải SVG, đổi thư mục hay khôi phục sao lưu vào `folder_id='hinh-mau'`: trả 422 "Thư mục Hình mẫu chỉ nhận hình qua 'Đặt làm mẫu'". Xoá thư mục hệ thống vốn đã bị chặn.

### Sao lưu / khôi phục (FR9)
- **Xuất**: hàng không phải kit đều xuất, gồm cả Hình mẫu, kèm `exemplar`, `source_name` (tên gốc, nếu có) và `home_folder_id`.
- **Nhập** Hình mẫu: nếu còn chỗ (< 5) thì nhập làm Hình mẫu, nối lại `source_id` theo `source_name` nếu gốc có trong thư viện. Hết chỗ thì nhập thành hình thường đã duyệt vào `home_folder_id` hoặc thư mục của gốc; nếu không xác định được thì bỏ qua. Báo cáo nhập ghi rõ lý do từng dòng. Luật trùng tên giữ như cũ.

## 3. API

| Method | Path | Kết quả |
|---|---|---|
| DELETE | `/v1/admin/illustrations/{id}` | 204; 403 chỉ đọc (kit / Hình mẫu); **409 `illustration_in_use`**; 503 không kiểm tra được |
| POST | `/v1/admin/illustrations/{id}/exemplar` | **mới**: 201 + bản clone; 409 `exemplar_limit` / `already_exemplar`; 422 chưa duyệt / kit |
| DELETE | `/v1/admin/illustrations/{id}/exemplar` | **mới**: 200 + hình sau khi chuyển (gốc) hoặc 204 (clone đã xoá); 409 `illustration_in_use` |
| GET | `/v1/illustration-style` | `exemplar_ids` lấy từ DB |

Kiểu `Illustration` (Go + TS) thêm `source_id?: string` và `home_folder_id?: string`. `api-gateway/src/routes/illustrations.js` thêm 2 route `fast`, đặt **trước** `/:id` DELETE để không bị route đó bắt nhầm.

## 4. Giao diện (web-gui)

Theo `docs/ux-ui-design-rules.md` (Neubrutalism, `Button` variant sẵn có, thông báo dùng `.reveal`).
- **Ô hình** (`IllustrationLibraryPage` → `actions`):
  - Thêm **"Xoá"** (`dangerGhost`, cuối hàng nút) cho hình không phải kit / Hình mẫu. Bấm thì hỏi `window.confirm`, giống cách xoá thư mục hiện có. Lỗi 409 hoặc 503 hiện ở dòng `status` sẵn có, với thông báo từ server.
  - Thêm **"Đặt làm mẫu"** (`ghost`) cho hình đã duyệt, không phải kit / Hình mẫu, chưa có bản clone.
  - Hình mẫu: bỏ "Sửa code / Vẽ lại" và "Bỏ duyệt", thêm **"Bỏ làm mẫu"** (`ghost`, có xác nhận).
- **Mục "Luật style của kênh"**: tiêu đề phụ "Hình mẫu (x/5)". Mỗi ô Hình mẫu có nút "Bỏ làm mẫu". Chưa có Hình mẫu nào thì hiện dòng hướng dẫn "Chọn 'Đặt làm mẫu' trên một hình đã duyệt".
- **Trình sửa** (`IllustrationEditor`): `readOnly = builtin || exemplar`. Danh sách thư mục ở trình sửa và ở "AI vẽ" bỏ `hinh-mau`.
- API client: `makeExemplar(id)`, `unmakeExemplar(id)`. `deleteIllustration` giữ nguyên; lỗi server đã mang `message`.

## 5. Test
- **Go domain**: `ReadOnly()`, luật trạng thái chặn / không chặn, tên clone (`Mau`, `Mau2`, đổi `export function`).
- **Go application** (fake repo / port): xoá bị chặn / không bị chặn / orchestrator 404 / orchestrator lỗi; `ignoreProject`; đặt mẫu (đủ 5, đã clone, chưa duyệt, kit); bỏ mẫu (clone xoá, gốc chuyển về); `references` lấy từ repo; chặn ghi vào `hinh-mau`; nhập / xuất Hình mẫu.
- **Go postgres** (DB test sẵn có): bước chuyển dữ liệu chạy 2 lần vẫn đúng; seeder không tạo lại Hình mẫu đã bỏ; truy vấn ứng viên bắt được `code_content` gọi tên và không bắt nhầm tên dài hơn (`Bus` so với `BusStop`).
- **api-gateway**: test route mới (theo `tests/` sẵn có).
- **web-gui (Vitest)**: nút Xoá, Đặt / Bỏ làm mẫu hiện đúng chỗ; thông báo 409 hiện ra; Hình mẫu không có nút sửa.
- `make check`, rồi rebuild `authoring-service`, `api-gateway`, `web-gui` và kiểm tra trực tiếp theo 8 tiêu chí chấp nhận.

## 6. Việc làm theo thứ tự
1. authoring: schema + chuyển dữ liệu + seeder + domain (`ReadOnly`, tên clone, luật trạng thái).
2. authoring: use case xoá có kiểm "đang dùng" + cổng orchestrator; đặt / bỏ mẫu; chặn `hinh-mau`; `references`; loại clone khỏi catalog và `ForCode`; sao lưu.
3. authoring: HTTP handler + DTO; test.
4. api-gateway: 2 route + test.
5. web-gui: client, trang Thư viện, trình sửa; test.
6. `make check`, rebuild, kiểm tra trực tiếp, cập nhật audit.

## 7. ADR
Không cần ADR mới: không thêm service, không đổi contract message, và dùng lại lời gọi HTTP authoring → orchestrator đã có (`GetStatus`). Thay đổi "Hình mẫu từ file nhúng sang dữ liệu DB" được ghi trong tài liệu này và trong audit.
