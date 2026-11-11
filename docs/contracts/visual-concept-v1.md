# Ngôn ngữ hình ảnh: API của bước 8 (`/v1`)

Bước 8 (visual-service, ADR-0049) đọc kịch bản đã chốt ở bước 7, cho AI viết tám phần của ngôn ngữ hình ảnh, để Creator sửa, duyệt, cho AI rà soát cả bộ và chốt một phiên bản bất biến cho các bước hình ảnh sau đọc. Mọi đường dẫn công khai đi qua api-gateway (route AI không có timeout ở gateway). Mã: `services/visual-service/internal/domain/` (`visual_concept.go`, `concept_checks.go`, `findings.go`, `document.go`, `input.go`, `prompt_seeds.go`), `internal/application/concept_*.go`, `internal/adapters/http/concept.go`.

## Đầu vào

Một lần đọc `GET /internal/v1/authoring/{id}/script-versions/latest` ([`script-lock-v1.md`](script-lock-v1.md)) và các mô tả tham chiếu video đã chọn (`project_reference_picks` nối `reference_styles`: `description` là `summary`, cột `detail` JSONB giữ `source`, `drawing`, `layout`; `detail` rỗng đọc là mọi trường trống). Tham chiếu chỉ cho cách vẽ và layout; màu, bố cục, nhân vật, chữ, chuyển động do AI tự suy ra theo chủ đề và cảnh.

| Nguồn (khoá hash) | Lấy từ phần của bản chốt | Hash |
|---|---|---|
| `brief` | `brief` | `sha` của phần |
| `research` | `facts` (danh sách rỗng = bỏ qua Nghiên cứu, cờ `research_skipped`) | `sha` |
| `story` | `story` | `sha` |
| `beats` | `order`, `beat:{k}` | hash của các `sha` |
| `script` | `script:{k}`, `handoff:{k}`, `glossary` | hash của các `sha` |
| `refs` | mô tả đã chọn (mã, tên, `summary`, `drawing`, `layout`) | hash của các trường đó; chỉ phần 1 và 3 đọc |

Authoring trả `404` → `409 {"error":"Chốt kịch bản ở bước 7 trước","code":"script_not_locked"}`. Lỗi khác của authoring → `502` có tên authoring-service.

## Tám phần (đơn vị duyệt `kind = visual_concept`)

| # | `unit_key` | Nguồn đọc | Kiểm khi đọc/lưu |
|---|---|---|---|
| 1 | `creative_direction` | brief, story, beats, **refs** | `abstraction_level`, `abstraction_map[].level` thuộc `realistic, semi_real, stylized, character, diagram, symbol, abstract, hybrid`; `refs_used` ⊆ mô tả đang chọn |
| 2 | `color_system` | story, beats | hex `#rrggbb`; `token` duy nhất; mỗi beat ở đúng một đoạn `journey`; `accent_token`, `background_token`, `contrast.text_token`, `exceptions[].token` có trong `palette` |
| 3 | `visual_language` | beats, **refs** | `shape_rules`, `line_rules`, `lighting_rules`, `depth_rules`, `composition_rules` không rỗng; `surface_rules` không rỗng khi tham chiếu có `fill`, `texture` hay `medium`; `layout_rules` không rỗng khi tham chiếu có `layout` |
| 4 | `representation_rules` | research, script | `concept_id` duy nhất; `facts` là mã có thật |
| 5 | `typography` | brief, script | `font` có trong danh sách font của rendering (`GET {RENDERING_URL}/v1/fonts`, đọc mỗi lần mở màn, không cache, không danh sách dự phòng). Rendering không trả lời thì phần 5 có `problems` nêu rendering: không duyệt, không viết bằng AI, không chốt được (các phần khác không bị ảnh hưởng) |
| 6 | `motion_language` | story, beats | mỗi beat ở đúng một đoạn `pace`; `pace` ∈ `slow, medium, fast` |
| 7 | `do_not_use` (`{items:[…]}`) | research, script | `source` ∈ `accuracy, style`; `facts` có thật |
| 8 | `feasibility` | brief, beats | `scope.beats` và `scope.characters` do máy đếm (ghi đè); `verdict: risk` phải có `mitigation` |

