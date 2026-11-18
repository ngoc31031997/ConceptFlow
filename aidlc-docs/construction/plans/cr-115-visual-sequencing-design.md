# CR-115 · Bước 10 Chuỗi hình (Visual Sequencing) — thiết kế

Nhánh `feature/cr-115-visual-sequencing` (worktree riêng). Nguồn:
- spec bước 10 của Creator (session 2026-11-18) và câu trả lời ở `cr-115-visual-sequencing-questions.md`;
- mockup v2 https://claude.ai/artifact/JoL1XHbGXYhsBDih6ubhUE (Creator chốt 2026-11-18);
- ADR-0049 (visual-service sở hữu bước 8–14), ADR-0047 (AI ngoài, ghi DB sau khi có kết quả), ADR-0044 (bản chốt kịch bản).

Khuôn: bước 9 (CR-114), đã có trên main. Bước 10 làm lại đúng các mảnh của bước 9 (review_units theo beat, máy kiểm tính khi đọc, vòng AI rà soát, bản chốt bất biến); chỗ khác được nêu rõ.

## Creator đã chốt
1. Bước 10 **không đổi thứ tự** unit của bước 9; chỉ thêm cho mỗi unit: vai trò trong mạch, nhịp, chuyển cảnh từ unit trước, motif.
2. Đơn vị duyệt là **một beat**; AI viết từng beat, thấy unit cuối của beat trước. **Motif là một đơn vị duyệt riêng**, phải duyệt trước khi AI viết chuỗi.
3. **Không có thư viện nhịp và chuyển cảnh.** AI tự viết bằng chữ tự do, Creator sửa được. Không kiểm tính dựng được bằng code; bước render lo.
4. Motif là output của bước 10, mỗi motif neo vào một mục `representation_rules` đã chốt ở bước 8. Motif có ở dưới 2 unit thì cảnh báo.
5. Có vòng AI rà soát cả chuỗi như bước 8 và 9.
6. Input đọc ở bản chốt: kế hoạch hình và ngôn ngữ hình ảnh từ DB của visual-service; beat-sheet và kịch bản qua `GET /internal/v1/authoring/{id}/script-versions/latest` (không đọc `project_beat_sheet`/`project_beat`).

## Input
| Input | Nguồn | Phần dùng |
|---|---|---|
| visual-plan | `visual_plan_versions` mới nhất (`PlanRepository`) | `units[]` theo `order`: `unit_id`, `beat_id`, `script_refs`, `duration`, `communication_goal`, `visual_approach`, `narration_relationship`, `on_screen_text`; `script_version`, `concept_version` |
| visual-concept | `visual_concept_versions` mới nhất | `representation_rules` (mã neo motif), `motion_language` (`pace` theo beat, `preferred_transitions`, `avoid`), `color_system.journey`, `do_not_use` |
| beat-sheet | bản chốt kịch bản: `beat:{k}`, `order` | chức năng beat (Hook, Cao trào, Payoff…), tiêu đề, mục tiêu |
| script-final | bản chốt kịch bản: `script:{k}`, `handoff:{k}`, `glossary` | lời thoại của các câu trong unit, ghi chú bàn giao |

**Phiên bản lệch.** Bước 10 làm trên bản chốt mới nhất của bước 9. Nếu bước 8 hoặc kịch bản đã chốt số mới hơn `concept_version`/`script_version` mà bản chốt bước 9 ghi, màn hiện cảnh báo "Kế hoạch hình dựa trên bản cũ, chốt lại bước 9" và `lock` trả `409 plan_outdated`. Viết, sửa, duyệt vẫn làm được. Chưa chốt bước 9 → `409 plan_not_locked` cho mọi route ghi; màn chỉ hiện khung "Chốt Kế hoạch hình ở bước 9 trước".

**Mã neo motif** (`anchor_ref`), cùng cách viết với `concept_rule_refs` của bước 9:
`representation_rules.scientific_concepts.<concept_id>`, `representation_rules.characters.items.<concept_id>`, `representation_rules.objects.<concept_id>`, `representation_rules.invisible.<concept_id>`.

