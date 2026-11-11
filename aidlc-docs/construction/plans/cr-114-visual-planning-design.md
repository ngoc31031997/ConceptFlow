# CR-114 · Bước 9 Kế hoạch hình (Visual Planning) — thiết kế

Nhánh `feature/cr-114-visual-planning` (worktree riêng). Nguồn:
- spec bước 9 của Creator (session 2026-11-10/11);
- mockup và review input/output https://claude.ai/artifact/Dh8YvLdSqzQL9L6FRsH4Y8 (v2, Creator chốt 2026-11-11);
- ADR-0049 (visual-service sở hữu bước 8–14), ADR-0047 (AI ngoài, ghi DB sau khi có kết quả), ADR-0044 (bản chốt kịch bản).

Khuôn: bước 8 (CR-112b), đã có trên main.

## Creator đã chốt
1. Bước 9 chuyển kịch bản thành các **đơn vị hình** (unit), mỗi unit có một mục tiêu truyền đạt. Bước Phân rã kịch bản không còn (ADR-0049).
2. Input: script-final, beat-sheet, research-dossier, visual-concept, creative-brief. Glossary đã nằm trong script-final (phần `glossary` của bản chốt bước 7), không thêm input riêng.
3. Output `visual-plan.json` theo khuôn của Creator, cộng các bổ sung trong artifact: `source_versions`, `coverage`, `order`, `duration` (thay `est_duration_s`, xem mục Thời lượng), `on_screen_text`, `fact_refs`, `essential_visual_info` có `priority`, `concept_rule_refs` trỏ ID cụ thể, `risk_notes` và `open_questions` dạng object.
4. Giao diện theo mockup: 1 Đầu vào đã chốt · 2 Chạy AI · 3 Độ phủ kịch bản và bộ lọc · 4 Kế hoạch hình gom theo beat (trái lời thoại, phải hình) · Chốt.
5. (2026-11-11, trả lời Q1–Q4)
   - Q1: đơn vị duyệt ở mức beat, nút Duyệt ở từng VP unit.
   - Q2: thời lượng giọng đọc chỉ là **mức tối thiểu**; một số unit cần dài hơn giọng đọc.
   - Q3: giữ danh sách `method` hiện tại; góp ý thêm thì nêu với Creator.
   - Q4: **có** vòng AI rà soát cả kế hoạch như bước 8.

## Thời lượng
- `narration_min_s`: máy tính, AI và Creator không sửa. Chưa có file giọng thì lấy theo số chữ của các câu trong unit nhân với tốc độ đọc của **chính dự án** = tổng số chữ của kịch bản chốt ÷ thời lượng trong creative brief. Không ghi cứng tốc độ đọc trong code. Khi bước Giọng có file thật cho các câu đó (sau này), dùng độ dài file thật.
- `planned_s`: AI đề xuất, Creator sửa được; phải ≥ `narration_min_s`. Để trống nghĩa là bằng mức tối thiểu.
- `extra_reason`: bắt buộc khi `planned_s` > `narration_min_s` (vì sao cần giữ hình lâu hơn lời: để khán giả kịp nhìn, chuyển cảnh, nhịp lắng…).
- Máy kiểm: `planned_s` < mức tối thiểu là lỗi (chặn duyệt); mỗi beat luôn hiện tổng `planned_s` cạnh thời lượng beat-sheet và độ lệch, không có ngưỡng; chỉ cảnh báo (không chặn) khi tổng `planned_s` cả video vượt thời lượng creative brief.

## Khác so với artifact (theo dữ liệu thật trong code)
- Kịch bản **không có cảnh `SC-xxx`**. Mỗi câu thoại có ID cố định `<beat>.<n>` (ví dụ `b2.3`, `ScriptLine` trong authoring). `script_refs` là danh sách ID câu: `["b2.1","b2.2","b2.3"]`. Mockup ghi `SC-002 #s1` sẽ đổi thành `b2.1–b2.3`.
- Mỗi câu đã gắn `facts[]`. Vì thế **`fact_refs` do máy tính** từ các câu thuộc unit (AI không khai). `risk_notes[].fact_id` phải nằm trong `fact_refs`.
- `concept_rule_refs` trỏ vào ID có thật trong bản chốt bước 8: `color_system.palette.<token>`, `representation_rules.scientific_concepts.<concept_id>`, `representation_rules.characters.items.<concept_id>`, `visual_language.<nhóm>[<i>]`, `motion_language.preferred_transitions[<i>]`. Máy kiểm ref có tồn tại.