Phần 3 ánh xạ từ tham chiếu: `drawing.shape` → `shape_rules`; `drawing.outline` → `line_rules`; `drawing.light` + `drawing.shading` → `lighting_rules`; `drawing.depth` + `drawing.perspective` → `depth_rules`; `drawing.fill` + `drawing.texture` + `drawing.medium` → `surface_rules`; nhóm `layout` → `layout_rules`. `composition_rules` do AI đặt theo chủ đề và cảnh.

Khuôn từng phần như tài liệu bản chốt trong thiết kế, thêm: `color_system.journey[].background_token` (nền sau chữ của đoạn; rỗng = `background`), `color_system.contrast.text_token` (màu chữ) và `representation_rules.scientific_concepts[].term`, để máy kiểm được tương phản và phủ thuật ngữ.

Phần không qua kiểm vẫn lưu (hiện `problems[]`) nhưng không duyệt được. Đọc JSON chặt: khoá lạ hoặc sai kiểu thì AI trả về là `502` không lưu gì, sửa tay là `400`.

**Stale:** mỗi phần lưu `input_hashes` của các nguồn nó đọc lúc viết (hoặc lúc duyệt). Nguồn nào khác hash hiện tại thì phần `stale` và `stale_sources[]` nêu tên nguồn. Đổi lựa chọn hoặc sửa nội dung một mô tả chỉ làm phần 1 và 3 thành stale. Duyệt một phần stale là xác nhận nó vẫn hợp (hash được làm mới); "Duyệt tất cả" bỏ qua phần stale.

**Máy kiểm cả bộ** (tính lại mỗi lần đọc): `m-contrast` (chữ đạt `text_min` trên nền từng đoạn `journey`, WCAG 2), `m-terms` (mỗi thuật ngữ của `glossary` có một `term` trùng chữ ở phần 4, không phân biệt hoa thường), `m-accuracy` (mỗi dữ kiện `hypothesis` hoặc `misconception` mà lời thoại dẫn có một dòng `source = accuracy` ở phần 7). Creator chấp nhận kèm lý do một mục chưa đạt; quyết định gắn dấu vân tay của `detail`, chi tiết đổi thì mở lại.

## Đường dẫn công khai

