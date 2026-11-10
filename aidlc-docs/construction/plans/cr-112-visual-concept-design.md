# CR-112 · Bước 8 Ngôn ngữ hình ảnh (Visual Concept) và `visual-service` — thiết kế

Nhánh `feature/cr-112-visual-concept` (worktree riêng). Nguồn:
- spec Visual Concept của Creator, các quyết định trong session 2026-10-28 đến 2026-11-10;
- mockup https://claude.ai/artifact/JSuFKcJQBkQ2qAVZRKRsck (v3);
- ADR-0048, ADR-0047, ADR-0035.

Khuôn của các bước AI: bước 5–7 (CR-108, CR-109, CR-111).

## Creator đã chốt
1. Flow còn **23 bước**: bỏ bước Phân rã kịch bản; bước 8 là Ngôn ngữ hình ảnh; các bước sau lùi một số.
2. Bước 8 làm theo spec Visual Concept: 9 việc; output `visual-concept` và `visual-concept-review`; 5 tiêu chí hoàn thành.
3. **Tham chiếu phong cách chỉ là mô tả dạng chữ**, lưu ở Cài đặt › Thư viện tham chiếu, dùng chung mọi video. Bước 8 tick chọn mô tả. Không có file hình; AI không đọc hình.
4. **Phong cách do AI đề xuất tự do**: bỏ quyết định cũ "một ngôn ngữ hình được ghim".
5. Style frame làm sau.
6. Service mới `visual-service`. Prompt và model mặc định nằm ở llm-service (CR-113). visual-service tham gia saga xoá video.
7. **Bản chốt dùng khuôn JSON của Creator**, có bổ sung các trường để khớp giao diện (xem "Tài liệu bản chốt").

## Phụ thuộc và trùng lặp
- **Cần CR-113 xong trước** (thư viện prompt ở llm-service). Phần không dùng prompt (service, DB, route đọc, thư viện tham chiếu, sửa tay, duyệt) làm được song song.
- Cần bước 7 (kịch bản chốt, `script_versions`, ghi chú bàn giao) đã có trên main.
- **Chạm vào:** authoring (một route đọc), orchestrator (route xoá cuối saga), api-gateway, docker compose, web-gui. CR-113 cũng chạm authoring và api-gateway nhưng ở phần khác (prompt), nên merge cần pull main mới nhất.

## Input
| Input | Nguồn | Phần dùng |
|---|---|---|
| creative-brief | bước 2 | khán giả, biết, hiểu, cảm nhận, giọng điệu, nền tảng, thời lượng, ngoài phạm vi |
| research-dossier | bước 3 | dữ kiện đang giữ: mã, nội dung, loại (fact, hyp, unk, myth), ghi chú. Bỏ qua Nghiên cứu thì danh sách rỗng và có cờ `research_skipped` |
| story-structure | bước 4 | câu hỏi mở đầu, lời hứa, chuỗi nhân quả, payoff |
| beat-sheet | bước 5 | key, chức năng, tiêu đề, mục tiêu, thời lượng |
| script-final | bản chốt bước 7 | số phiên bản, lời thoại theo beat, ghi chú bàn giao, bảng thuật ngữ |
| tham chiếu | visual-service | các mô tả video này đã chọn |

**Điều kiện vào:** đã có kịch bản chốt (authoring trả 404 khi chưa có). Chưa có thì `409 script_not_locked`, và màn báo "Chốt kịch bản ở bước 7 trước".

## Tài liệu bản chốt (`document`)
Khuôn là JSON của Creator; khoá có dấu `+` là bổ sung.

```jsonc
{
  "schema_version": "1.0", "project_id": "…", "document_id": "visual-concept-v{n}",
  "status": "locked", "+version": 1, "+script_version": 2, "+locked_at": "…",
  "creative_direction": { "style_name", "description", "abstraction_level", "tone[]", "principles[]",
    "+why", "+feeling", "+refs_used[]", "+abstraction_map[{for, level, example}]", "+examples[{do, dont}]" },
  "color_system": { "background", "palette[{token, hex, role, +name}]",
    "+journey[{beat_keys[], mood, accent_token}]", "+contrast{text_min, subject_vs_bg, accent_per_frame}",
    "+exceptions[{token, beat_keys[], why}]", "+examples[]" },
  "visual_language": { "shape_rules[]", "line_rules[]", "lighting_rules[]", "depth_rules[]", "+composition_rules[]", "+examples[]" },
  "representation_rules": {
    "characters": { "approach", "allowed_variations[]", "+items[{concept_id, term, draw, must_not_imply[], facts[]}]" },
    "+objects[]", "+invisible[]",
    "scientific_concepts[{concept_id, representation, must_not_imply[], +facts[]}]",
    "typography": { "font_direction", "label_style", "+font", "+labels{max_words, when, min_px}", "+captions", "+terms[{say, show_on_screen}]", "+formulas" },
    "+examples[]" },
  "motion_language": { "principles[]", "preferred_transitions[]", "avoid[]", "+pace[{beat_keys[], pace}]", "+character_motion", "+examples[]" },
  "do_not_use[{text, +why, +facts[], +source: accuracy|style}]",
  "+feasibility": { "scope{beats, est_shots, settings, characters}", "checks[{item, assessment, verdict: ok|risk, mitigation}]" },
  "+accuracy_notes[{fact, note}]",
  "+reference_styles[{ref_id, name, description}]",
  "review": { "reviewer": "creator", "approved_at", "comments[{+id, +group, text, +decision, +why}]" }
}
```