## Phụ thuộc và trùng lặp
- Cần bước 8 đã chốt (`visual_concept_versions`). Chưa chốt thì `409 concept_not_locked`; màn báo "Chốt Ngôn ngữ hình ảnh ở bước 8 trước".
- Cần kịch bản chốt bước 7 (route `script-versions/latest` đã có).
- **Đang mở:** `feature/cr-112-visual-concept`, `cr-112c`, `cr-112d` (worktree vẫn còn). CR-112b đã vào main. CR-114 chạm cùng visual-service (domain, http, postgres, web-gui flow). Không đổi bảng hay route của bước 8, chỉ thêm. Merge cần pull main mới nhất.
- Không chạm authoring, orchestrator (purge đã gọi `DELETE /internal/v1/visual/{id}`; chỉ cần xoá thêm bảng mới).

## Input
| Input | Nguồn | Phần dùng |
|---|---|---|
| script-final | authoring `script-versions/latest` | `version`; `script:{k}` (câu, ID, `facts`); `handoff:{k}`; `glossary` (`term`, `say`, `draw`, `avoid`) |
| beat-sheet | trong bản chốt (`beat:{k}`, `order`) | key, chức năng, tiêu đề, mục tiêu, thời lượng |
| research-dossier | trong bản chốt (`facts`) | mã, nội dung, loại (`fact`, `hyp`, `unk`, `myth`) |
| creative-brief | trong bản chốt (`brief`) | khán giả, độ khó, thời lượng, giọng |
| visual-concept | `visual_concept_versions` (DB của chính visual-service) | bản chốt mới nhất: quy tắc, palette, khái niệm, chữ, `do_not_use`, `accuracy_notes` |

Hash từng nguồn như bước 8. Riêng kịch bản, hash **theo từng beat** (`script:{k}`, `handoff:{k}`), để đổi beat 3 chỉ làm stale các unit của beat 3.

## Tài liệu bản chốt (`document`)
```jsonc
{
  "schema_version": "1.0", "project_id": "…", "document_id": "visual-plan-v{n}",
  "visual_concept_id": "visual-concept-v{m}", "status": "locked",
  "+version": 1, "+locked_at": "…",
  "+source_versions": { "script": 2, "visual_concept": 1 },   // brief, beat, dossier nằm trong bản chốt kịch bản
  "+coverage": { "lines_total": 45, "lines_covered": 45, "uncovered": [] },
  "units": [{
    "unit_id": "VP-002", "+order": 2, "beat_id": "b2",
    "script_refs": ["b2.1", "b2.2", "b2.3"],
    "+duration": { "narration_min_s": 11, "planned_s": 14, "extra_reason": "giữ hình hai góc nhìn để khán giả kịp so sánh" },
    "narration_summary": "…", "communication_goal": "…",
    "essential_visual_info": [{ "item": "…", "priority": "must|nice" }],
    "+on_screen_text": ["…"],
    "visual_approach": { "method": "<enum>", "description": "…", "concept_rule_refs": ["color_system.palette.observer_a"] },
    "narration_relationship": "<enum>",
    "+fact_refs": ["F08"],
    "risk_notes": [{ "fact_id": "F08", "note": "…" }],
    "open_questions": [{ "id": "Q1", "question": "…", "answer": "…" }],
    "status": "approved"
  }]
}
```

Enum (trong contract, AI không tự thêm):
- `visual_approach.method`: `spatial_composition`, `split_perspective_comparison`, `diagram`, `analogy`, `process_sequence`, `scale_comparison`, `data_visual`, `character_action`, `text_emphasis`, `timeline` (dòng thời gian, trình tự theo năm), `cross_section` (mặt cắt, cấu trúc bên trong). Creator đã duyệt (11 giá trị).
- `narration_relationship`: `direct_illustration`, `contrast`, `complement`, `metaphor`.
- `status` unit: `pending` (cần xem lại), `approved`, `running`. `needs_review` trong JSON gốc = `pending`; `ready` trong JSON gốc bỏ, vì "sẵn sàng" chính là đã duyệt.

