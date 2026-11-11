# CR-114 · Bước 9 Kế hoạch hình (Visual Planning) — giao việc

Ba phần, mỗi phần một nhánh và một worktree riêng, tạo từ `main` mới nhất bằng `scripts/worktree.sh add <nhánh>`:

| Phần | Nhánh | Phụ thuộc | Chạy song song với |
|---|---|---|---|
| CR-114a · backend, máy kiểm, rà soát (chưa có AI) | `feature/cr-114a-visual-plan-core` | thiết kế CR-114 đã vào main | CR-114c (phần mockup) |
| CR-114b · prompt và AI | `feature/cr-114b-visual-plan-ai` | CR-114a đã vào main; Creator duyệt nguyên văn prompt | CR-114c |
| CR-114c · web-gui bước 9 | `feature/cr-114c-visual-plan-page` | mockup bổ sung đã được Creator chốt; CR-114a đã vào main trước khi nối API | CR-114b |

## Đọc trước (mọi phần)
- **Thiết kế đã duyệt:** `aidlc-docs/construction/plans/cr-114-visual-planning-design.md`. Đây là nguồn chính: input, khuôn tài liệu, enum, thời lượng, luồng, máy kiểm, API, kiểm thử. Có chỗ nào mâu thuẫn với file này thì thiết kế thắng; không rõ thì dừng và hỏi Creator.
- **ADR:** `ADR-0049` (visual-service), `ADR-0047` (AI ngoài, ghi DB sau khi có kết quả), `ADR-0044` (bản chốt kịch bản), `ADR-0035` (đơn vị duyệt).
- **Hợp đồng:** `docs/contracts/visual-service-v1.md`, `script-lock-v1.md` (route `script-versions/latest`, ID câu `b{k}.{n}`, `facts` theo câu, `glossary`), `prompt-library-v1.md`, `external-ai-v1.md`, `llm-chat-v1.md`.
- **Quy tắc:** `CLAUDE.md`, `docs/code-standards-rules.md`, `docs/ux-ui-design-rules.md`.
- **Mockup:** https://claude.ai/artifact/Dh8YvLdSqzQL9L6FRsH4Y8 (đọc bằng Artifact tool, action `read`).
- **Khuôn code: bước 8 trong visual-service.** Làm theo cùng cấu trúc, không viết khung mới:
  - domain `visual_concept.go`, `concept_checks.go`, `findings.go`, `review_unit.go`, `input.go`;
  - application `concept_desk.go`, `concept_writer.go`, `concept_reviewer.go`, `concept_prompt.go`, `external_ai.go`;
  - adapters `http/concept.go`, `postgres/concept_repository.go`, `postgres/project_data_repository.go`;
  - web-gui `pages/VisualConceptStepPage.tsx`.

## Quy tắc chung cho agent
- Không code giả, không bỏ qua yêu cầu im lặng (`CLAUDE.md`). Việc nào chưa làm được thì ghi rõ trong báo cáo.
- Test không gọi AI thật; reply của AI là fake.
- Sau mỗi lần chạy test, dọn đúng dữ liệu test đã tạo và ghi lại trong báo cáo.
- Đổi code thì rebuild đúng service đã đổi bằng `scripts/worktree.sh rebuild <service>` và xác nhận healthy.
- Comment không ghi số CR.
- Chỉ commit sau khi Creator duyệt phần đó.

---

## CR-114a · Backend, máy kiểm, rà soát (chưa có AI)

### domain (`services/visual-service/internal/domain`)
- [ ] **`visual_plan.go`**
  - Kiểu VP unit đúng khuôn trong thiết kế, gồm `duration{narration_min_s, planned_s, extra_reason}`, `risk_notes[{fact_id, note}]`, `open_questions[{id, question, answer}]`, `essential_visual_info[{item, priority}]`.
  - Enum `method` gồm 11 giá trị, enum `narration_relationship` gồm 4 giá trị, enum `priority` gồm `must` và `nice`. Giá trị lạ thì báo lỗi, không đoán.
  - Nội dung của đơn vị duyệt một beat là `{units:[…]}`.
  - Tách unit tại một câu; gộp hai unit liền nhau cùng beat.
  - Mã `VP-xxx` lấy từ bộ đếm, không dùng lại.
  - Ghép tài liệu bản chốt: `source_versions`, `coverage`, `units` theo `order`, `review.comments`.
  - `fact_refs` do máy tính từ `facts` của các câu trong `script_refs`.
