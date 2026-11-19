# illustration-library v1

Thư viện hình minh hoạ do rendering giữ (CR-116, [ADR-0050](../../aidlc-docs/decisions/ADR-0050-rendering-owns-illustrations-kit-is-reference-data.md)): thư mục, hình, duyệt, Hình mẫu, xem trước, bộ vẽ AI, sao lưu. Mã: `services/rendering/` (`domain/illustration.py`, `application/illustration_*.py`, `adapters/http/illustrations.py`, `adapters/persistence/illustration_repository.py`). Hình vẽ theo bộ thông số (kit) của một **phong cách tham chiếu** (`ref_id`, ví dụ `R12`) ở visual-service.

Thiết kế: [cr-116-visuals-to-rendering-design.md](../../aidlc-docs/construction/plans/cr-116-visuals-to-rendering-design.md) §3–§4. Prompt: [`prompt-library-v1.md`](prompt-library-v1.md).

## Quy ước chung

- api-gateway chuyển nguyên request và response tới rendering (`RENDERING_URL`, mạng `backend`).
- Thân yêu cầu và trả về là JSON, trừ ảnh (`image/png`, `image/gif`) và file sao lưu (`application/zip`). Thân không phải JSON, hoặc trường sai kiểu → `400 {"error":"invalid JSON body"}`. Trường lạ bị bỏ.
- Lỗi: `{"error":"<thông báo tiếng Việt cho Creator>"}`. Code hình bị bộ dựng từ chối: `422 {"error","diagnostics":[{"message","line"}]}` (`line` có thể `null`).
- Hình dựng theo `ref_id`: rendering lấy kit bằng `GET /internal/v1/reference-styles/{ref_id}/kit` của visual-service mỗi lần dựng rồi gọi use case xem trước trong cùng tiến trình (cùng hàng đợi một-yêu-cầu-một-lúc với `POST /v1/illustrations/preview`). `ref_id` trống → `400`; phong cách không có → `404`; phong cách chưa có kit → `409`; visual-service hoặc bộ dựng không chạy được → `503`.
- Hình lưu `ref_id` của lần vẽ đầu và giữ nguyên khi sửa. Hình có sẵn của kit (`builtin`) không có `ref_id`: muốn dựng ảnh (xem lần đầu, `rerender`) phải gửi `?ref_id=`.

## Hình (`Illustration`)

```json
{ "id": "…", "kind": "figure|backdrop", "name": "SchoolBus", "title": "", "folder_id": "phuong-tien",
  "tags": [], "description": "", "usage": "", "code": "…",
  "ref_id": "R12", "detail_level": "rich|medium|silhouette", "builtin": false, "exemplar": false,
  "source_id": "…", "home_folder_id": "…", "warnings": [{"message","line"}],
  "status": "draft|approved", "version": 1, "has_preview": true, "created_at": "RFC3339", "updated_at": "RFC3339" }
```

`code`, `source_id`, `home_folder_id` bỏ khỏi JSON khi trống (hình có sẵn không có `code`). `name` là component được export: PascalCase, 2–41 ký tự chữ/số, duy nhất toàn thư viện. `folder_id` là slug chữ thường không dấu. Thẻ được cắt khoảng trắng, hạ chữ thường, bỏ trùng.

- `detail_level` quyết hạng mà bộ kiểm tra tĩnh dùng: `rich` → `hero`, `medium` → `support`, `silhouette` → `background`.
- **Hình có sẵn** (`builtin`, id `builtin-<Tên>`): 34 hình và 6 nền của kit, nạp lại mỗi lần khởi động; **Hình mẫu** (`exemplar`): hình AI học theo, ở thư mục `hinh-mau`, tối đa 5. Cả hai chỉ đọc: sửa/vẽ lại/duyệt/xoá → `403`.
- Đổi `code` hoặc `name` của một hình → về `draft`, `version` + 1, dựng lại ảnh. Chỉ đổi thư mục/tiêu đề/thẻ/mô tả thì giữ trạng thái và không dựng.
- Thư mục `hinh-mau` không nhận hình bằng tay (tạo/sửa/vẽ vào đó → `422`).

## Route