So với JSON gốc: `do_not_use` đổi từ mảng chuỗi sang mảng object; `reference_assets` (đường dẫn file) đổi thành `reference_styles` (văn bản); `review.comments` lấy từ các vấn đề rà soát.

## 8 phần trên màn hình (đơn vị duyệt)
Mỗi phần là một đơn vị `kind = visual_concept`, `unit_key` là tên phần trong tài liệu. Cột "Việc" là số việc trong spec.

| # | `unit_key` | Việc | Nằm ở đâu trong tài liệu | Máy kiểm khi lưu |
|---|---|---|---|---|
| 1 | `creative_direction` | 1, 2 | `creative_direction` | `abstraction_level` và `level` thuộc enum; `refs_used` ⊆ mô tả đang chọn |
| 2 | `color_system` | 3 | `color_system` | hex hợp lệ; `token` duy nhất; mọi beat có trong `journey`; `accent_token` có trong `palette` |
| 3 | `visual_language` | 4 | `visual_language` | không mảng nào rỗng |
| 4 | `representation_rules` | 5 | `representation_rules` (trừ `typography`) | `concept_id` duy nhất; `facts` là mã có thật |
| 5 | `typography` | 6 | `representation_rules.typography` | `font` có trong danh sách font của rendering |
| 6 | `motion_language` | 7 | `motion_language` | mọi beat có `pace`; `pace` thuộc enum |
| 7 | `do_not_use` | 8 | `do_not_use` | `source` thuộc enum; `facts` có thật |
| 8 | `feasibility` | 9 | `feasibility` | `scope` do máy đếm (AI không ghi đè); `verdict: risk` thì phải có `mitigation` |

`accuracy_notes` không phải đơn vị duyệt: lúc chốt, máy gom từ `facts` của phần 4 và các dòng `source = accuracy` của phần 7.

**Máy kiểm cả bộ** (thẻ đầu, tính lại mỗi lần đọc, không lưu):
- chữ đạt `contrast.text_min` trên nền của từng phần trong `journey`;
- mọi thuật ngữ trong ghi chú bàn giao có một `term` ở phần 4;
- mọi dữ kiện loại `hyp` hoặc `myth` đang dùng trong lời thoại có ít nhất một dòng `accuracy` ở phần 7.

## Luồng nghiệp vụ
1. **Mở màn** (`GET /visual-concept`):
   - đọc input từ authoring;
   - tính hash từng nguồn và so với `input_hashes` của từng phần; phần nào khác thì đánh dấu `stale` và liệt kê nguồn đã đổi;
   - tính máy kiểm;
   - trả toàn màn.
2. **AI viết** một phần (`POST /units/{key}/generate`, ghi chú không bắt buộc):
   - một lượt `Chat`, prompt `visual_concept_<key>`, đọc input cộng các phần đã có;
   - kiểm reply theo schema; giữ các trường trong `creator_fields`;
   - chỉ ghi DB sau khi có kết quả (ADR-0047);
   - phần về `pending`.
3. **AI viết các phần còn trống** (`POST /generate`): viết lần lượt theo thứ tự 1→8, phần sau đọc phần trước. Đây là nhiều lượt nên không có AI ngoài. "Chạy tất cả bằng DeepSeek" xếp từng phần vào hàng đợi extension, giống CR-111.
4. **"Viết lại cả bộ quy tắc":** hộp xác nhận nêu số phần đã duyệt sẽ về chờ duyệt; trường Creator sửa tay giữ nguyên.
5. **Sửa tay** (`PUT /units/{key}`): ghi đè trường; thêm đường dẫn trường vào `creator_fields`; phần về `pending`.
6. **Duyệt / bỏ duyệt** từng phần, và "Duyệt tất cả phần còn lại". Không duyệt được khi `running` hoặc schema chưa hợp lệ.
7. **Rà soát** (`POST /review`, một lượt AI, prompt `visual_concept_review`):
   - AI trả `findings[{group, unit_key, field, text, detail, fix}]`;
   - "Rà soát lại" so với vấn đề cũ, giống vòng 6 ở bước 7: vấn đề đã hết thì đánh dấu đã sửa, vấn đề mới thì thêm vào.
   - Xử lý từng vấn đề:
     - **Áp dụng gợi ý:** ghi `fix` vào `field`; trường đó thành của Creator; phần về `pending`.
     - **Chấp nhận kèm lý do.**
     - **Mở phần đó.**