## Stale
Mỗi đơn vị lưu `input_hashes` lúc được ghi (hoặc lúc duyệt), như bước 9.
| Đơn vị | Nguồn (khoá hash) |
|---|---|
| beat `b{k}` | `plan` (hash các unit của beat trong bản chốt bước 9), `prev` (hash unit cuối của beat liền trước, vì chuyển cảnh vào beat phụ thuộc nó), `script` (`beat:{k}`, `script:{k}`, `handoff:{k}`), `motion` (`motion_language` của bước 8), `motifs` (hash danh sách motif đã duyệt) |
| `motifs` | `plan` (cả kế hoạch), `rep` (`representation_rules` của bước 8), `beats` (`order` + chức năng các beat) |

Beat stale: Duyệt trả `409 beat_stale`, "Duyệt tất cả" bỏ qua, không chốt được tới khi viết lại hoặc duyệt lại (duyệt lại = xác nhận vẫn hợp, hash làm mới). Bước 9 chốt lại làm danh sách unit của một beat đổi (thêm, bớt, tách, gộp) → beat đó stale và các unit mới hiện "chưa có chuỗi".

## Đơn vị lưu (`review_units`, đã có)
| `kind` | `unit_key` | `content` |
|---|---|---|
| `visual_sequence` | `b{k}` | `{"items":[SeqItem…]}` theo thứ tự unit của beat |
| `visual_sequence_motifs` | `main` | `{"motifs":[Motif…], "next_seq": n}` |
| `visual_sequence_review` | `main` | cùng khuôn bước 8/9: `findings`, `next_seq`, `notes`, `rounds`, `checks` |

```jsonc
// SeqItem
{ "unit_id": "VP-002",
  "purpose_in_flow": "Giới thiệu hai góc nhìn đối chiếu.",
  "pacing": { "name": "Đối chiếu", "tempo": "medium", "description": "luân phiên hai góc nhìn mỗi 3–4s" },
  "transition_from_previous": { "type": "Match cut", "description": "…", "via_motif": "MOTIF-01" }, // null ở unit đầu video
  "motif_refs": ["MOTIF-01", "MOTIF-02"],
  "status": "pending" }               // pending | approved
// Motif
{ "motif_id": "MOTIF-01", "name": "Hố đen làm điểm neo", "description": "…",
  "anchor_ref": "representation_rules.scientific_concepts.black_hole",
  "role": "Điểm neo thị giác", "planned_units": ["VP-001","VP-002","VP-011"] }
```
- `tempo` ∈ `slow, medium, fast, calm` (enum duy nhất của bước; dùng tô dải dòng chảy và so với `pace` của bước 8). Mọi trường chữ khác tự do.
- `motif_id` do máy cấp `MOTIF-01`, `MOTIF-02`… từ `next_seq`, không dùng lại (giống `VP-xxx`). Khác mẫu JSON gốc (`MOTIF-BH`): mã có nghĩa do AI đặt dễ trùng và đổi nghĩa khi sửa tên; tên đọc được nằm ở `name`.
- `planned_units` là gợi ý của AI lúc đặt motif, đưa vào prompt viết beat. `unit_refs` trong tài liệu chốt do máy gom từ `motif_refs` của các unit (motif có ở đâu là theo chuỗi thật, không theo gợi ý).
- `creator_fields` của dòng beat: đường dẫn `VP-002.pacing`, `VP-002.transition_from_previous`, `VP-002.motif_refs`, `VP-002.purpose_in_flow`; của dòng motif: `MOTIF-01.description`… AI viết lại giữ các trường đó.
- Trạng thái dòng beat `approved` khi mọi item đã duyệt (như bước 9).