| Route | Việc | Trả về / lỗi |
|---|---|---|
| `GET /v1/illustration-folders` | | `{"folders":[{id,name,description,position,is_system}]}` |
| `POST /v1/admin/illustration-folders` | `{id,name,description}` | `201` thư mục; `400` id/tên sai; `409` id đã dùng |
| `DELETE /v1/admin/illustration-folders/{id}` | chỉ thư mục rỗng của Creator | `204`; `403` thư mục hệ thống; `404`; `409` còn hình |
| `GET /v1/illustration-style[?ref_id=]` | Hình mẫu và mô tả phong cách | `{"exemplar_ids":[…],"max_exemplars":5,"style":null\|{id,name,summary,drawing,layout}}`; `404` khi `ref_id` không có |
| `GET /v1/illustrations?folder=&q=&status=` | `q` khớp tên, tiêu đề, thẻ, mô tả | `{"illustrations":[…]}`, hình có sẵn trước, rồi theo thư mục và tiêu đề |
| `GET /v1/illustrations/{id}` | | hình; `404` |
| `GET /v1/illustrations/{id}/preview.png`, `preview.gif` `[?ref_id=]` | ảnh đã lưu; chưa có cho phiên bản hiện tại thì dựng rồi lưu | `Cache-Control: private, max-age=86400`; `404` nếu không có ảnh loại đó |
| `POST /v1/illustration-tries` | `{name,code,kind,detail_level,ref_id}`; dựng thử, không lưu | `{"png":b64,"gif":b64,"warnings":[…]}`; `422` kèm `diagnostics` |
| `POST /v1/admin/illustrations` | `{name,title,folder_id,tags,description,usage,code,kind,detail_level,ref_id}`; dựng trước khi lưu | `201` hình `draft` `version 1`; `422`/`404` thư mục/`409` tên trùng/`400` |
| `PUT /v1/admin/illustrations/{id}` | như trên; `code`, `folder_id`, `name` trống giữ giá trị cũ (`title` trống lấy theo `name`); `kind`, `detail_level`, `ref_id` đã chốt ở lần vẽ đầu, `ref_id` chỉ nhận khi hình chưa có phong cách và `code` đổi | `200` hình |
| `POST /v1/admin/illustrations/{id}/status` | `{status:"draft"\|"approved"}` | `200` hình; `400` trạng thái lạ; `403` |
| `POST /v1/admin/illustrations/{id}/rerender[?ref_id=]` | dựng lại ảnh phiên bản hiện tại | `{"png":b64,"gif":b64}` |
| `DELETE /v1/admin/illustrations/{id}` | xoá hàng và thư mục ảnh | `204`; `403`; `404` |
| `POST /v1/admin/illustrations/{id}/exemplar` | sao hình đã duyệt (không phải hình có sẵn) vào `hinh-mau` dưới tên `<Tên>Mau` (rồi `Mau2`…) | `201` bản sao; `409` đã có bản mẫu / đủ 5; `422` hình chưa duyệt hoặc là hình có sẵn |
| `DELETE /v1/admin/illustrations/{id}/exemplar` | bản sao bị xoá (`204`); hình mẫu gốc về `home_folder_id` (`200` hình) | `409` không phải Hình mẫu |
| `POST /v1/admin/illustrations/draw` | `{description,folder_id,name,model,kind,detail_level,ref_id}` | `201` hình `draft`; xem mục Bộ vẽ AI |
| `POST /v1/admin/illustrations/{id}/redraw` | `{note,model,ref_id}`; giữ tên, ra phiên bản `draft` mới | `200` hình |
| `GET /v1/admin/illustrations/export` | cả thư viện thành ZIP | `application/zip`, `Content-Disposition: attachment; filename="conceptflow-thu-vien-hinh-YYYYMMDD-HHMM.zip"`, `no-store` |
| `POST /v1/admin/illustrations/import[?on_conflict=skip\|replace][&ref_id=]` | thân là ZIP thô, tối đa 256 MB | `200` báo cáo; `400` file không hợp lệ/`on_conflict` lạ/thân trống; `413` quá lớn |

Mọi lỗi không nêu: `400` với thông báo của luật bị vi phạm. Lỗi không lường trước: `500 {"error"}` và ghi log.

## Bộ vẽ AI