8. **Chốt** (`POST /lock`). Điều kiện:
   - 8/8 phần đã duyệt và không phần nào `stale`;
   - không còn vấn đề mở;
   - máy kiểm cả bộ đạt, hoặc đã chấp nhận kèm lý do.

   Khi chốt, hệ thống ghép tài liệu và ghi `visual_concept_versions` với `status: locked`. Từ v2 trở đi có hộp xác nhận nêu các phần đã đổi so với bản trước.
9. **Thư viện tham chiếu:**
   - Thêm, sửa, xoá mô tả.
   - Sửa hoặc xoá một mô tả đang được video nào đó chọn thì hộp xác nhận liệt kê các video đó. Hash nguồn `refs` đổi nên các phần đọc tham chiếu thành `stale`. Bản chốt giữ nguyên vì nó chép văn bản.
   - Tick chọn ở bước 8: `PUT /visual-concept/refs`.

## Dữ liệu (`visual-db`)
| Bảng | Khoá | Cột |
|---|---|---|
| `review_units` | `(project_id, kind, unit_key)` | `status` (pending, approved, running), `content JSONB`, `creator_fields TEXT[]` (đường dẫn trường), `input_hashes JSONB`, `redo_count`, `created_at`, `updated_at`. `kind`: `visual_concept`, `visual_review` (`unit_key = main`, `content` = findings và quyết định, có dấu vân tay để quyết định cho mục máy kiểm tự hết hiệu lực khi nội dung đổi) |
| `visual_concept_versions` | `(project_id, version)` | `script_version`, `document JSONB`, `note`, `locked_at` |
| `project_visual` | `project_id` | `concept_model` (`''` = mặc định ở llm-service), `created_at`, `updated_at` |
| `reference_styles` | `id` (`R01`…) | `seq BIGSERIAL` (mã không dùng lại), `name`, `description`, `tags TEXT[]`, `created_at`, `updated_at` |
| `project_reference_picks` | `(project_id, ref_id)` | khoá ngoại tới `reference_styles`, `ON DELETE CASCADE` |

Schema tạo bằng `CREATE TABLE IF NOT EXISTS` lúc khởi động, như authoring. Purge xoá các dòng của dự án trong 4 bảng; `reference_styles` giữ nguyên.

## Thiết kế mức thấp
### visual-service (Go, `services/visual-service`)
| Lớp | Tệp |
|---|---|
| domain | `visual_concept.go` (8 phần, schema, kiểm từng phần, ghép tài liệu, `accuracy_notes`), `concept_checks.go` (tương phản WCAG, phủ thuật ngữ, phủ giả thuyết và ngộ nhận), `review_unit.go`, `findings.go` (đọc reply rà soát, gộp vòng sau), `reference_style.go`, `input.go` (input từ authoring, hash từng nguồn) |
| application | `concept_desk.go` (trạng thái màn, sửa, duyệt, chốt), `concept_writer.go` (AI viết một phần hoặc lần lượt), `concept_reviewer.go`, `reference_library.go`, `purge.go` |
| ports | `AuthoringInputPort`, `LLMPort` (Chat), `PromptLibraryPort` (CR-113), `FontCatalogPort`, các kho |
| adapters | `http` (router, middleware `X-External-AI`), `postgres`, `authoring` (client HTTP), `llm` (client), `rabbitmq` (purge) |
| cmd | `cmd/visual-service/main.go` |

Danh sách font lấy từ rendering hoặc từ một cấu hình dùng chung. Tôi chưa kiểm rendering có route liệt kê font chưa; nếu chưa có thì phải hỏi Creator trước khi chọn cách.

### authoring-service
`GET /internal/v1/authoring/{id}/script-versions/latest`: trả nguyên bản kịch bản chốt mới nhất (mọi phần kèm hash). `404` khi chưa chốt. Có test Go.

### orchestrator
visual-service không giữ file nên không vào `PurgeTargets`. Cuối saga xoá, giống authoring, orchestrator gọi `DELETE /internal/v1/visual/{id}` (cố gắng một lần, lỗi thì ghi log). Có test.

