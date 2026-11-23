# Visual sequence v1 (bước 10 Chuỗi hình)

Bước 10 (visual-service, ADR-0049) đọc kế hoạch hình đã chốt ở bước 9 và, cho từng unit của kế hoạch, thêm vai trò trong mạch, nhịp, chuyển cảnh từ unit trước và motif. Bước này không đổi thứ tự unit. Motif là hình lặp lại có chủ ý để nối mạch; mỗi motif neo vào một quy tắc thể hiện đã chốt ở bước 8. Mọi đường dẫn công khai đi qua api-gateway. Mã: `services/visual-service/internal/domain/` (`visual_sequence.go`, `sequence_input.go`, `sequence_checks.go`, `sequence_document.go`, `sequence_reply.go`, `sequence_prompt_seeds.go`), `internal/application/sequence_desk.go`, `sequence_actions.go`, `sequence_writer.go`, `sequence_prompt.go`, `internal/adapters/http/sequence.go`, `internal/adapters/postgres/sequence_repository.go`.

AI làm ba việc, mỗi việc một lượt model (`step = visual_sequence`, model riêng của video ở `project_visual.sequence_model`, rỗng là model mặc định của bước): đặt motif (`visual_sequence_motifs`), viết chuỗi một unit (`visual_sequence_unit`, mỗi unit một lượt; một beat hay mọi unit còn trống là nhiều lượt nối tiếp), rà soát cả chuỗi (`visual_sequence_review`). Nguyên văn prompt: `aidlc-docs/construction/plans/cr-115b-prompts-draft.md`. Kết quả chỉ được lưu sau khi reply về đủ và đúng khuôn; reply sai khuôn trả `502`, không ghi gì.

## Input

| Input | Nguồn | Phần dùng |
|---|---|---|
| Kế hoạch hình chốt | bản mới nhất của `visual_plan_versions` ([`visual-plan-v1.md`](visual-plan-v1.md)) | `units[]` theo `order`: mã, beat, câu, mục tiêu, cách hiển thị, thời lượng; `source_versions` |
| Ngôn ngữ hình ảnh chốt | bản mới nhất của `visual_concept_versions` ([`visual-concept-v1.md`](visual-concept-v1.md)) | `representation_rules` (mã neo motif), `motion_language` (`pace` theo beat) |
| Kịch bản chốt | `GET /internal/v1/authoring/{id}/script-versions/latest` ([`script-lock-v1.md`](script-lock-v1.md)) | `beat:{k}`, `order`, `script:{k}`, `handoff:{k}` |

Chưa chốt bước 8: `409 concept_not_locked`; chưa chốt bước 9: `409 plan_not_locked`. **Kế hoạch lệch phiên bản:** khi `source_versions` của bản chốt bước 9 khác số chốt mới nhất của kịch bản hoặc ngôn ngữ hình ảnh, màn nêu lý do ở `input.outdated[]` và trong `blockers`, còn `lock` trả `409 plan_outdated`. Sửa, duyệt vẫn làm được.

**Mã neo motif** (`anchor_ref`): `representation_rules.scientific_concepts.<concept_id>`, `representation_rules.characters.items.<concept_id>`, `representation_rules.objects.<concept_id>`, `representation_rules.invisible.<concept_id>`. Toàn màn trả `motifs.anchors` (mã → chữ mô tả cách vẽ).

## Đơn vị lưu (`review_units`)

| `kind` | `unit_key` | `content` |
|---|---|---|
| `visual_sequence` | khoá beat (`b1`) | `{"items":[SeqItem]}` theo thứ tự unit của beat |
| `visual_sequence_motifs` | `main` | `{"motifs":[Motif], "next_seq": n}` |
| `visual_sequence_review` | `main` | cùng khuôn rà soát bước 8 và 9 |