## Luồng nghiệp vụ
1. **Mở màn** (`GET /visual-sequence`): đọc input; tính hash, stale; ghép danh sách unit của bản chốt bước 9 với item đã có; tính máy kiểm và dải dòng chảy; trả toàn màn.
2. **AI đặt motif** (`POST /motifs/generate`, `{note}`, `X-External-AI`): một lượt, prompt `visual_sequence_motifs`. Reply `{"motifs":[{same_as, name, description, anchor_ref, role, planned_units}]}`. `same_as` trỏ motif cũ để giữ mã. Motif Creator đã sửa giữ trường đã sửa; motif đang được unit dùng mà AI bỏ → reply bị từ chối (502). Đơn vị motif về `pending`.
3. **Sửa, thêm, xoá motif** (modal): `PUT /motifs/{mid}`, `POST /motifs`, `DELETE /motifs/{mid}`. Xoá: gỡ khỏi `motif_refs` mọi unit, `via_motif` trỏ tới nó thành rỗng, các beat đó về `pending`; xác nhận nêu unit bị ảnh hưởng (luật 5).
4. **Duyệt motif** (`POST /motifs/approve` · `/unapprove`). Không duyệt được khi có `anchor_ref` không tồn tại.
5. **AI viết chuỗi một beat** (`POST /beats/{k}/generate`, `{note}`, `X-External-AI`): cần motif đã duyệt (`409 motifs_not_approved`). Một lượt, prompt `visual_sequence_beat`. Reply `{"items":[{unit_id, purpose_in_flow, pacing, transition_from_previous, motif_refs}]}` phải phủ đúng các unit của beat, đúng thứ tự; khoá lạ, `tempo` ngoài enum, `motif_refs`/`via_motif` không là motif có thật, unit đầu video có chuyển cảnh, unit khác thiếu chuyển cảnh → 502, không ghi gì. Giữ `creator_fields`.
6. **Viết cho mọi beat còn trống** (`POST /generate`): lần lượt theo thứ tự beat, mỗi beat thấy kết quả beat trước. DeepSeek xếp hàng đợi như bước 9.
7. **Sửa tay một item** (modal, `PUT /units/{vp}`): `{fields:{…}}`; item về `pending`.
8. **Duyệt / bỏ duyệt** item; "Duyệt tất cả còn lại" (bỏ qua beat stale, item có lỗi máy kiểm chặn).
9. **Rà soát** (`POST /review`, prompt `visual_sequence_review`): như bước 9, `unit_key` của vấn đề là `VP-xxx` hoặc `MOTIF-xx`, `field` là trường cấp ngoài. Nhóm dùng chung: `accuracy`, `consistency` (nhịp trái `pace` bước 8, chuyển cảnh trùng `motion_language.avoid`, motif lệch cách vẽ đã chốt), `clarity` (lặp một kiểu chuyển cảnh liên tục, nhịp đều đều cả video), `feasibility`. Áp dụng gợi ý, chấp nhận kèm lý do, mở unit.
10. **Chốt** (`POST /lock`). Điều kiện: bước 9 đã chốt và không lệch phiên bản; motif đã duyệt; mọi unit của bản chốt bước 9 có item; mọi item đã duyệt; không beat stale; đã rà soát ít nhất một lần, không còn vấn đề mở; máy kiểm đạt hoặc đã chấp nhận kèm lý do; có thay đổi so với bản trước. Từ v2 hộp xác nhận nêu unit đã đổi.

## Máy kiểm (không lưu, tính mỗi lần đọc)
- `m-seq-cover`: mỗi unit của bản chốt bước 9 có đúng một item; không item thừa (unit đã bị bước 9 bỏ).
- `m-seq-first`: unit đầu video có `transition_from_previous = null`; unit khác có `type` không rỗng.
- `m-seq-motif-ref`: `motif_refs` ⊆ motif đang có.
- `m-seq-via`: `via_motif` (nếu có) nằm trong `motif_refs` của cả unit này lẫn unit liền trước (unit trước có thể ở beat trước).
- `m-motif-anchor`: `anchor_ref` tồn tại trong bản chốt bước 8 mới nhất.
- `m-motif-count`: mỗi motif có ở ít nhất 2 unit (cảnh báo, chấp nhận được kèm lý do).
- `m-seq-empty`: `purpose_in_flow`, `pacing.name` không rỗng.
Lỗi `m-seq-first`, `m-seq-motif-ref`, `m-seq-empty` chặn duyệt item; `m-motif-anchor` chặn duyệt motif; còn lại hiện ở tab Rà soát, chấp nhận được kèm lý do.

