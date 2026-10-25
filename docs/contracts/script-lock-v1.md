# Rà soát và chốt kịch bản: API của bước 7 (`/v1`)

Bước Rà soát và chốt (authoring-service) rà soát cả kịch bản bằng máy và AI, giữ ghi chú bàn giao của từng beat cùng bảng thuật ngữ chung, và chốt một phiên bản bất biến để các bước sau đọc (ADR-0044). Mọi đường dẫn công khai nằm dưới `/v1` (ADR-0008) qua api-gateway; chỉ thêm trường thì giữ `/v1`. Mã tương ứng: `services/authoring-service/internal/domain/script_review.go`, `handoff.go`, `script_version.go`, `internal/application/script_lock_*.go`, `handoff_units.go`, `internal/adapters/http/script_lock.go`, `internal/adapters/postgres/script_version_repository.go`.

## Dữ liệu

| Lưu ở | Kind / bảng | Key | Nội dung |
|---|---|---|---|
| `project_handoff` | `handoff_beat` (đơn vị duyệt, qua API đơn vị chung) | `(project_id, beat_key)` | cột `must`, `terms`, `exact`, `free`, `unsure` (jsonb, mảng câu), `status`, `creator_fields`, `redo_count`, `input_ref`, `input_sha`; dựng từ `script_beat:{key}` (`project_script`) nên `stale` khi lời thoại đổi |
| `project_script_review` | `script_review` (không qua API đơn vị chung) | `project_id` | cột `findings`, `next_seq`, `decisions`, `ai_notes`, `my_notes`, `reviewed_sha`, `rounds`, `status`: các mục AI rà soát (`r1`, `r2`… không dùng lại), quyết định của Creator theo mã mục kèm dấu vân tay nội dung mục, ghi chú của AI và của Creator, hash lời thoại lúc AI rà soát gần nhất |
| `project_glossary` | `glossary` (không qua API đơn vị chung) | `(project_id, position)` | một dòng một thuật ngữ: `term`, `say`, `draw`, `avoid`, `mine`; lưu là thay cả bảng của dự án trong một giao dịch |
| `script_versions` | bảng riêng | `(project_id, version)` | ảnh chụp bất biến: `parts[]` gồm `brief`, `facts`, `story`, `order`, `beat:{k}`, `script:{k}`, `handoff:{k}`, `glossary`, `review` (mục đã chấp nhận kèm lý do, ghi chú rà soát), mỗi phần có `sha`; ghi chú phiên bản, lúc chốt. Chỉ thêm, không sửa; trùng số phiên bản thì 409 |

Sửa ghi chú bàn giao qua `PUT /v1/projects/{id}/units/handoff_beat/{key}` với `{"fields":{…}}`: khoá là `must|terms|exact|free|unsure`, giá trị mỗi dòng một mục (rỗng thì bỏ trống mục); `POST …/redo` (AI viết lại cả ghi chú của beat, ghi chú thôi là của Creator); `POST …/approve` (cần có ít nhất một mục). Beat chưa có ghi chú tự viết được bằng đường sửa.

## Đường dẫn công khai