## Đơn vị duyệt và luồng nghiệp vụ
Đơn vị duyệt là **một beat** (`kind = visual_plan`, `unit_key = b{k}`, `content` = các VP unit của beat đó). Lý do: AI viết theo beat (mỗi lượt đọc lời thoại và ghi chú bàn giao của beat), stale tính theo beat, và dùng lại nguyên khung `review_units` của bước 8. Trên màn, nút Duyệt vẫn nằm ở từng VP unit như mockup; beat chỉ "đã duyệt" khi mọi VP unit của nó đã duyệt.

1. **Mở màn** (`GET /visual-plan`): đọc input; tính hash; đánh dấu beat stale; tính máy kiểm và độ phủ; trả toàn màn.
2. **AI lập kế hoạch một beat** (`POST /beats/{k}/generate`, `{note}`, có `X-External-AI`): một lượt `Chat`, prompt `visual_plan_beat`; đọc input, bản chốt bước 8, beat trước và sau (để nối mạch); kiểm reply theo schema và máy kiểm; giữ VP unit Creator đã sửa tay (`creator_fields`); ghi DB sau khi có kết quả.
3. **AI lập cho mọi beat còn trống** (`POST /generate`): lần lượt theo thứ tự beat. DeepSeek: xếp từng beat vào hàng đợi như bước 8.
4. **Sửa tay một VP unit** (modal, `PUT /units/{vp}`): form theo schema; unit về `pending`.
5. **Tách / gộp** (modal): tách một unit thành hai tại một câu; gộp hai unit liền nhau **trong cùng beat**. Mã `VP-xxx` không dùng lại (giống `R01` của thư viện tham chiếu).
6. **Trả lời câu hỏi mở** (modal, `PUT /units/{vp}/questions/{qid}`).
7. **Duyệt / bỏ duyệt** từng VP unit; "Duyệt tất cả còn lại". Không duyệt được khi còn câu hỏi chưa trả lời, schema chưa hợp lệ hoặc beat stale.
8. **Rà soát** (`POST /review`, `{note}`, có `X-External-AI`; một lượt AI, prompt `visual_plan_review`), giống bước 8:
   - AI đọc cả kế hoạch cùng input, trả `findings[{group, unit_id, field, text, detail, fix}]`. Nhóm gợi ý: mạch hình giữa các unit, mục tiêu truyền đạt chưa rõ, sai khoa học so với dossier, trái quy tắc bước 8, thời lượng giữ hình bất hợp lý, chữ trên hình quá nhiều;
   - "Rà soát lại" gộp với vấn đề cũ: vấn đề đã hết thì đánh dấu đã sửa, vấn đề mới thì thêm vào;
   - xử lý từng vấn đề: **Áp dụng gợi ý** (ghi `fix` vào trường, trường thành của Creator, unit về `pending`), **Chấp nhận kèm lý do**, **Mở unit đó**.
   - Lưu ở `review_units` với `kind = visual_plan_review`, `unit_key = main`, có dấu vân tay như bước 8.
9. **Chốt** (`POST /lock`). Điều kiện: phủ 100% câu; không câu nào thuộc hai unit; mọi unit đã duyệt; không beat nào stale; mọi câu hỏi mở đã trả lời; đã rà soát ít nhất một lần và không còn vấn đề mở; máy kiểm cả bộ đạt hoặc đã chấp nhận kèm lý do. Từ v2 có hộp xác nhận nêu unit đã đổi. Vấn đề đã chấp nhận ghi vào `review.comments` của tài liệu.

**Máy kiểm (không lưu, tính mỗi lần đọc):**
- độ phủ: mỗi ID câu của bản chốt kịch bản thuộc đúng một unit; `script_refs` liên tiếp và cùng beat;
- `concept_rule_refs` tồn tại trong bản chốt bước 8;
- `risk_notes[].fact_id` ∈ `fact_refs`; dữ kiện loại `hyp` hoặc `myth` trong `fact_refs` phải có `risk_note`;
- `on_screen_text` không chứa cách gọi nằm trong `glossary.avoid` hay `do_not_use` của bước 8;
- thời lượng theo mục "Thời lượng".