- Tối đa 3 lượt gọi model (`MAX_DRAW_ATTEMPTS`). Mỗi lượt qua llm-service `POST /v1/chat` (`role` = role prompt, `step="illustrations"`, `phase="attempt_N"`, `temperature` 0.5, `max_tokens` = `ILLUSTRATION_MAX_OUTPUT_TOKENS`); không có `model` trong yêu cầu thì lấy model mặc định của bước `illustrations`.
- Câu trả lời đúng khuôn `TÊN / TÊN HIỂN THỊ / THẺ / MÔ TẢ / CÁCH GỌI` và một khối ```` ```tsx ````; sai khuôn hoặc code bị bộ dựng từ chối thì lượt sau gửi lại câu hỏi gốc kèm code lần trước và lỗi có số dòng. Qua 3 lượt vẫn hỏng → `422` (có `diagnostics` nếu lượt cuối là lỗi code) với thông báo bắt đầu "AI vẽ 3 lần vẫn chưa qua kiểm tra".
- Prompt hệ thống đọc từ llm-service mỗi lần vẽ: `illustration_drawer` (hình) hoặc `illustration_backdrop_drawer` (nền). Hình tham chiếu nối vào prompt: Hình mẫu (cho hình, cũ nhất trước) rồi tối đa 2 hình đã duyệt cùng loại và cùng thư mục.
- Redraw giữ tên component; AI đổi tên → lỗi và không lưu.
- Lỗi model (HTTP): `502` key bị từ chối / hết số dư / hết hạn mức token / rỗng / kết quả không dùng được / llm-service hoặc nhà cung cấp lỗi; `429` bị giới hạn tốc độ; `504` quá giờ; `503` chưa cấu hình `HIVE_API_KEY`, llm-service chưa có prompt của role, bộ vẽ tắt.

## Sao lưu

ZIP có `manifest.json` (`format="conceptflow-illustration-library"`, `format_version=1`, `exported_at`, `folders`, `illustrations`) và mỗi hình của Creator (không có hình có sẵn; có Hình mẫu, đánh dấu `exemplar`, `source_name`, `home_folder_id`) gồm `hinh/<thư mục>/<Tên>.tsx`, `.png`, `.gif`. Mỗi mục của `illustrations` mang `name, title, folder_id, tags, description, usage, status, version, created_at, updated_at, kind, detail_level, ref_id, code_file, png_file?, gif_file?`.

Khôi phục: tạo thư mục của Creator còn thiếu; từng hình đi qua bộ dựng với kit của `ref_id` trong file (file cũ không có thì dùng `?ref_id=`, không có cả hai thì hình đó `failed`); ảnh lưu là ảnh dựng lúc này. Trùng tên hình có sẵn/Hình mẫu → `skipped`; trùng tên hình của Creator → `skipped`, hoặc `replaced` (phiên bản + 1, cùng hàng) khi `on_conflict=replace`. Hình mẫu của file không ghi đè hình của Creator; thư mục Hình mẫu đầy thì hình mẫu gốc nhập thành hình thường ở thư mục cũ, bản sao bị bỏ. Báo cáo: `{exported_at, folders_created[], folder_errors[], items:[{name,folder_id,result:"created|replaced|skipped|failed",reason?,diagnostics?}], aborted?, not_processed}`. Bộ dựng/visual-service/database hỏng giữa chừng thì dừng: `aborted` có lý do, `not_processed` là số hình chưa thử, nhập lại cùng file sẽ làm tiếp.

## Dữ liệu

Bảng ở `rendering-db`, tạo bằng `CREATE TABLE IF NOT EXISTS` lúc khởi động (ADR-0013); thư mục hệ thống và 40 hình có sẵn được nạp lại mỗi lần khởi động (hình của Creator đã dùng tên của một hình có sẵn thì giữ tên đó).

- `illustration_folders(id, name, description, position, is_system)`.
- `illustrations(id, name UNIQUE, title, folder_id → folders, tags text[], description, usage, code, builtin, status, version, kind, detail_level, ref_id, exemplar, source_id → illustrations ON DELETE SET NULL, home_folder_id → folders ON DELETE SET NULL, warnings jsonb, preview_version, preview_png_path, preview_gif_path, created_at, updated_at)`. `has_preview` = có đường dẫn PNG và `preview_version = version`.
- Ảnh: `/shared/illustrations/{id}/v{version}.png|.gif` ([`shared-artifacts.md`](shared-artifacts.md)).

## Cấu hình (env của rendering)

`VISUAL_SERVICE_URL`, `LLM_SERVICE_URL`, `LLM_SERVICE_TIMEOUT_SECONDS` (0 = không giới hạn), `ILLUSTRATION_MAX_OUTPUT_TOKENS` (mặc định như `HIVE_MAX_OUTPUT_TOKENS`), `ILLUSTRATIONS_DIR` (mặc định `/shared/illustrations`).

## Khác với thư viện cũ ở authoring-service

- `kit_key` → `ref_id`; thân/phản hồi không còn `kit_key`. `GET /v1/illustration-style` không còn `rules` (luật nằm ở phong cách tham chiếu và prompt), có thêm `style`.
- Bỏ hình theo dự án (`project_illustrations`), bộ lập hình (planner) và ngôn ngữ hình ảnh; vì vậy xoá hình không còn kiểm dự án đang dùng (không còn `409 illustration_in_use`/`503 usage unknown`).
- Ảnh nằm ở file, không ở database. Sao lưu ghi thêm `kind`, `detail_level`, `ref_id` cho mỗi hình.