- [ ] **`plan_checks.go`**
  - Độ phủ: mỗi ID câu thuộc đúng một unit; `script_refs` liên tiếp và cùng beat.
  - `concept_rule_refs` tồn tại trong bản chốt bước 8.
  - `risk_notes[].fact_id` nằm trong `fact_refs`.
  - Dữ kiện loại `hyp` hoặc `myth` phải có `risk_note`.
  - `on_screen_text` không dùng cách gọi trong `glossary.avoid` hay `do_not_use`.
  - Thời lượng:
    - `narration_min_s` = số chữ × tốc độ đọc của dự án, với tốc độ đọc = tổng chữ của kịch bản chốt ÷ thời lượng brief.
    - `planned_s` < `narration_min_s` là lỗi chặn duyệt.
    - `planned_s` > `narration_min_s` mà không có `extra_reason` là lỗi chặn duyệt.
    - So với beat-sheet (Creator chốt 2026-11-11): **không có ngưỡng**. Mỗi beat luôn trả hai số và độ lệch (tổng `planned_s` và thời lượng beat-sheet). Chỉ cảnh báo khi tổng `planned_s` của cả video vượt thời lượng trong creative brief. Cảnh báo không chặn chốt.
- [ ] **`plan_input.go`**
  - Input lấy từ bản chốt kịch bản và bản chốt bước 8 mới nhất.
  - Hash theo beat (`script:{k}`, `handoff:{k}`), cộng hash cho brief, facts, glossary, concept.
  - Beat nào đổi hash thì beat đó stale, kèm danh sách nguồn đã đổi.
- [ ] Findings rà soát: dùng lại `findings.go` nếu được. Nếu cần tách phần chung khỏi bước 8 thì làm sao cho không đổi hành vi bước 8, và test bước 8 vẫn qua.

### Dữ liệu (`adapters/postgres`)
- [ ] Bảng `visual_plan_versions(project_id, version, script_version, concept_version, document JSONB, note, locked_at)`, chỉ thêm dòng, trùng phiên bản trả 409.
- [ ] Bảng `visual_plan_seq(project_id, next)` cấp mã VP trong giao dịch.
- Model riêng của video (`project_visual.plan_model`, `PUT …/visual-plan/model`) không làm ở CR-114a: chưa có AI để chạy và bước `visual_plan` chưa có ở llm-service. Chuyển sang CR-114b.
- [ ] `review_units` dùng `kind` `visual_plan` (`unit_key = b{k}`) và `visual_plan_review` (`unit_key = main`).
- [ ] Purge xoá thêm hai bảng mới. Test purge.

### application
- [ ] **`plan_desk.go`**
  - Trạng thái toàn màn: input tóm tắt, beat, unit, độ phủ, máy kiểm, rà soát, điều kiện chốt và lý do chưa chốt được, phiên bản.
  - Các thao tác: sửa tay (ghi `creator_fields`, unit về `pending`), tách, gộp, trả lời câu hỏi, duyệt, bỏ duyệt, duyệt tất cả, chốt v1 và v2.
  - Chưa chốt bước 8 thì trả `409 concept_not_locked`. Chưa chốt kịch bản thì trả `409 script_not_locked`.
  - Máy kiểm cứng (lỗi của unit, chặn Duyệt, không chấp nhận được): schema, enum, câu có thật và liền nhau, ref concept, risk note khớp dữ kiện, thời lượng. Máy kiểm mềm của cả kế hoạch (chặn chốt, chấp nhận kèm lý do được): `m-plan-risk`, `m-plan-avoid`. Chi tiết: `docs/contracts/visual-plan-v1.md`.
- [ ] **Rà soát:**
  - Áp dụng gợi ý: ghi `fix` vào trường, unit về `pending`.
  - Chấp nhận kèm lý do.
  - Gộp với vòng rà soát trước.
  - Có dấu vân tay để quyết định cũ hết hiệu lực khi nội dung đổi.
  - Phần gọi AI (`visual_plan_review`) để cho CR-114b. Ở CR-114a, route `POST /review` chưa có: **không tạo route trả dữ liệu giả.**
- [ ] `ConceptDesk.LatestDocument` không đổi.

### HTTP (`adapters/http/plan.go`)
- [ ] Mọi route trong bảng "Hợp đồng HTTP công khai" của thiết kế, **trừ** `generate`, `beats/{k}/generate`, `review` và `model` (thuộc CR-114b).
- [ ] Route nội bộ `GET /internal/v1/visual/{id}/plan`.
- [ ] api-gateway proxy `/v1/projects/:id/visual-plan…` sang visual-service.

### Hợp đồng
- [ ] Viết `docs/contracts/visual-plan-v1.md` và thêm một dòng vào `docs/contracts/README.md`.

### Test (Go)
- [ ] Schema và enum; độ phủ (thiếu câu, trùng câu, không liên tiếp, khác beat).
- [ ] Ref concept không tồn tại; rủi ro `hyp` và `myth`; glossary `avoid`.
- [ ] Ba trường hợp thời lượng.
- [ ] Tách, gộp; mã VP không dùng lại.
- [ ] Stale chỉ ở beat có hash đổi; giữ `creator_fields`.
- [ ] Gộp findings; điều kiện chốt; chốt v1 rồi v2; purge.
- [ ] HTTP với fake port; kho Postgres (bỏ qua khi thiếu `TEST_DATABASE_URL`).

### Xong khi
- `go test ./...` của visual-service qua; test bước 8 không đổi kết quả.
- visual-service rebuild và healthy.
- Báo cáo ghi rõ những gì để lại cho CR-114b.

---