```jsonc
// SeqItem
{ "unit_id": "VP-002", "purpose_in_flow": "…",
  "pacing": { "name": "Đối chiếu", "tempo": "medium", "description": "…" },
  "transition_from_previous": { "type": "Match cut", "description": "…", "via_motif": "MOTIF-01" },
  "motif_refs": ["MOTIF-01"], "status": "pending" }
// Motif
{ "motif_id": "MOTIF-01", "name": "…", "description": "…",
  "anchor_ref": "representation_rules.scientific_concepts.black_hole", "role": "…", "planned_units": [] }
```

- `pacing.tempo` ∈ `slow, medium, fast, calm`; mọi trường chữ khác tự do. `transition_from_previous` là `null` ở unit đầu video và bắt buộc ở unit khác; `via_motif` không bắt buộc.
- `motif_id` máy cấp `MOTIF-01`, `MOTIF-02`… từ `next_seq`; mã đã xoá không dùng lại.
- `planned_units` là gợi ý của AI lúc đặt motif. Motif có ở unit nào là theo `motif_refs` của item.
- `creator_fields` của dòng beat là `VP-002.pacing`…; của dòng motif là `MOTIF-01.description`….
- Dòng beat `approved` khi mọi unit của beat có item đã duyệt. Motif duyệt chung một lần (trạng thái của dòng motif).

**Stale.** Dòng beat lưu `input_hashes` của: `plan` (các unit của beat trong bản chốt bước 9, không tính số thứ tự), `prev` (unit cuối của beat liền trước), `script` (`beat:{k}`, `script:{k}`, `handoff:{k}`), `motion` (`motion_language`), `motifs` (mã, tên, mô tả, neo, vai trò của mọi motif; không tính `planned_units`). Dòng motif lưu `plan` (cả kế hoạch), `rep` (`representation_rules`), `beats` (thứ tự và chức năng beat). Beat stale: duyệt item trả `409 beat_stale`, "Duyệt tất cả" bỏ qua; `beats/{k}/confirm` xác nhận chuỗi vẫn hợp (lấy hash hiện tại, mọi item của beat về `pending`). Motif stale: chặn chốt; duyệt motif lại là xác nhận (lấy hash hiện tại).

## Máy kiểm

Của từng item (chặn duyệt item): vai trò trong mạch, tên nhịp không rỗng; `tempo` thuộc danh sách; chuyển cảnh đúng vị trí (unit đầu không có, unit khác có và có `type`); `motif_refs` là motif có thật, không lặp; `via_motif` nằm trong `motif_refs` của chính item.

Của từng motif (chặn duyệt motif): mã đúng dạng; tên, mô tả, vai trò không rỗng; `anchor_ref` có trong bản chốt bước 8.

Của cả chuỗi (tính mỗi lần đọc, chấp nhận được kèm lý do, quyết định gắn dấu vân tay của `detail`):
- `m-seq-via`: `via_motif` có trong `motif_refs` của unit liền trước (có thể ở beat trước).
- `m-motif-count`: mỗi motif xuất hiện ở ít nhất 2 unit.

`coverage` (`units_total`, `units_sequenced`, `missing[]`, `extra[]`): mọi unit của kế hoạch phải có item; item của unit kế hoạch không còn là `extra`. Không chấp nhận được, chỉ chặn chốt.

