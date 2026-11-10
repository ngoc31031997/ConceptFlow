# visual-service v1

visual-service giữ các bước tiền kỳ hình ảnh (bước 8–14) và DB riêng `visual-db` (ADR-0048). Mã: `services/visual-service/`.

## HTTP nội bộ

| Bên gọi | Phương thức | Đường dẫn | Việc |
|---|---|---|---|
| orchestrator | DELETE | `/internal/v1/visual/{id}` | Xoá mọi dữ liệu của dự án đã xoá: `review_units`, `visual_concept_versions`, `project_reference_picks`, `project_visual`, trong một transaction. `204`; dự án chưa có dữ liệu cũng `204`. Thư viện `reference_styles` dùng chung nên giữ nguyên |
| compose healthcheck | GET | `/health` | `{"status":"ok"}` |

Orchestrator gọi route xoá ở cuối saga xoá dự án, sau khi mọi service giữ file (`PurgeTargets`) đã báo xong và dòng dự án đã xoá. Giống cách xoá authoring: cố gắng một lần, lỗi thì ghi log, không chặn saga. visual-service không giữ file nên không phải một `PurgeTarget`.

## visual-service gọi đi

| Bên nhận | Đường dẫn | Hợp đồng |
|---|---|---|
| authoring-service | `GET /internal/v1/authoring/{id}/script-versions/latest` | [`script-lock-v1.md`](script-lock-v1.md) |

## Bảng (`visual-db`)

| Bảng | Khoá | Nội dung |
|---|---|---|
| `review_units` | `(project_id, kind, unit_key)` | Đơn vị duyệt của các bước hình ảnh, cùng dạng với authoring, thêm `input_hashes` (hash từng input lúc viết). Bước 8: `kind` `visual_concept` (mỗi phần của tài liệu visual-concept một đơn vị), `visual_review` (`main`) |
| `visual_concept_versions` | `(project_id, version)` | Bản chốt bất biến: `script_version`, `document` (tài liệu visual-concept), `note`, `locked_at` |
| `project_visual` | `project_id` | Thiết lập riêng từng video: `concept_model` (`''` = mặc định của bước) |
| `reference_styles` | `id` (`R01`…) | Thư viện mô tả phong cách dùng chung; `seq` không dùng lại |
| `project_reference_picks` | `(project_id, ref_id)` | Mô tả video chọn; xoá mô tả thì xoá theo (`ON DELETE CASCADE`) |

Route công khai của bước 8 và thư viện tham chiếu được thêm ở CR-112b và CR-112c.