## Tài liệu bản chốt (`visual_sequence_versions.document`)
```jsonc
{
  "schema_version": "1.0", "project_id": "…", "document_id": "visual-sequence-v{n}",
  "visual_plan_id": "visual-plan-v{p}", "status": "locked", "version": 1, "locked_at": "…",
  "source_versions": { "script": 2, "visual_concept": 1, "visual_plan": 1 },
  "sequence": [{ "sequence_index": 1, "unit_id": "VP-001", "beat_id": "b1",
    "purpose_in_flow": "…", "pacing": { "name": "…", "tempo": "slow", "description": "…" },
    "transition_from_previous": null, "motif_refs": ["MOTIF-01"], "status": "approved" }],
  "motifs": [{ "motif_id": "MOTIF-01", "name": "…", "description": "…", "anchor_ref": "…", "role": "…", "unit_refs": ["VP-001","VP-002"] }],
  "sequence_notes": ["…"],     // nhận xét của vòng AI rà soát cuối
  "review": { "reviewer": "creator", "approved_at": "…", "comments": [{ "id", "group", "text", "decision", "why" }] }
}
```
`sequence_index` do máy đánh theo `order` của bước 9. `planned_units` không vào tài liệu chốt.

## Dữ liệu (`visual-db`)
| Bảng | Khoá | Ghi chú |
|---|---|---|
| `review_units` (đã có) | `(project_id, kind, unit_key)` | thêm ba `kind` ở trên |
| `visual_sequence_versions` (mới) | `(project_id, version)` | `plan_version`, `concept_version`, `script_version`, `document JSONB`, `note`, `locked_at`; chỉ thêm, trùng số → 409 |
| `project_visual` (đã có) | thêm cột `sequence_model TEXT NOT NULL DEFAULT ''` | `''` = mặc định của bước `visual_sequence` ở llm-service |

Migration thêm vào `internal/adapters/postgres/db.go` (cùng chỗ `visual_plan_versions`), `IF NOT EXISTS`. Purge: thêm `visual_sequence_versions` vào `projectTables` (`project_data_repository.go`); `review_units` đã xoá theo dự án.

## Thiết kế mức thấp
### visual-service
| Lớp | Tệp mới |
|---|---|
| domain | `visual_sequence.go` (SeqItem, Motif, enum `tempo`, ghép tài liệu chốt), `sequence_checks.go` (máy kiểm trên), `sequence_input.go` (input theo beat, hash), `sequence_reply.go` (đọc chặt reply AI), `sequence_prompt_seeds.go` |
| application | `sequence_desk.go` (màn, sửa, motif, duyệt, chốt), `sequence_writer.go` (AI motif + theo beat, lần lượt), `sequence_reviewer.go`, `sequence_prompt.go` |
| adapters | `http/sequence.go`; `postgres/sequence_repository.go` (phiên bản); `PlanDesk.LatestDocument` đã có, dùng lại để đọc bản chốt bước 9 |

