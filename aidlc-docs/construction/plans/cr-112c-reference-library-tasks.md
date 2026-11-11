# CR-112c · Thư viện tham chiếu phong cách — giao việc

Nhánh `feature/cr-112c-reference-library`, worktree `.claude/worktrees/feature-cr-112c-reference-library` (tạo từ main sau CR-112a và CR-113). Chạy song song với CR-112b (`feature/cr-112b-visual-concept-step`).

## Đọc trước
- `aidlc-docs/construction/plans/cr-112-visual-concept-design.md`: mục "Thư viện tham chiếu", bảng dữ liệu, API.
- `aidlc-docs/decisions/ADR-0049-visual-service-owns-visual-preproduction.md`.
- `docs/contracts/visual-service-v1.md`.
- `docs/ux-ui-design-rules.md` và `docs/code-standards-rules.md` (bắt buộc).
- Mockup đã chốt: https://claude.ai/artifact/JSuFKcJQBkQ2qAVZRKRsck, nút "Cài đặt · Thư viện tham chiếu". Đọc bằng Artifact tool, action `read`.
- `CLAUDE.md` của repo (graphify trước, không commit trước khi Creator duyệt, dọn dữ liệu test, rebuild sau khi sửa).

## Phạm vi
Thư viện mô tả phong cách **dạng chữ** (không có file hình), dùng chung cho mọi video. Ở bước 8, video chỉ tick chọn; phần tick chọn thuộc CR-112b.

### visual-service
Bảng `reference_styles` và sequence `reference_styles_seq` đã có (CR-112a). Làm:
- **domain:** `ReferenceStyle {id, name, description, tags[], created_at, updated_at}`.
  - Luật: tên 1–80 ký tự; mô tả 1–1000 ký tự; tối đa 8 nhãn, mỗi nhãn tối đa 24 ký tự.
  - Cắt khoảng trắng hai đầu; bỏ nhãn trùng.
  - Mã `R` + số từ sequence, đệm hai chữ số (`R01`, `R12`, `R105`). Mã đã xoá không dùng lại.
- **application:** `ReferenceLibraryUseCase`: List, Create, Update, Delete, và `UsedBy(id)` trả các `project_id` đang chọn mô tả đó, đọc từ `project_reference_picks`.
- **adapters/postgres:** repository, có test (bỏ qua khi thiếu `TEST_DATABASE_URL`; tự dọn dòng test bằng mã/tên có tiền tố `test-cr112c`).
- **adapters/http:**

  | Phương thức | Đường dẫn | Trả về |
  |---|---|---|
  | GET | `/v1/reference-styles` | `{"styles":[…]}`, theo `seq` |
  | POST | `/v1/reference-styles` | `201` |
  | PUT | `/v1/reference-styles/{id}` | `200` |
  | DELETE | `/v1/reference-styles/{id}` | `204` |
  | GET | `/v1/reference-styles/{id}/usage` | `{"projects":[{"project_id"}]}`, để hộp xác nhận sửa/xoá nêu video bị ảnh hưởng |

  Lỗi `{"error": "…"}`: `400` khi sai luật, `404` khi không có.
- Nối dây trong `cmd/visual-service/main.go`.

### api-gateway
- Proxy `/v1/reference-styles…` sang visual-service theo cách các route khác đang proxy (`services/api-gateway/src/routes/`).
- Thêm `VISUAL_SERVICE_URL` cho api-gateway trong `docker-compose.yml`.
- Có test như các route khác.

### web-gui
- **Trang** `/settings/reference-styles` "Thư viện tham chiếu", bám mockup:
  - nút "+ Thêm mô tả";
  - bảng Mã · Tên · Mô tả · Nhãn · Sửa / Xoá.
- **Hộp thoại** Thêm/Sửa (luật UI 4) gồm Tên, Mô tả, Nhãn (cách nhau bằng dấu phẩy).
- **Xác nhận** (luật UI 5):
  - Xoá luôn có hộp xác nhận.
  - Sửa hoặc xoá một mô tả đang có video chọn thì hộp xác nhận liệt kê các video đó và ghi: "các phần đọc tham chiếu ở bước Ngôn ngữ hình ảnh sẽ cần làm lại; bản đã chốt giữ nguyên".
  - Tên video lấy theo cách các trang khác đang lấy tên dự án.
- **Menu Cài đặt:** thêm mục "Thư viện tham chiếu", phụ đề "mô tả phong cách cho AI".
- **Client API** trong `src/api/client.ts`.
- **Vitest** cho trang và các hộp thoại.

### Tài liệu
- `docs/contracts/visual-service-v1.md`: thêm các route công khai.
- `docs/contracts/README.md`: thêm dòng web-gui → api-gateway → visual-service.

## Không làm
- Tick chọn mô tả cho video (`PUT /v1/projects/{id}/visual-concept/refs`): thuộc CR-112b.
- Bất kỳ phần AI nào.

## Trùng với CR-112b
Cả hai CR đều sửa `services/visual-service/cmd/visual-service/main.go`, `internal/adapters/http/router.go`, `docker-compose.yml` (api-gateway), `services/web-gui/src/api/client.ts`, `App.tsx` và menu. Giữ thay đổi ở các file này **chỉ là thêm vào**, không sắp xếp lại code có sẵn.

Trước khi báo xong: `git fetch` rồi merge `origin/main`. Nếu có xung đột không phải dạng "hai bên cùng thêm dòng" thì dừng và báo, không tự giải.

## Xong khi
- `go vet`, `go test` cho visual-service đều qua.
- Test api-gateway và vitest web-gui đều qua.
- Đã rebuild `visual-service`, `api-gateway`, `web-gui` bằng `scripts/worktree.sh rebuild …` và cả ba healthy.
- Đã kiểm tay trên stack: thêm, sửa, xoá một mô tả có tên bắt đầu bằng `test-cr112c`, rồi xoá nó.
- **Không commit.** Báo lại cho Creator: đã làm gì, test nào chạy và kết quả, đã dọn gì, chỗ nào khác mockup.