## API công khai

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/visual-sequence` | Toàn màn (kèm `model`, `default_model`, `models[]`): `input` (`plan_version`, `concept_version`, `script_version`, `units_count`, `outdated[]`), `motifs` (`status`, `stale`, `stale_sources[]`, `motifs[]` kèm `anchor_text`, `unit_refs`, `problems`, `creator_fields`; `anchors`), `beats[]` (`key`, `number`, `fn`, `title`, `pace`, `written`, `approved`, `stale`, `stale_sources[]`, `units[]`: thông tin unit bước 9, `previous_unit`, `item` hoặc `null`, `problems`, `creator_fields`), `items_approved`, `coverage`, `checks[]`, `review`, `blockers[]`, `can_lock`, `versions[]`, `locked_version`, `next_version`, `draft[]` |
| PUT | `…/model` | `{"model":"…"}` model của video cho bước 10; `""` là model mặc định. Model lạ `400` |
| POST | `…/motifs/generate` | `{"note":"…"}`, nhận `X-External-AI`. AI đặt motif. Reply `{"motifs":[{same_as, name, description, anchor_ref, role, planned_units}]}`: `same_as` là mã motif cũ để giữ mã, rỗng là motif mới (nhận mã kế tiếp). Trường Creator đã sửa giữ nguyên. `502` khi: khoá lạ (kể cả `motif_id`), `same_as` không có thật hoặc lặp, `planned_units` có unit lạ, bỏ motif đang được item dùng hoặc Creator đã sửa. Motif về `pending`, lấy hash input hiện tại |
| POST | `…/generate` | `{"note":"…"}`. AI viết lần lượt mọi unit chưa có item, theo thứ tự kế hoạch, mỗi unit một lượt model; unit sau đọc item unit trước vừa viết. Cần motif đã duyệt (`409 motifs_not_approved`). Lỗi giữa chừng dừng lại, unit đã viết giữ nguyên |
| POST | `…/beats/{k}/generate` | `{"note":"…"}`. AI viết lại mọi unit của beat, mỗi unit một lượt; xong cả beat thì beat lấy hash input hiện tại (hết stale). Không nhận `X-External-AI` (nhiều lượt). Beat lạ `404` |
| POST | `…/units/{vp}/generate` | `{"note":"…"}`, nhận `X-External-AI`. Cần motif đã duyệt (`409 motifs_not_approved`). Reply `{"item":{unit_id, purpose_in_flow, pacing, transition_from_previous, motif_refs}}`. `502` khi: khoá lạ, `unit_id` khác unit đang viết, `tempo` ngoài enum, motif lạ, unit đầu video có chuyển cảnh, unit khác thiếu chuyển cảnh hoặc `type` rỗng, `via_motif` lạ hoặc không có trong `motif_refs`. Trường Creator đã sửa của unit giữ nguyên; item về `pending`. Beat chưa có item nào thì lấy hash input hiện tại; beat đã có item giữ hash cũ (beat stale vẫn stale tới khi xác nhận hoặc viết lại cả beat). Unit lạ `404`, đang chạy `409` |
| POST | `…/review` | `{"note":"…"}`, nhận `X-External-AI`. Cần ít nhất một item (`409`). Vấn đề có `unit_key` là `VP-…` (trường `purpose_in_flow`, `pacing`, `transition_from_previous`, `motif_refs`) hoặc `MOTIF-…` (trường `name`, `description`, `anchor_ref`, `role`); khác đi thì `502`. Gộp vòng như bước 8 và 9 |
| PUT | `…/units/{vp}` | Sửa tay `{"fields":{…}}`, chỉ `purpose_in_flow`, `pacing`, `transition_from_previous`, `motif_refs`. Unit chưa có item thì bắt đầu từ item rỗng. Trường thành của Creator, item về `pending`. Unit lạ `404`; trường lạ, sai kiểu `400` |
| POST | `…/units/{vp}/approve` · `/unapprove` | Duyệt cần item đã có (`409`), beat không stale (`409 beat_stale`), item hết `problems` (`409`) |
| POST | `…/approve-all` | Duyệt mọi item đạt trong beat không stale |
| POST | `…/beats/{k}/confirm` | Xác nhận beat stale vẫn hợp. Beat lạ `404`, beat chưa có item `409` |
| POST | `…/motifs` | `{"fields":{name, description, anchor_ref, role}}` thêm motif, nhận mã kế tiếp; motif về `pending` |
| PUT | `…/motifs/{mid}` | Sửa các trường trên; motif về `pending`. Motif lạ `404` |
| DELETE | `…/motifs/{mid}` | Xoá motif; gỡ khỏi `motif_refs` và `via_motif` của mọi item, item bị đổi về `pending` |
| POST | `…/motifs/approve` · `/unapprove` | Duyệt cần ít nhất một motif (`409`) và mọi motif hết `problems` (`409`) |
| POST | `…/findings/{fid}/apply` | Ghi `fix` vào `field` của item (`VP-…`) hoặc motif (`MOTIF-…`); chỉ trường văn bản; item hoặc motif về `pending` |
| POST | `…/findings/{fid}/accept` | `{"why":"…"}` chấp nhận vấn đề, hoặc máy kiểm `m-seq-via`, `m-motif-count` đang không đạt |
| POST | `…/lock` | `{"note":"…"}` chốt phiên bản. `409 plan_outdated` khi kế hoạch lệch; `409` kèm `blockers` khi chưa đủ điều kiện; `409` khi không đổi gì |
| GET | `…/versions/{n}` | Phiên bản đã chốt kèm `document` |

Điều kiện chốt: kế hoạch không lệch phiên bản; có motif, motif đã duyệt và không stale; mọi unit có item, không item thừa; mọi item đã duyệt và hết `problems`; không beat stale; đã rà soát ít nhất một lần và không còn vấn đề mở; máy kiểm cả chuỗi đạt hoặc đã chấp nhận; có thay đổi so với bản chốt trước.

Lỗi `{"error":"…","code":"…"}` như bước 8 và 9.

## Lượt chạy AI (ADR-0052)

Ba lượt AI một prompt chạy qua lượt chạy có xem trước của visual-service, `/v1/projects/{id}/visual/ai-runs`, theo [`ai-runs-v1.md`](ai-runs-v1.md). Điểm dừng xem trước của từng lượt, sau khi đọc và kiểm tra câu trả lời, trước mọi lần ghi: `/motifs/generate` → `{motifs, next_seq}` (bộ motif sau khi giữ id và trường Creator đã sửa); `/units/{unit}/generate` → item của unit sau khi giữ trường Creator đã sửa; `/review` → `{findings, next_seq, notes, rounds, checks}` (rà soát đã gộp với vấn đề cũ).

`/generate` (mọi unit còn trống) và `/beats/{beat}/generate` chạy từng unit, mỗi unit đọc item **đã lưu** của unit trước, nên vẫn lưu ngay từng unit và không đi qua lượt chạy. Trên kênh extension, web-gui đưa mỗi unit vào hàng đợi AI theo thứ tự kế hoạch; kênh thủ công không dùng được cho chạy hàng loạt.

## API nội bộ

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/internal/v1/visual/{id}/sequence` | Tài liệu của bản chốt mới nhất. `404` khi chưa chốt |