### Hợp đồng HTTP công khai (qua api-gateway, `VISUAL_SERVICE_URL`)
| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/visual-sequence` | toàn màn |
| PUT | `…/visual-sequence/model` | `{model}` |
| POST | `…/visual-sequence/motifs/generate` | `{note}`, `X-External-AI` |
| POST | `…/visual-sequence/motifs` · PUT · DELETE `…/motifs/{mid}` | thêm, sửa, xoá |
| POST | `…/visual-sequence/motifs/approve` · `/unapprove` | |
| POST | `…/visual-sequence/generate` | mọi beat còn trống |
| POST | `…/visual-sequence/beats/{k}/generate` | `{note}`, `X-External-AI` |
| PUT | `…/visual-sequence/units/{vp}` | sửa tay |
| POST | `…/visual-sequence/units/{vp}/approve` · `/unapprove` | |
| POST | `…/visual-sequence/approve-all` | |
| POST | `…/visual-sequence/review` | `{note}`, `X-External-AI` |
| POST | `…/visual-sequence/findings/{fid}/apply` · `/accept` `{why}` | |
| POST | `…/visual-sequence/lock` | `{note}` |
| GET | `…/visual-sequence/versions/{n}` | tài liệu đã chốt |

Nội bộ: `GET /internal/v1/visual/{id}/sequence` (bản chốt mới nhất, 404 khi chưa có) cho bước 11. Hợp đồng mới `docs/contracts/visual-sequence-v1.md`, thêm vào README hợp đồng; sửa luôn bảng và route còn thiếu của bước 9 trong `visual-service-v1.md`.

### web-gui
Trang `/projects/:id/visual-sequence` theo mockup (hai tab: Chuỗi hình; Rà soát và chốt). `flow.ts` bước 10 thêm `screen: "visual-sequence"`; `ProjectFlowContext` đánh dấu bước 10 xong khi đã chốt. Dùng lại các thành phần của bước 9 (`AiOperationCard`, `AiModelSelect`, `UnitStatusChip`, `ExternalAiDialog`, `ImpactConfirm`, `CopyBox`, dialog shell). Mới: dải dòng chảy, thẻ motif.

### Prompt
Ba vai `visual_sequence_motifs`, `visual_sequence_beat`, `visual_sequence_review`, seed trong visual-service, đẩy lên llm-service với `owner = "visual"`, `step = visual_sequence`. **Nguyên văn đưa Creator duyệt trước khi ghi seed.** Prompt nói rõ: nhịp và chuyển cảnh viết tự do theo ý đồ hình ảnh; `pace` và `motion_language` của bước 8 là định hướng, không phải danh sách bắt buộc.

## Kiểm thử
- Go: đọc chặt reply (motif, beat), enum `tempo`, máy kiểm (cover, first, motif ref, via, anchor, count), mã MOTIF không dùng lại, xoá motif gỡ ref, giữ `creator_fields`, stale theo beat và theo unit cuối beat trước, lệch phiên bản chặn chốt, điều kiện chốt, v1 rồi v2, purge, HTTP với fake port, kho Postgres.
- Vitest: trang bước 10 (motif, dải dòng chảy, chuỗi theo beat, khoá nút Chốt có lý do).
- Không gọi AI thật (fake reply). Dọn dữ liệu test sau khi chạy.

## Chia việc
1. **CR-115a:** domain, máy kiểm, DB, route, chốt, purge (chưa có AI). Kiểm bằng test.
2. **CR-115b:** ba prompt (sau khi Creator duyệt nguyên văn), AI motif, AI theo beat, rà soát, DeepSeek.
3. **CR-115c:** web-gui bước 10 theo mockup v2.
Mỗi phần rebuild visual-service (và web-gui ở 115c), chờ Creator duyệt rồi commit.

## Creator xác nhận (2026-11-18): cả bốn điểm dưới đây theo đề xuất
1. **Tên bước trong thanh bước:** `flow.ts` đang ghi bước 10 là "Diễn tiến hình ảnh". Đổi thành "Chuỗi hình" như mockup?
2. **Chồng với bước 13 "Thiết kế chuyển động"** (chuyển động và chuyển cảnh của từng shot): đề xuất bước 10 giữ chuyển cảnh ở mức ý đồ giữa hai unit; bước 13 sau này đọc nó và chi tiết hoá theo shot. Đồng ý?
3. **Mã motif** máy cấp `MOTIF-01`… thay vì mã có nghĩa như `MOTIF-BH` (tên đọc được nằm ở `name`). Đồng ý?
4. **Gợi ý unit của AI khi đặt motif** chỉ là gợi ý cho lượt viết beat (`planned_units`), không ghi thẳng vào unit như mockup v2 viết. Lý do: unit có motif nào là quyết định của lượt viết beat (và của Creator), tránh hai nơi cùng ghi một trường. Đồng ý?

## Chưa làm (nói rõ)
- Bước 11 Storyboard đọc bản chốt bước 10.
- Xuất file `visual-sequence.json` ra ngoài app.
