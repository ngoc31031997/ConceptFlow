# visual-service v1

visual-service giữ các bước tiền kỳ hình ảnh (bước 8–14) và DB riêng `visual-db` (ADR-0049). Mã: `services/visual-service/`.

## HTTP nội bộ

| Bên gọi | Phương thức | Đường dẫn | Việc |
|---|---|---|---|
| orchestrator | DELETE | `/internal/v1/visual/{id}` | Xoá mọi dữ liệu của dự án đã xoá: `review_units`, `visual_concept_versions`, `project_reference_picks`, `project_visual`, trong một transaction. `204`; dự án chưa có dữ liệu cũng `204`. Thư viện `reference_styles` dùng chung nên giữ nguyên |
| bước sau bước 8 | GET | `/internal/v1/visual/{id}/concept` | Tài liệu ngôn ngữ hình ảnh của phiên bản chốt mới nhất ([`visual-concept-v1.md`](visual-concept-v1.md)). `404` khi chưa chốt. `?step=` dành cho các bước 9–14 khi chúng có; hiện chưa bước nào đọc theo tên nên mọi `step` khác rỗng trả `400` |
| compose healthcheck | GET | `/health` | `{"status":"ok"}` |

Orchestrator gọi route xoá ở cuối saga xoá dự án, sau khi mọi service giữ file (`PurgeTargets`) đã báo xong và dòng dự án đã xoá. Giống cách xoá authoring: cố gắng một lần, lỗi thì ghi log, không chặn saga. visual-service không giữ file nên không phải một `PurgeTarget`.

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