## CR-114b · Prompt và AI

### Bước 0: duyệt prompt (bắt buộc, trước mọi code)
- [ ] Soạn nguyên văn hai prompt `visual_plan_beat` và `visual_plan_review` vào `aidlc-docs/construction/plans/cr-114b-prompts-draft.md`, theo khuôn `cr-112b-prompts-draft.md`.
- [ ] Prompt không được chứa số cứng hay ví dụ riêng của một chủ đề (memory của Creator).
- [ ] Prompt `visual_plan_beat` phải yêu cầu đủ trường theo schema: 11 `method`; `planned_s` ≥ mức tối thiểu (mức tối thiểu được truyền vào prompt), có `extra_reason` khi dài hơn; `concept_rule_refs` chỉ lấy từ danh sách ID được truyền vào.
- [ ] **Dừng, đưa Creator xem nguyên văn, chờ "ok"** rồi mới ghi seed.

### Code
- [ ] Seed hai role trong `domain/prompt_seeds.go`, đẩy lên llm-service với `owner = "visual"` như bước 8.
- [ ] **`plan_writer.go`**
  - `POST /beats/{k}/generate`: một lượt `Chat`; input gồm beat đó, beat trước và sau, bản chốt bước 8, glossary, facts, `narration_min_s` của từng câu.
  - Kiểm reply bằng domain của CR-114a; reply hỏng thì báo lỗi, không ghi.
  - Giữ `creator_fields`; chỉ ghi DB sau khi có kết quả.
  - `POST /generate`: viết lần lượt các beat còn trống theo thứ tự.
- [ ] **`plan_reviewer.go`:** `POST /review`, một lượt, prompt `visual_plan_review`, gộp findings bằng code của CR-114a.
- [ ] `X-External-AI` cho `beats/{k}/generate` và `review`, dùng lại `external_ai.go`.
- [ ] Model: cột `project_visual.plan_model` (`ALTER TABLE … ADD COLUMN IF NOT EXISTS`), `PUT /v1/projects/{id}/visual-plan/model` (gateway proxy kèm), và `model`, `default_model`, `models` trong trạng thái màn; để trống thì dùng model mặc định của bước `visual_plan` (thêm `"visual_plan"` vào `STEPS` trong `services/llm-service/app/modelcatalog.py`, sửa test).
- [ ] api-gateway: timeout dài cho các route AI mới.

### Test
- [ ] Go với fake LLM: reply đúng; reply hỏng; giữ `creator_fields`; viết lần lượt; rà soát gộp vòng.
- [ ] pytest llm-service cho `STEPS`.

### Xong khi
- Test qua; rebuild visual-service, llm-service, api-gateway và tất cả healthy.
- Không chạy model thật; muốn chạy thật thì hỏi Creator trước (chỉ Ollama).

---

## CR-114c · Web-gui bước 9

### Bước 0: bổ sung mockup (bắt buộc, trước mọi code)
- [ ] Thêm vào artifact https://claude.ai/artifact/Dh8YvLdSqzQL9L6FRsH4Y8:
  - tab "Rà soát và chốt" như bước 8 (vấn đề, áp dụng, chấp nhận kèm lý do, rà soát lại, điều kiện chốt, phiên bản);
  - ô thời lượng trên mỗi unit (tối thiểu, dự kiến, lý do);
  - đổi `SC-xxx` thành ID câu `b{k}.{n}`;
  - modal Sửa, Tách, Gộp, Trả lời câu hỏi, Làm lại kèm ghi chú.
- [ ] **Dừng, đưa Creator xem, chờ chốt** rồi mới code.

### Code (`services/web-gui`)
- [ ] Trang `/projects/:id/visual-plan`, thay trang "chưa phát triển" của bước 9. `AppShell wide`.
- [ ] Bố cục theo mockup đã chốt và `docs/ux-ui-design-rules.md`:
  - thứ tự 1 → 4 rồi Chốt;
  - modal cho mọi việc sửa;
  - màu nút theo tầm ảnh hưởng (Làm lại màu cam khi bước sau đã dùng, tím khi chưa);
  - `CopyBox` cho mô tả cách hiển thị.
- [ ] Dùng lại `AiOperationCard`, `AiModelSelect`, `UnitStatusChip`, `ExternalAiDialog`, `ExternalRedoDeepSeek`, `useDeepSeekQueue`, `ImpactConfirm` và dialog shell dùng chung.
- [ ] `ProjectFlowContext` đánh dấu bước 9 xong khi đã chốt.
- [ ] Nút AI chỉ hiện khi CR-114b đã vào main. Nếu chưa thì không hiện nút, **không làm nút giả.**

### Test
- [ ] Vitest: bố cục, độ phủ và nút tới câu chưa phủ, bộ lọc, nút Chốt bị khoá kèm lý do, modal, thời lượng.
- [ ] Kiểm bên cạnh trang bước 8 ở cùng cỡ cửa sổ (quy tắc 7).

### Xong khi
- Vitest qua; rebuild web-gui và healthy; báo Creator để xem trực tiếp.
