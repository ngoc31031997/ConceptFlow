# CR-050 — Workflow Planning và thiết kế Unit 1

## Date
2026-09-29

## Các unit
Requirements: `aidlc-docs/inception/requirements/cr-050-llm-call-chunking-resume.md`.

| Unit | FR | Nội dung | Thiết kế cần trước khi code | Trạng thái |
|---|---|---|---|---|
| U1 | FR-17..FR-20 | Sửa lỗi từ review: danh sách hình cũ, không xoá khi lỗi, rào chắn chọn model cho bước Code, chuỗi dừng khi storyboard có cảnh báo | Mục "Thiết kế Unit 1" bên dưới. Không đổi contract giữa service, chỉ thêm một cột DB | Code xong, chờ duyệt |
| U2 | FR-1..FR-9, FR-21, FR-22 | Lưu từng đoạn Code vào DB, chạy tiếp khi lỗi, chạy lại hoặc dùng AI ngoài từng đoạn, ghi usage từng lượt, lưu chẩn đoán, UI danh sách đoạn | **ADR-0030** (bảng đoạn, contract HTTP theo đoạn giữa `authoring-service` ↔ `llm-service`, dấu vân tay) + Functional Design | Chờ thiết kế |
| U3 | FR-10, FR-11 | Storyboard theo cảnh: khung chung, rồi mỗi cảnh một lượt; ghép và finalize | Functional Design (prompt khung chung, prompt cảnh), dùng lại hạ tầng đoạn của U2 | Sau U2 |
| U4 | FR-12..FR-15 | Shot Spec: schema, trình biên dịch, kiểm trên spec, Pha 0 đo trên `f7103848` | **ADR-0031** (định dạng Shot Spec, nơi đặt trình biên dịch, versioning) + Functional Design | Sau U2 (dùng đoạn của U2) |
| U5 | FR-16 | Shot Spec vào luồng chính, chọn được theo project | Theo kết quả Pha 0 | Sau khi Creator chấm Pha 0 |

Các stage bỏ qua cho CR này: User Stories (Creator không chọn), Units Generation mới (5 unit trên đủ), Infrastructure Design (không đổi hạ tầng, chỉ thêm bảng và cột), Operations.

## Thiết kế Unit 1

### FR-17 — Danh sách hình minh hoạ biết khi nào mình đã cũ
- **Dữ liệu**: `project_authoring.illustrations_storyboard_sha TEXT NOT NULL DEFAULT ''`, là sha256 của storyboard đã dùng để lập danh sách.
  - Storyboard được lưu ở dạng chuẩn hoá (qua `storyboard/finalize`), nên cùng nội dung thì cùng hash.
- **Cũ** khi: hash đã lưu khác rỗng **và** khác hash của storyboard hiện tại.
  - Danh sách lập trước CR-050 có hash rỗng; không có căn cứ để coi là cũ, nên không đánh dấu (không bắt project cũ phải lập lại).
- **Repo**: `MarkIllustrationsPlanned(ctx, projectID, storyboardSHA)`; `IllustrationsPlanned` trả thêm hash.
- **Use case**:
  - `Plan` ghi hash của storyboard nó đã đọc.
  - `Stale(ctx, projectID)` so hai hash.
  - `List` trả kèm cờ `stale` cho giao diện.
- **Lập lại mà không vẽ lại**:
  - `Prepare` (bước Hình minh hoạ trong chuỗi) lập lại danh sách khi danh sách đã cũ.
  - `Plan` **giữ lại** hàng cùng tên từ danh sách cũ nếu hàng đó đã có hình (đã vẽ / chờ duyệt / dùng lại) hoặc đã bị Creator bỏ qua. Chỉ cập nhật danh sách shot của hàng đó. Nhờ vậy hình đã vẽ, kể cả bản nháp chưa duyệt, không bị vẽ lại.
- **Bước Code** từ chối chạy khi danh sách đã cũ, với lỗi mới `ErrIllustrationsStale`: "Danh sách hình minh hoạ được lập từ storyboard cũ — chạy lại bước Hình minh hoạ".
- **Giao diện**: `ProjectIllustrationsPanel` hiện dải cảnh báo khi `stale`, cạnh nút "Lập lại danh sách" sẵn có.

### FR-18 — Lượt AI lỗi không xoá các bước sau
- Bỏ ba lời gọi `clearDownstream` trên đường lỗi của `runInner` (lỗi provider, nội dung rỗng, finalize lỗi), và bỏ luôn hàm đó.
- Việc xoá theo tầng khi **lưu nội dung mới** vẫn giữ nguyên, ở `SaveAuthoringStory/Storyboard`.

### FR-19 — Rào chắn chọn model cho bước Code
- **Luật (domain)**: `ModelAllowedForStep(step, id)`. Bước `code` từ chối `ollama` và `ollama/<model>`: 5/5 lượt timeout đã đo, và model local không đủ sức viết code Remotion/Manim. Bước Hình minh hoạ dùng model của bước Code nên cùng luật.
- **Chặn ở 3 chỗ**:
  - Lưu lựa chọn model (`SaveAuthoringModels`) trả 400.
  - `runCode` / `runIllustrations` báo lỗi rõ cho project đã lưu lựa chọn cũ.
  - Danh mục trả về giao diện có `code_ok: false`, để dropdown Code không liệt kê model đó.
- **Số liệu đo**:
  - `GET /v1/llm/status` trả thêm `code_stats` theo model, tính từ `llm_usage` (bước code, phase chunk, 30 ngày): số lượt, số lượt lỗi theo loại, trung bình token ra và thời gian của lượt thành công.
  - Dropdown Code hiện một dòng số liệu của model đang chọn.
  - Khi model có lỗi `timeout`/`budget`/`truncated` thì dòng đó hiện ở dạng cảnh báo.
  - Không có số liệu thì ghi "chưa có số liệu".

### FR-20 — Chuỗi dừng khi storyboard có cảnh báo
- Trong `AuthoringChainRunner.run`: nếu bước `storyboard` xong **có cảnh báo** và chuỗi còn bước sau, chuỗi kết thúc ở trạng thái `Waiting` (đã có từ CR-045), với `WaitingStep = "storyboard"` và câu: "Storyboard có N cảnh báo — xem lại, sửa nếu cần, rồi chạy tiếp bước Hình minh hoạ/Code".
- Giao diện đã sẵn: theo `waiting_step` chuyển về tab Visual, hiện câu chờ cùng khối cảnh báo (CR-048). Creator chạy tiếp từ tab sau như bình thường.

### Kiểm thử Unit 1
- `authoring-service` (Go):
  - Plan ghi hash; Stale đúng/sai và trường hợp hash rỗng; Prepare lập lại khi cũ và giữ hàng đã vẽ hoặc đã bỏ qua; runCode từ chối khi cũ.
  - Không xoá bước sau khi lượt lỗi (test cũ kỳ vọng xoá được sửa theo FR-18).
  - Luật model và lỗi 400 khi lưu; `code_stats`.
  - Chuỗi dừng khi có cảnh báo, và không dừng khi không có cảnh báo hoặc storyboard là bước cuối.
- `web-gui` (vitest): dải cảnh báo danh sách cũ; dropdown Code lọc `code_ok` và hiện số liệu.
- `make check`; rebuild `authoring-service` và `web-gui`, kiểm healthy.