Mọi lời gọi trả lại toàn màn như GET.

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/visual-concept` | Toàn màn: `input`, `input_view[]` (văn bản AI đọc), `refs[]` (thư viện, `picked`), `units[]`, `approved_count`, `checks[]`, `fonts[]`, `fonts_error` (lý do rendering không liệt kê được font, `""` khi được), `review`, `criteria[]` (5 tiêu chí), `blockers[]`, `can_lock`, `versions[]`, `locked_version`, `next_version`, `draft[]`, `model`, `default_model`, `models[]` |
| PUT | `…/refs` | `{ref_ids[]}`; mã không có trong thư viện → `400`, không ghi gì |
| PUT | `…/model` | `{model}`; `''` = mặc định của bước `visual_concept` ở llm-service; model ngoài danh mục → `400` |
| POST | `…/generate` | `{note, rewrite_all}`: AI viết lần lượt 1→8 các phần chưa có (hoặc mọi phần khi `rewrite_all`), phần sau đọc phần trước; mỗi phần lưu ngay khi có kết quả; lỗi thì dừng, phần đã viết giữ lại. Không nhận `X-External-AI` |
| POST | `…/units/{key}/generate` | `{note}`; một lượt AI; giữ các trường trong `creator_fields`; phần về `pending`. Nhận `X-External-AI: prompt|paste` ([`external-ai-v1.md`](external-ai-v1.md)) |
| PUT | `…/units/{key}` | `{fields:{trường cấp đầu: giá trị JSON}}`; trường thành của Creator; phần về `pending`; phần chưa có thì bắt đầu từ rỗng |
| POST | `…/units/{key}/approve` · `/unapprove` | Duyệt cần phần đã viết, không có `problems`, AI không đang viết (`409`) |
| POST | `…/approve-all` | Duyệt mọi phần đã viết, chờ duyệt, không `problems`, không stale |
| POST | `…/review` | `{note}`; một lượt AI vai `visual_concept_review`; nhận `X-External-AI` |
| POST | `…/findings/{fid}/apply` | Ghi `fix` vào `field` của phần (trường chữ: thay; danh sách chữ: thêm dòng); trường thành của Creator; phần về `pending`; vấn đề thành `applied`. Không có `field`/`fix` hoặc trường có cấu trúc → `400` |
| POST | `…/findings/{fid}/accept` | `{why}` (rỗng = “không ghi lý do”); `fid` là mã vấn đề (`f1`…) hoặc mã máy kiểm (`m-…`) đang chưa đạt |
| POST | `…/lock` | `{note}` (≤ 200 ký tự; rỗng = “Chốt lần đầu” / “Chốt N thay đổi”) |
| GET | `…/versions/{n}` | `{version, script_version, document, note, locked_at}` |

### Rà soát

AI trả `{"findings":[{group, unit_key, field, text, detail, fix, same_as}], "notes":[…]}`. `group` ∈ `accuracy, consistency, clarity, feasibility`. Nhóm lạ, phần lạ, trường không có ở phần, nội dung rỗng, `same_as` không có hoặc lặp → `502`, không lưu gì. Rà soát lại: vấn đề cũ AI không báo nữa thành `fixed`; báo lại (`same_as`) giữ mã và mở lại (trừ `accepted`, giữ lý do); vấn đề mới lấy mã tiếp theo. Lưu ở đơn vị `visual_review/main`: `findings`, `next_seq`, `notes`, `rounds`, `checks` (quyết định cho máy kiểm).

### Điều kiện chốt

1. Đủ 8 phần, đã duyệt, không phần nào stale.
2. Không còn vấn đề `open`.
3. Mỗi máy kiểm cả bộ đạt hoặc đã chấp nhận kèm lý do.
4. Có thay đổi so với phiên bản mới nhất (`draft[]` liệt kê phần đổi, kịch bản chốt đổi số, mô tả tham chiếu đổi); không đổi gì → `409`.

Chưa đủ → `409` kèm lý do. Trùng số phiên bản → `409`.

## Tài liệu bản chốt (`visual_concept_versions.document`)

Khuôn JSON trong thiết kế: `schema_version` `1.0`, `document_id` `visual-concept-v{n}`, `status` `locked`, `version`, `script_version`, `locked_at`, tám phần (phần 5 nằm trong `representation_rules.typography`, phần 7 là mảng `do_not_use`), `accuracy_notes[{fact, note}]` (máy gom từ `facts` của phần 4 và các dòng `accuracy` của phần 7), `reference_styles[{ref_id, name, summary, drawing, layout}]` (chép nguyên văn), `review{reviewer, approved_at, comments[{id, group, text, decision, why}]}`.

## Đường nội bộ

`GET /internal/v1/visual/{id}/concept` trả nguyên tài liệu của phiên bản chốt mới nhất; `404` khi chưa chốt. `?step=` để dành cho các bước 9–14; hiện chưa có bước nào đọc theo tên nên `step` khác rỗng trả `400`.

## AI

Vai prompt `visual_concept_<unit_key>` (8) và `visual_concept_review`, seed ở `prompt_seeds.go`, đẩy lên llm-service lúc khởi động với `owner = "visual"`. Gọi `/v1/chat` với `expect_json`, `step = visual_concept`, `role` = vai, `model` = `project_visual.concept_model` hoặc mặc định của bước. AI ngoài: `/v1/chat/external`; chỉ ghi DB sau khi có kết quả (ADR-0047). Phần đang được AI viết đọc là `running` trong lúc request mở (không lưu).

## Mã lỗi

| Mã | Khi |
|---|---|
| 400 | sửa sai khuôn, mã tham chiếu lạ, model lạ, vấn đề không áp dụng được, `step` lạ, AI ngoài nhiều lượt |
| 404 | phần, vấn đề hay phiên bản không có |
| 409 | chưa chốt kịch bản (`script_not_locked`), AI đang viết, phần chưa đạt kiểm hoặc chưa viết, chưa đủ điều kiện chốt, không có thay đổi, trùng phiên bản |
| 502 | câu trả lời AI không dùng được; llm-service hoặc authoring-service lỗi |
