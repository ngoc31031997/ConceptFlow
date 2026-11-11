# Kế hoạch hình: API của bước 9 (`/v1`)

Bước 9 (visual-service, ADR-0049) đọc kịch bản đã chốt ở bước 7 và ngôn ngữ hình ảnh đã chốt ở bước 8, giữ các **unit hình** của từng beat để Creator sửa, tách, gộp, trả lời câu hỏi, duyệt, xử lý vấn đề rà soát và chốt một phiên bản bất biến cho các bước hình ảnh sau đọc. Mọi đường dẫn công khai đi qua api-gateway. Mã: `services/visual-service/internal/domain/` (`visual_plan.go`, `plan_input.go`, `plan_checks.go`, `plan_document.go`), `internal/application/plan_desk.go`, `plan_actions.go`, `internal/adapters/http/plan.go`, `internal/adapters/postgres/plan_repository.go`.

Phần gọi AI (lập kế hoạch từng beat, rà soát, model riêng của video) chưa có trong phiên bản này: route `generate`, `beats/{k}/generate`, `review` và `model` thuộc CR-114b.

## Đầu vào

| Nguồn | Lấy từ | Dùng cho |
|---|---|---|
| Kịch bản chốt | `GET /internal/v1/authoring/{id}/script-versions/latest` ([`script-lock-v1.md`](script-lock-v1.md)): `script:{k}` (câu `b{k}.{n}` kèm `facts`), `handoff:{k}`, `beat:{k}`, `order`, `brief`, `facts`, `glossary` | câu lời thoại, dữ kiện, thuật ngữ, thời lượng |
| Ngôn ngữ hình ảnh chốt | bản mới nhất của `visual_concept_versions` ([`visual-concept-v1.md`](visual-concept-v1.md)) | các quy tắc mà unit trỏ tới |

Authoring `404` → `409 {"code":"script_not_locked"}`. Chưa chốt bước 8 → `409 {"error":"Chốt Ngôn ngữ hình ảnh ở bước 8 trước","code":"concept_not_locked"}`.

**Tốc độ đọc** của video = tổng số chữ của mọi câu trong kịch bản chốt chia cho `brief.duration_seconds`; không có số cứng nào trong code. Brief chưa có thời lượng hoặc kịch bản chưa có chữ thì không tính được: mọi unit có `problems` nêu lý do và không duyệt được.

**Stale:** mỗi beat lưu `input_hashes` lúc được ghi. Nguồn gồm `beat` (hash của `beat:{k}`, `script:{k}`, `handoff:{k}` của chính beat đó), `brief`, `facts`, `glossary`, `concept` (cả tài liệu chốt bước 8). Đổi lời thoại một beat chỉ làm beat đó stale; đổi bảng thuật ngữ hay ngôn ngữ hình ảnh làm mọi beat stale. Beat stale: `Duyệt` trả `409 beat_stale`, "Duyệt tất cả" bỏ qua beat đó, và không chốt được cho tới khi beat được lập lại.

## Unit và đơn vị duyệt

Đơn vị lưu (`review_units`, `kind = visual_plan`) là **một beat**: `unit_key` là khoá beat (`b1`), `content` là `{"units":[…]}` theo thứ tự câu. Trạng thái duyệt nằm ở từng unit (`status` ∈ `pending, approved`); trạng thái của dòng beat là `approved` khi beat có unit và mọi unit đã duyệt. `creator_fields` của dòng beat là đường dẫn `VP-002.communication_goal`.

Unit (`VP-001`…, mã cấp theo `visual_plan_seq`, không dùng lại, kể cả khi một lần gộp bỏ mất mã):

| Trường | Ghi chú |
|---|---|
| `unit_id` | không sửa được |
| `script_refs` | ID câu liền nhau của đúng một beat; đổi bằng tách/gộp |
| `narration_summary`, `communication_goal` | văn bản, bắt buộc |
| `essential_visual_info[{item, priority}]` | `priority` ∈ `must, nice`; ít nhất một dòng |
| `on_screen_text[]` | chữ hiện trên hình |
| `visual_approach{method, description, concept_rule_refs[]}` | `method` ∈ `spatial_composition, split_perspective_comparison, diagram, analogy, process_sequence, scale_comparison, data_visual, character_action, text_emphasis, timeline, cross_section` |
| `narration_relationship` | `direct_illustration, contrast, complement, metaphor` |
| `risk_notes[{fact_id, note}]` | `fact_id` phải nằm trong `fact_refs` |
| `open_questions[{id, question, answer}]` | `answer` rỗng = chưa trả lời |
| `duration{planned_s, extra_reason}` | xem "Thời lượng" |
| `status` | không sửa tay; Duyệt/Bỏ duyệt |