### Hợp đồng HTTP công khai (qua api-gateway)
| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/visual-concept` | toàn màn: input tóm tắt, 8 phần, máy kiểm, rà soát, tiêu chí, phiên bản |
| PUT | `/v1/projects/{id}/visual-concept/refs` | `{ref_ids[]}` |
| PUT | `/v1/projects/{id}/visual-concept/model` | `{model}` |
| POST | `/v1/projects/{id}/visual-concept/generate` | AI viết các phần còn trống (lần lượt) |
| POST | `/v1/projects/{id}/visual-concept/units/{key}/generate` | `{note}`, có `X-External-AI` |
| PUT | `/v1/projects/{id}/visual-concept/units/{key}` | sửa tay `{fields}` |
| POST | `/v1/projects/{id}/visual-concept/units/{key}/approve` · `/unapprove` | |
| POST | `/v1/projects/{id}/visual-concept/approve-all` | |
| POST | `/v1/projects/{id}/visual-concept/review` | `{note}`, có `X-External-AI` |
| POST | `/v1/projects/{id}/visual-concept/findings/{fid}/apply` · `/accept` `{why}` | |
| POST | `/v1/projects/{id}/visual-concept/lock` | `{note}` |
| GET | `/v1/projects/{id}/visual-concept/versions/{n}` | tài liệu đã chốt |
| GET, POST | `/v1/reference-styles` | |
| PUT, DELETE | `/v1/reference-styles/{id}` | trả danh sách video bị ảnh hưởng |

Route nội bộ: `GET /internal/v1/visual/{id}/concept?step=…` trả phần bước đó cần từ bản chốt mới nhất (`404` khi chưa chốt).

Ghi trong `docs/contracts/visual-concept-v1.md` và một dòng trong README hợp đồng.

### web-gui
- Flow 23 bước: thanh bước, bản đồ flow, menu. Bước 9 cũ (trang "chưa phát triển") thay bằng bước 8 mới.
- Trang `/projects/:id/visual-concept` bám mockup v3: thẻ đầu, tab "Quy tắc hình ảnh" (thẻ tham chiếu và 8 phần gấp lại khi đã duyệt), tab "Rà soát và chốt". Dùng lại `AiOperationCard`, `AiModelSelect`, `UnitStatusChip`, `ExternalAiDialog`, `ExternalRedoDeepSeek`, `useDeepSeekQueue`, `ImpactConfirm`.
- Trang Cài đặt › Thư viện tham chiếu.
- `ProjectFlowContext` đánh dấu bước 8 xong khi đã chốt.

### docker compose
Thêm `visual-db` và `visual-service`, có healthcheck và biến môi trường `AUTHORING_SERVICE_URL`, `LLM_SERVICE_URL`, `RABBITMQ_URL`. api-gateway thêm `VISUAL_SERVICE_URL`.

## Prompt
Hai nhóm vai, seed nằm trong visual-service và đẩy lên llm-service theo CR-113:
- `visual_concept_<key>`: 8 vai, mỗi phần một vai;
- `visual_concept_review`: một vai.

**Nguyên văn từng prompt sẽ đưa Creator duyệt trước khi ghi vào seed.**

## Kiểm thử
- **Go (visual-service):** schema và máy kiểm từng phần, tương phản, phủ thuật ngữ, phủ giả thuyết và ngộ nhận; ghép tài liệu; `accuracy_notes`; stale theo từng nguồn; giữ `creator_fields` khi AI làm lại; đọc reply hỏng; gộp findings; điều kiện chốt; chốt v1 rồi v2; purge; HTTP với fake port; kho Postgres (bỏ qua khi thiếu `TEST_DATABASE_URL`).
- **Go (authoring):** route `script-versions/latest`.
- **Go (orchestrator):** saga xoá gọi xoá dữ liệu visual-service, lỗi không chặn saga.
- **Vitest:** trang bước 8, trang thư viện, rail 23 bước.
- Không gọi AI thật. Dọn dữ liệu test sau khi chạy.

## Chia việc
1. **CR-112a:** dựng service, DB, docker compose, route đọc của authoring, xoá dữ liệu cuối saga ở orchestrator. Proxy ở gateway thêm cùng các route công khai ở CR-112b và CR-112c.
2. **CR-112c:** thư viện tham chiếu (BE và trang Cài đặt).
3. **CR-112b:** bước 8 (AI, sửa, duyệt, rà soát, chốt, web-gui, flow 23 bước). Làm sau CR-113.

## Khác mockup (nói rõ)
- Mockup v3 mô phỏng một số chỗ; giao diện thật lấy dữ liệu thật.
- "Sửa tay" dùng form theo schema của từng phần, không dùng một ô văn bản tự do như mockup.

## Không làm
- Xuất file `visual-concept.md` (Creator: không cần).

## Chưa làm (nói rõ)
- Style frame.
- Bước 9–14 đọc bản chốt của bước 8.
- Khung đơn vị duyệt dùng chung giữa hai service.