Mọi lời gọi POST dưới đây trả lại toàn màn như GET.

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/script-lock` | Toàn màn: `script_ready`, `reviewed`, `findings[]`, `open_count`, `ai_notes[]`, `my_notes[]`, `glossary[]`, `beats[]` (khoá, số, chức năng, `written`, `status`, `mine[]`, `stale`, `note`), `conditions[]`, `can_lock`, `package` (bộ A), `handoff` (đếm cho bộ B), `sections[]` (năm mục kèm vai đọc), `versions[]`, `locked_version`, `next_version`, `draft[]`, `skipped_research` |
| POST | `/v1/projects/{id}/script-lock/review` | AI rà soát cả kịch bản một lượt, hoặc rà soát lại (so bản hiện tại với các mục đã phát hiện); header `X-Operation-Id` |
| POST | `…/findings/{finding}/apply` | Thay câu `line` của mục AI bằng `fix`: là một lần sửa tay của beat đó, nên beat về chờ duyệt và ghi chú bàn giao của beat thành `stale` |
| POST | `…/findings/{finding}/handoff` | Thêm dòng `handoff` của mục AI vào ghi chú bàn giao của beat; ghi chú về chờ duyệt |
| POST | `…/findings/{finding}/accept` | `{"why":"…"}`; lý do rỗng lưu là “không ghi lý do” |
| POST | `…/notes` | `{"text":"…"}` ghi chú của Creator |
| POST | `…/glossary` | `{"term","say","draw","avoid"}` thêm một dòng thuật ngữ (dòng của Creator, AI không ghi đè) |
| POST | `…/handoff/generate` | AI viết ghi chú bàn giao cho các beat chưa có, từng beat một; `{"beat":"b2"}` chỉ một beat (phải chưa có ghi chú); thuật ngữ AI đề xuất gộp vào bảng, dòng đã có giữ nguyên; header `X-Operation-Id` |
| POST | `…/handoff/{key}/confirm` | Giữ ghi chú như cũ sau khi lời thoại đổi và duyệt lại |
| POST | `…/lock` | `{"note":"…"}` chốt phiên bản `next_version`; rỗng thì ghi chú mặc định “Chốt lần đầu” hoặc “Chốt N thay đổi” |

### Mục rà soát (`findings[]`)

`{id, source:"machine"|"ai", group, group_label, state, title, detail, line, fix, handoff:{beat,section,text}|null, research, why}`. `state`: `open` (giữ nút chốt), `ok` (máy kiểm đạt), `info`, `fixed`, `accepted`. Nhóm: `accuracy`, `research`, `timing`, `logic`, `clarity`, `production`, và `final` cho mục máy “AI chưa rà soát / lời thoại đã đổi sau lần rà soát gần nhất”.

Máy tính lại mỗi lần tải, không lưu: dữ kiện bắt buộc có trong lời thoại; câu dùng dữ kiện có đoạn trích không khớp (hoặc, khi bỏ qua Nghiên cứu, đúng một cảnh báo “chưa có nguồn”); câu có con số hay mốc thời gian mà chưa gắn dữ kiện; tổng thời lượng so với khung; câu dài, câu khó nghe và chỗ nối trùng từ (luật của bước 6); lời thoại đổi sau lần AI rà soát gần nhất. Quyết định “đã sửa” hay “chấp nhận” của Creator gắn với dấu vân tay chữ của mục: mục đổi chữ thì mở lại.

Lần AI rà soát lại: mục cũ mà AI không báo nữa thành `fixed`; mục AI báo lại (`same_as`) giữ mã, mục Creator đã đánh dấu `fixed` thì mở lại vì vẫn còn, mục đã `accepted` giữ nguyên lý do; mục mới lấy mã tiếp theo. Câu trả lời AI không đọc được (JSON hỏng, nhóm lạ, mã câu hay mã beat không có, `same_as` không có) thì không lưu gì: 502.

### Điều kiện chốt (`conditions[]`, `can_lock`)

1. Không còn mục `open` (kể cả “AI chưa rà soát” khi chưa rà soát; chưa cấu hình AI thì chấp nhận kèm lý do).
2. Lời thoại mọi beat đã duyệt.
3. Ghi chú bàn giao mọi beat đã có, đã duyệt và không `stale`.

Chốt khi không đủ điều kiện: 409. Chốt khi bản nháp không đổi gì so với phiên bản mới nhất: 409. `draft[]` liệt kê phần đổi so với phiên bản mới nhất (“Lời thoại · B2”).

## Đường nội bộ

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/internal/v1/authoring/{id}/handoff?role=…` | Phần của phiên bản chốt mới nhất mà vai đó đọc (FR-32 mục 4). `role`: `art_world`, `storyboard`, `art_color`, `voice`, `illustration`, `code`, `validate`. 404 khi chưa chốt; 400 cho vai lạ |

Trả `{version, locked_at, brief?, glossary?, beats:[{key, number, fn?, title?, goal?, next?, lines?, notes:{<mục>:[…]}}]}`. Chưa có bước nào gọi route này; Mỹ thuật lượt 1, Storyboard, Mỹ thuật lượt 2, Giọng, Hình minh hoạ, Code và Validate nối vào khi CR con của chúng vào main. Storyboard, Hình minh hoạ và Code hiện có chưa đọc phiên bản chốt (`flow.go` không đổi).

## Mã lỗi

| Mã | Khi |
|---|---|
| 400 | thao tác không hợp mục (`ErrFindingNotApplicable`), mục bàn giao trống, beat đã có ghi chú khi viết từ đầu, sửa sai mục |
| 404 | không có mục rà soát, beat hay phiên bản đã chốt |
| 409 | script chưa duyệt hết, chưa đủ điều kiện chốt, không có thay đổi để chốt, trùng số phiên bản |
| 502 | câu trả lời AI không dùng được |
| 503 | chưa cấu hình AI |

Đầu vào của AI rà soát: brief đã duyệt (`project_brief`), dữ kiện đang giữ (`project_research`), khung câu chuyện (`project_story`), dàn beat đầy đủ (`project_beat`: thời lượng, nội dung, chỗ nối — `{{outline}}`) và lời thoại (`project_script`). Cờ bỏ qua bước Nghiên cứu đọc từ `project_research_gate`.

Model AI của bước là model Creator chọn ở khoá `story` (như bước Viết kịch bản). Vai prompt: `script_review` (rà soát) và `script_handoff` (ghi chú bàn giao của một beat). Lượt AI ghi nhận dưới kind `script_review` và `handoff_beat`.