## Dữ liệu (`visual-db`)
| Bảng | Khoá | Ghi chú |
|---|---|---|
| `review_units` (đã có) | `(project_id, 'visual_plan', b{k})` | `content` = `{units:[…]}` |
| `visual_plan_versions` (mới) | `(project_id, version)` | `script_version`, `concept_version`, `document JSONB`, `note`, `locked_at` |
| `project_visual` (đã có) | thêm cột `plan_model` | `''` = mặc định ở llm-service |
| `visual_plan_seq` (mới) | `project_id` | số VP tiếp theo, không dùng lại |

Purge xoá thêm hai bảng mới.

## Thiết kế mức thấp
### visual-service
| Lớp | Tệp mới |
|---|---|
| domain | `visual_plan.go` (schema unit, enum, ghép tài liệu, tách/gộp), `plan_checks.go` (độ phủ, ref concept, rủi ro, glossary, thời lượng), `plan_input.go` (input theo beat, hash) |
| application | `plan_desk.go` (màn, sửa, tách, gộp, trả lời, duyệt, chốt), `plan_writer.go` (AI theo beat, lần lượt) |
| adapters | `http/plan.go`; `postgres` thêm kho phiên bản; `ConceptDesk.LatestDocument` nhận `step="visual_plan"` cho bước 10 sau này (chưa làm ở CR này) |

### Hợp đồng HTTP công khai (qua api-gateway)
| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/visual-plan` | toàn màn |
| PUT | `…/visual-plan/model` | `{model}` |
| POST | `…/visual-plan/generate` | AI cho mọi beat còn trống |
| POST | `…/visual-plan/beats/{k}/generate` | `{note}`, `X-External-AI` |
| PUT | `…/visual-plan/units/{vp}` | sửa tay |
| POST | `…/visual-plan/units/{vp}/split` | `{at_line}` |
| POST | `…/visual-plan/units/{vp}/merge` | `{with}` |
| PUT | `…/visual-plan/units/{vp}/questions/{qid}` | `{answer}` |
| POST | `…/visual-plan/units/{vp}/approve` · `/unapprove` | |
| POST | `…/visual-plan/approve-all` | |
| POST | `…/visual-plan/review` | `{note}`, `X-External-AI` |
| POST | `…/visual-plan/findings/{fid}/apply` · `/accept` `{why}` | |
| POST | `…/visual-plan/lock` | `{note}` |
| GET | `…/visual-plan/versions/{n}` | tài liệu đã chốt |

Nội bộ: `GET /internal/v1/visual/{id}/plan` (bản chốt mới nhất, 404 khi chưa có) cho bước 10. Ghi trong `docs/contracts/visual-plan-v1.md` và README hợp đồng.

### web-gui
Trang `/projects/:id/visual-plan` theo mockup; thay trang "chưa phát triển" của bước 9; `ProjectFlowContext` đánh dấu bước 9 xong khi đã chốt. Dùng lại `AiOperationCard`, `AiModelSelect`, `UnitStatusChip`, `ExternalAiDialog`, `ImpactConfirm`, `CopyBox`, dialog shell dùng chung.

### Prompt
Hai vai `visual_plan_beat` và `visual_plan_review`, seed trong visual-service và đẩy lên llm-service như bước 8. **Nguyên văn đưa Creator duyệt trước khi ghi seed.**

## Kiểm thử
- Go: schema, enum, độ phủ, ref concept, rủi ro, glossary, tách/gộp, mã VP không dùng lại, stale theo beat, giữ `creator_fields`, reply hỏng, điều kiện chốt, v1 rồi v2, purge, HTTP với fake port, kho Postgres.
- Vitest: trang bước 9 (bố cục 4 phần, độ phủ, lọc, khoá nút Chốt có lý do).
- Không gọi AI thật. Dọn dữ liệu test sau khi chạy.

## Chia việc
1. **CR-114a:** domain, máy kiểm, DB, route, chốt (chưa có AI). Kiểm bằng test.
2. **CR-114b:** prompt `visual_plan_beat` và `visual_plan_review` (sau khi Creator duyệt nguyên văn), AI theo beat, rà soát, DeepSeek.
3. **CR-114c:** web-gui bước 9: tab "Kế hoạch hình" và tab "Rà soát và chốt" như bước 8. Mockup cần thêm tab rà soát và ô thời lượng (tối thiểu / dự kiến / lý do) trước khi code phần này.


## Chưa làm (nói rõ)
- Bước 10 đọc bản chốt bước 9.
- Xuất file `visual-plan.json` ra ngoài app.