`concept_rule_refs` là mã quy tắc trong bản chốt bước 8: `color_system.palette.<token>`, `visual_language.<nhóm>[<i>]` (nhóm: `shape_rules, line_rules, lighting_rules, depth_rules, surface_rules, layout_rules, composition_rules`), `representation_rules.characters.items.<concept_id>`, `representation_rules.objects.<concept_id>`, `representation_rules.invisible.<concept_id>`, `representation_rules.scientific_concepts.<concept_id>`, `motion_language.preferred_transitions[<i>]`, `creative_direction.principles[<i>]`.

`fact_refs` không lưu và AI không khai: máy lấy hợp các `facts` của những câu trong `script_refs`.

JSON đọc chặt: khoá lạ hoặc sai kiểu khi sửa tay là `400`.

## Kiểm

**Của từng unit** (`problems[]`, chặn Duyệt): mã `VP-nnn`; `script_refs` có thật, liền nhau, cùng beat; tóm tắt, mục tiêu, mô tả cách hiển thị không rỗng; các giá trị thuộc enum; mọi `concept_rule_refs` có trong bản chốt bước 8; `risk_notes` nói về dữ kiện của chính unit; câu hỏi có mã duy nhất; thời lượng (dưới). Câu hỏi chưa trả lời cũng chặn Duyệt (`Câu hỏi Q1 chưa trả lời`).

**Của cả kế hoạch** (`checks[]`, chặn chốt, chấp nhận kèm lý do được; lý do hết hiệu lực khi `detail` đổi):
- `m-plan-risk`: dữ kiện loại giả thuyết (`hypothesis`) hoặc ngộ nhận (`misconception`) trong `fact_refs` của unit phải có `risk_note`;
- `m-plan-avoid`: chữ trên hình không chứa cách nói trong cột `avoid` của bảng thuật ngữ (tách ở `;` và xuống dòng, không phân biệt hoa thường). Mục `do_not_use` của bước 8 là mô tả, không so khớp chữ; việc đó thuộc vòng AI rà soát.

**Độ phủ** (`coverage`): mỗi câu của kịch bản chốt thuộc đúng một unit. `uncovered[]` là câu không thuộc unit nào, `duplicated[]` là câu thuộc nhiều unit. Chưa phủ 100% thì không chốt được, và không chấp nhận được.

## Thời lượng

- `narration_min_s` (trong unit view và tài liệu chốt): máy tính, số chữ các câu của unit × tốc độ đọc, làm tròn lên một phần mười giây; không ai sửa.
- `duration.planned_s`: 0 hoặc để trống = bằng `narration_min_s`. Nhỏ hơn mức tối thiểu là lỗi chặn Duyệt. Lớn hơn thì bắt buộc có `duration.extra_reason`.
- `effective_planned_s` trong unit view: `planned_s` nếu có, không thì `narration_min_s`.
- `timing.beats.{k}`: tổng `narration_min_s`, tổng `planned_s`, `beat_sheet_s` và `delta_s = planned_s − beat_sheet_s`. **Không có ngưỡng.** `timing.over_brief` bật khi tổng `planned_s` của mọi unit vượt `brief_s`; chỉ là cảnh báo, không chặn chốt.

## Rà soát

Kết quả rà soát lưu ở `review_units` `kind = visual_plan_review`, `unit_key = main`, cùng khuôn với bước 8 (`findings[{id, group, unit_key, field, text, detail, fix, state, why}]`, `rounds`, `notes`, quyết định cho máy kiểm). `unit_key` của một vấn đề là mã unit (`VP-002`), `field` là trường cấp ngoài của unit. Nhóm (`group`): `flow` Mạch hình giữa các unit, `goal` Mục tiêu truyền đạt, `accuracy` Độ chính xác khoa học, `concept` Trái quy tắc ngôn ngữ hình ảnh, `timing` Thời lượng giữ hình, `text` Chữ trên hình. Lượt gọi AI để tạo vấn đề thuộc CR-114b; xử lý vấn đề có ở đây.