## Tài liệu bản chốt (`visual_sequence_versions.document`)

```jsonc
{
  "schema_version": "1.0", "project_id": "…", "document_id": "visual-sequence-v1",
  "visual_plan_id": "visual-plan-v1", "status": "locked", "version": 1, "locked_at": "…",
  "source_versions": { "script": 2, "visual_concept": 1, "visual_plan": 1 },
  "sequence": [{ "sequence_index": 2, "unit_id": "VP-002", "beat_id": "b2", "purpose_in_flow": "…",
    "pacing": { "name": "…", "tempo": "medium", "description": "…" },
    "transition_from_previous": { "type": "…", "description": "…", "via_motif": "MOTIF-01" },
    "motif_refs": ["MOTIF-01"], "status": "approved" }],
  "motifs": [{ "motif_id": "MOTIF-01", "name": "…", "description": "…", "anchor_ref": "…", "role": "…", "unit_refs": ["VP-001", "VP-002"] }],
  "sequence_notes": ["…"],
  "review": { "reviewer": "creator", "approved_at": "…", "comments": [{ "id": "…", "group": "…", "text": "…", "decision": "…", "why": "…" }] }
}
```

`sequence_index` là vị trí của unit trong kế hoạch, từ 1. `unit_refs` do máy gom từ `motif_refs`. `sequence_notes` là nhận xét của vòng rà soát gần nhất. `review.comments` gồm các vấn đề rà soát và máy kiểm đã chấp nhận (`group: "machine"`).

## Bảng

`review_units` (ba `kind` ở trên); `visual_sequence_versions(project_id, version, plan_version, concept_version, script_version, document, note, locked_at)` chỉ thêm dòng, trùng số phiên bản thì `409`. Xoá video xoá cả bảng này (`DELETE /internal/v1/visual/{id}`).
