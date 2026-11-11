# visual-service v1

visual-service giữ các bước tiền kỳ hình ảnh (bước 8–14) và DB riêng `visual-db` (ADR-0049). Mã: `services/visual-service/`.

## HTTP nội bộ

| Bên gọi | Phương thức | Đường dẫn | Việc |
|---|---|---|---|
| orchestrator | DELETE | `/internal/v1/visual/{id}` | Xoá mọi dữ liệu của dự án đã xoá: `review_units`, `visual_concept_versions`, `project_reference_picks`, `project_visual`, trong một transaction. `204`; dự án chưa có dữ liệu cũng `204`. Thư viện `reference_styles` dùng chung nên giữ nguyên |
| bước sau bước 8 | GET | `/internal/v1/visual/{id}/concept` | Tài liệu ngôn ngữ hình ảnh của phiên bản chốt mới nhất ([`visual-concept-v1.md`](visual-concept-v1.md)). `404` khi chưa chốt. `?step=` dành cho các bước 9–14 khi chúng có; hiện chưa bước nào đọc theo tên nên mọi `step` khác rỗng trả `400` |
| compose healthcheck | GET | `/health` | `{"status":"ok"}` |

Orchestrator gọi route xoá ở cuối saga xoá dự án, sau khi mọi service giữ file (`PurgeTargets`) đã báo xong và dòng dự án đã xoá. Giống cách xoá authoring: cố gắng một lần, lỗi thì ghi log, không chặn saga. visual-service không giữ file nên không phải một `PurgeTarget`.

## HTTP công khai (qua api-gateway)

### Thư viện tham chiếu

Mô tả phong cách dạng chữ có trường, dùng chung cho mọi video. api-gateway chuyển nguyên request và response (`VISUAL_SERVICE_URL`).

| Phương thức | Đường dẫn | Thân gửi | Trả về |
|---|---|---|---|
| GET | `/v1/reference-styles` | | `200 {"styles":[ReferenceStyle]}`, theo thứ tự thêm (`seq`) |
| POST | `/v1/reference-styles` | `ReferenceStyleInput` | `201 ReferenceStyle` |
| POST | `/v1/reference-styles/import` | JSON đã dán, khuôn của `ReferenceStyleInput` (không cần `id`) | `200 {"style":ReferenceStyleInput}` đã chuẩn hoá; **không ghi DB** |
| PUT | `/v1/reference-styles/{id}` | `ReferenceStyleInput` | `200 ReferenceStyle` |
| DELETE | `/v1/reference-styles/{id}` | | `204`; các lượt chọn của video bị xoá theo |
| GET | `/v1/reference-styles/{id}/usage` | | `200 {"projects":[{"project_id"}]}`: video đang chọn mô tả, để hộp xác nhận sửa hoặc xoá nêu tên |

Khuôn (`ReferenceStyle` = `ReferenceStyleInput` + `id`, `created_at`, `updated_at`):

```json
{ "id": "R01", "name": "", "summary": "", "tags": [],
  "source": { "url": "", "title": "", "kind": "" },
  "drawing": { "medium": "", "shape": "", "outline": "", "fill": "", "shading": "", "light": "", "depth": "", "perspective": "", "texture": "" },
  "layout": { "structure": "", "layering": "", "persistent_elements": "" },
  "created_at": "", "updated_at": "" }
```

`id` là `R` + số từ `reference_styles_seq`, đệm hai chữ số (`R01`, `R12`, `R105`); mã đã xoá không dùng lại.

Luật (đếm theo ký tự, sau khi cắt khoảng trắng hai đầu):
- Chỉ `name` (1–80) và `summary` (1–1000) bắt buộc; mọi trường khác để trống được (chuỗi rỗng, mảng rỗng).
- `tags`: bỏ ô trống và nhãn trùng (không phân biệt hoa thường), tối đa 8 nhãn, mỗi nhãn tối đa 24.
- `source.url` nếu có phải là địa chỉ http hoặc https hợp lệ; `source.kind` là `video`, `image`, `site`, `other` hoặc rỗng; `source.title` tối đa 300.
- Mỗi trường chữ của `drawing` và `layout`: tối đa 300.

`/import` nhận thân `{"error":"…"}` (AI ngoài báo không xem được link) và trả `400` đúng lý do đó.

Lưu trữ: cột `description` giữ `summary`; cột `detail JSONB` giữ `source`, `drawing`, `layout`.

Lỗi `{"error":"…"}`: `400` khi sai luật (thông báo nêu trường sai, ví dụ `drawing.shape: dài quá 300 ký tự`) hoặc thân không phải JSON, `404` khi không có mô tả `{id}` (PUT, DELETE, usage), `500` khi lỗi DB. api-gateway trả `502 {"error":"upstream_unavailable","service":"visual-service"}` khi không gọi được visual-service.

## visual-service gọi đi

| Bên nhận | Đường dẫn | Hợp đồng |
|---|---|---|
| authoring-service | `GET /internal/v1/authoring/{id}/script-versions/latest` | [`script-lock-v1.md`](script-lock-v1.md) |
| rendering | `GET /v1/fonts` → `{"fonts":[…],"default":"…"}` (danh sách font cho phần Chữ của bước 8; lỗi, JSON sai hoặc danh sách trống là lỗi có tên rendering) | [`visual-concept-v1.md`](visual-concept-v1.md) |
| llm-service | `POST /v1/chat`, `POST /v1/chat/external` | [`llm-chat-v1.md`](llm-chat-v1.md) |
| llm-service | `GET /internal/v1/prompts/{role}`, `PUT /internal/v1/prompts/system` (owner `visual`, lúc khởi động, thử lại với quãng nghỉ tăng dần), `GET /internal/v1/step-models`, `GET /internal/v1/models` | [`prompt-library-v1.md`](prompt-library-v1.md) |

## Bảng (`visual-db`)

| Bảng | Khoá | Nội dung |
|---|---|---|
| `review_units` | `(project_id, kind, unit_key)` | Đơn vị duyệt của các bước hình ảnh, cùng dạng với authoring, thêm `input_hashes` (hash từng input lúc viết). Bước 8: `kind` `visual_concept` (mỗi phần của tài liệu visual-concept một đơn vị), `visual_review` (`main`) |
| `visual_concept_versions` | `(project_id, version)` | Bản chốt bất biến: `script_version`, `document` (tài liệu visual-concept), `note`, `locked_at` |
| `project_visual` | `project_id` | Thiết lập riêng từng video: `concept_model` (`''` = mặc định của bước) |
| `reference_styles` | `id` (`R01`…) | Thư viện mô tả phong cách dùng chung (tên, `description` = tóm tắt, `detail` JSONB: `source`, `drawing`, `layout`); `seq` không dùng lại. Bước 8 chỉ đọc |
| `project_reference_picks` | `(project_id, ref_id)` | Mô tả video chọn; xoá mô tả thì xoá theo (`ON DELETE CASCADE`) |

Route công khai của bước 8: [`visual-concept-v1.md`](visual-concept-v1.md).

## Cấu hình

`DATABASE_URL`, `AUTHORING_SERVICE_URL`, `LLM_SERVICE_URL` (bắt buộc); `LLM_TIMEOUT_SECONDS` (mặc định 0 = chờ như llm-service), `MAX_OUTPUT_TOKENS` (mặc định 128000), `RENDERING_URL` (mặc định `http://rendering:8000`, nơi liệt kê font).