## API công khai

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/visual-plan` | Toàn màn: `input`, `beats[]` (mỗi beat: `stale`, `stale_sources[]`, `timing`, `units[]`), `units_total`, `units_approved`, `coverage`, `timing`, `checks[]`, `review`, `blockers[]`, `can_lock`, `versions[]`, `locked_version`, `next_version`, `draft[]` |
| PUT | `…/units/{vp}` | Sửa tay `{"fields":{…}}`; chỉ các trường: `narration_summary, communication_goal, essential_visual_info, on_screen_text, visual_approach, narration_relationship, risk_notes, open_questions, duration`. Trường sửa thành của Creator, unit về `pending`. Sai trường hoặc sai kiểu: `400` |
| PUT | `…/units/{vp}/questions/{qid}` | `{"answer":"…"}` trả lời câu hỏi mở; unit về `pending`. Câu hỏi lạ `404`, câu trả lời rỗng `400` |
| POST | `…/units/{vp}/split` | `{"at_line":"b1.2"}` cắt trước câu đó; unit mới nhận mã mới, câu hỏi mở ở lại unit đầu. `400` khi câu không thuộc unit hoặc là câu đầu |
| POST | `…/units/{vp}/merge` | `{"with":"VP-002"}` gộp unit liền sau vào unit này (cùng beat). Khác đi `400` |
| POST | `…/units/{vp}/approve` · `/unapprove` | Duyệt cần: beat không stale (`409 beat_stale`), unit hết `problems`, mọi câu hỏi đã trả lời (`409`) |
| POST | `…/approve-all` | Duyệt mọi unit đạt trong beat không stale |
| POST | `…/findings/{fid}/apply` | Ghi `fix` của vấn đề vào `field` của unit; chỉ trường văn bản hoặc danh sách văn bản cấp ngoài; unit về `pending` |
| POST | `…/findings/{fid}/accept` | `{"why":"…"}` chấp nhận vấn đề, hoặc máy kiểm `m-plan-*` đang không đạt |
| POST | `…/lock` | `{"note":"…"}` chốt phiên bản. `409` kèm `blockers` khi chưa đủ điều kiện; `409` khi không đổi gì so với bản chốt |
| GET | `…/versions/{n}` | Phiên bản đã chốt kèm `document` |

Điều kiện chốt: có unit; độ phủ đầy đủ; mọi unit đã duyệt và hết `problems`; không beat nào stale; **đã rà soát ít nhất một lần** và không còn vấn đề mở; mọi máy kiểm của cả kế hoạch đạt hoặc đã chấp nhận kèm lý do; có thay đổi so với bản chốt trước.

## API nội bộ

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/internal/v1/visual/{id}/plan` | Tài liệu của bản chốt mới nhất. `404` khi chưa chốt |

## Tài liệu bản chốt (`visual_plan_versions.document`)

```jsonc
{
  "schema_version": "1.0", "project_id": "…", "document_id": "visual-plan-v1",
  "visual_concept_id": "visual-concept-v1", "status": "locked", "version": 1, "locked_at": "…",
  "source_versions": { "script": 2, "visual_concept": 1 },
  "coverage": { "lines_total": 45, "lines_covered": 45, "uncovered": [], "duplicated": [] },
  "units": [{
    "unit_id": "VP-002", "order": 2, "beat_id": "b2", "script_refs": ["b2.1", "b2.2"],
    "duration": { "narration_min_s": 11.0, "planned_s": 14.0, "extra_reason": "…" },
    "narration_summary": "…", "communication_goal": "…",
    "essential_visual_info": [{ "item": "…", "priority": "must" }],
    "on_screen_text": ["…"],
    "visual_approach": { "method": "diagram", "description": "…", "concept_rule_refs": ["color_system.palette.ink"] },
    "narration_relationship": "contrast", "fact_refs": ["F08"],
    "risk_notes": [{ "fact_id": "F08", "note": "…" }],
    "open_questions": [{ "id": "Q1", "question": "…", "answer": "…" }],
    "status": "approved"
  }],
  "review": { "reviewer": "creator", "approved_at": "…", "comments": [{ "id": "f1", "group": "flow", "text": "…", "decision": "applied|accepted|fixed", "why": "…" }] }
}
```

`order` đánh số từ 1 theo thứ tự beat rồi thứ tự unit trong beat. `review.comments` gồm các vấn đề rà soát và các máy kiểm đã chấp nhận (`group: "machine"`, `decision: "accepted"`).

## Bảng

`review_units` (`visual_plan`, `visual_plan_review`), `visual_plan_versions(project_id, version, script_version, concept_version, document, note, locked_at)` chỉ thêm dòng, trùng số phiên bản thì `409`; `visual_plan_seq(project_id, next)` cấp mã unit trong một câu lệnh. Xoá video xoá cả hai bảng mới (`DELETE /internal/v1/visual/{id}`).
