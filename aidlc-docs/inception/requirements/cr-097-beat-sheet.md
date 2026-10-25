# CR-097 · Bước 5 Outline / beat sheet — yêu cầu và thiết kế

Nhánh `feature/cr-097-beat-sheet`. Lộ trình: `aidlc-docs/inception/plans/cr-093-099-content-flow-roadmap.md`. Nguồn: luồng mẫu bước 5 (`docs/document/animation_pipeline_realistic_io_example.md`) và chốt của Creator ngày 2026-10-24 (bên dưới). Creator đã duyệt hướng thiết kế (2026-10-24).

## Chốt của Creator (2026-10-24)

1. Dữ liệu của bước 5 tách ra khỏi `review_units`, lưu ở bảng mới `project_beat_sheet`.
2. Phạm vi chỉ bước 5. Lời thoại bước 6 (`script_beat`) và ghi chú bàn giao bước 7 (`handoff_beat`) vẫn ở `review_units`.
3. Không chuyển dữ liệu cũ: các beat đang có trong `review_units` bị bỏ; dự án cũ dựng lại bộ khung.
4. Beat bỏ hẳn `review_units` và API đơn vị chung `/units/beat/…`; sửa, làm lại, duyệt, bỏ duyệt và xem tác động đi qua đường dẫn riêng `/v1/projects/{id}/beats/{key}…`.
5. Đầu vào của bước 5 là `project_story` (đã duyệt), `project_research` (các dòng `keep = true`) và `project_brief` (đã duyệt). Đã sửa ở fix `e5fb9e9`: dữ kiện và brief đọc đúng bảng, có nút "Xem đầu vào của AI".

## Hiện trạng (đối chiếu code)

- Mỗi beat là một unit `beat` (khoá `b1`, `b2`…), thứ tự và cờ duyệt bộ khung nằm ở unit `beat_sheet:main`, cả hai trong `review_units` (`beat_desk.go`, `beat_units.go`).
- Sửa, làm lại, duyệt, bỏ duyệt, xem tác động đi qua API đơn vị chung `/v1/projects/{id}/units/beat/{key}` (`ReviewUnitsUseCase` + `UnitKind`).
- Bước 6 ghi `input_ref = beat:<key>` và `input_sha` vào mỗi unit lời thoại; cờ "cũ" tính bằng SQL trên `review_units` (`reviewUnitColumns`, `UnitSHA`). Lời thoại viết từ một beat đã đổi vì thế hiện là cũ.
- Beat viết từ câu chuyện ghi `input_ref = story:main`; SQL băm `project_story` để hiện cờ cũ.
- Bảng đang có 9 beat và 1 `beat_sheet` của một dự án; không chuyển.

## Hợp đồng `beat-sheet.v1` · bảng `project_beat_sheet`

Một dòng một dự án, như `project_story`; khoá `project_id`. Các beat nằm trong cột `beats` theo thứ tự, nên đổi thứ tự là ghi một dòng, không phải đánh số lại.

| Cột | Kiểu | Ý nghĩa |
|---|---|---|
| `project_id` | text, khoá | |
| `beats` | jsonb, có thứ tự, tối đa 20 phần tử | mỗi phần tử: `key` (`b1`…, bền, không dùng lại), `fn`, `title`, `goal`, `content`, `facts[]` (mã trong `project_research`), `dur` (giây, 5–600), `next`, `step` (dòng câu chuyện beat dựng từ đó), `status` (`pending`/`approved`), `creator_fields[]`, `redo_count`, `story_hash` |
| `next_seq` | int | số của khoá kế tiếp |
| `skeleton_approved` | bool | bộ khung đã duyệt |
| `rebalanced` | int | số beat bị cân lại thời lượng ở lần viết gần nhất |
| `updated_at` | timestamptz | |

Ý nghĩa các trường và các luật (chức năng, ngân sách thời lượng, Hook ≤ 20 s, beat cuối là Payoff, dữ kiện bắt buộc) giữ nguyên như `docs/contracts/beats-v1.md`. Trường `story_hash` thay `input_ref/input_sha: story:main`: băm của `central_question`, `core_answer`, `narrative_arc` lúc beat được dựng; khác băm hiện tại thì `stale = true`.

Đối chiếu luồng mẫu: `beat-sheet.md` có Beat ID, chức năng, ý chính, thời lượng. Các trường `key`, `fn`, `content`, `dur` ứng với bốn cột đó; `goal`, `next`, `facts`, `step` là phần thêm của hệ thống, giữ như hiện nay.

**Hoàn thành khi:** có beat và mọi beat đã duyệt (giữ như hiện nay). Máy kiểm số beat, thứ tự Hook … Payoff, tổng thời lượng, dữ kiện bắt buộc; Creator xác nhận có thể kể lại video chỉ bằng các beat.

## Thiết kế

**Cổng lưu trữ.** Cổng mới `BeatSheetStorePort` (`GetBeatSheet`, `SaveBeatSheet`) do `PromptTemplateRepository` thực hiện, theo mẫu `StoryStorePort`. Mỗi thao tác của `BeatDeskUseCase` (dựng bộ khung, viết nội dung, thêm, xoá, đổi thứ tự, sửa, làm lại, duyệt) đọc cả dòng, đổi trong bộ nhớ, ghi lại cả dòng trong một câu lệnh. Cả dòng ghi một lần nên không còn trạng thái dở giữa "xoá beat cũ" và "ghi beat mới".

**Đường dẫn riêng cho từng beat.** `BeatDeskUseCase` có `Edit`, `Redo`, `Approve`, `Unapprove`, `Impact`; HTTP: `PUT /beats/{key}`, `POST /beats/{key}/redo`, `/approve`, `/unapprove`, `GET /beats/{key}/impact`, mỗi lệnh trả state của cả màn. `beat` không còn đăng ký trong `ReviewUnitsUseCase`; `ReviewUnitsUseCase` và `UnitKind` không đổi. api-gateway thêm năm route; web-gui đổi lời gọi sang `editBeat`, `redoBeat`, `approveBeat`, `unapproveBeat`, `getBeatImpact`, giao diện không đổi.

**Cờ cũ của lời thoại (bước 6).** `UnitSHA(kind="beat", key)` và SQL stale cho `input_ref = beat:<key>` đọc phần tử tương ứng của `project_beat_sheet.beats`, băm các trường nội dung (`fn`, `title`, `goal`, `content`, `facts`, `dur`, `next`), không băm `status`, như `md5(content)` của unit hiện nay. Bước 6 và 7 không đổi mã.

**Tác động đến dữ kiện.** `FactDependents` (bước 3 hỏi beat nào dùng một dữ kiện) đọc từ `project_beat_sheet`.

**Khởi động và xoá.** Migration `CREATE TABLE IF NOT EXISTS project_beat_sheet`; xoá phần authoring của dự án thì xoá dòng này. Các unit `beat` và `beat_sheet` cũ trong `review_units` bị xoá lúc khởi động (câu `DELETE` nằm trong DDL khởi động; an toàn khi chạy lại vì không còn gì ghi hai loại đó), vì không chuyển dữ liệu. Lời thoại (`script_beat`) và bàn giao (`handoff_beat`) của các beat cũ không bị xoá; DB hiện tại không có dòng nào.

**Prompt.** Không đổi vai `beat_sheet`; các biến `{{brief}}`, `{{facts}}`, `{{story}}` đã đúng nguồn sau fix `e5fb9e9`. Không đổi seed, không cần re-seed.

**Màn hình.** Giữ nguyên bố cục. Nút "Xem đầu vào của AI" đã có.

## Việc cần làm (sau khi duyệt)

1. Migration `project_beat_sheet`; kiểu domain `StoredBeatSheet`; repository và `BeatSheetStorePort` (+ test Postgres khứ hồi).
2. `BeatDeskUseCase` và `beat_units.go` chuyển sang cổng mới; `BeatUnitKind` cung cấp Get/SetStatus.
3. `ReviewUnitsUseCase` và repository: đọc, ghi trạng thái, `UnitSHA` và SQL stale cho kind `beat` từ `project_beat_sheet`; `story_hash` thay `story:main`.
4. `FactDependents`; xoá dòng khi xoá phần authoring; xoá unit `beat`/`beat_sheet` cũ lúc khởi động.
5. `docs/contracts/beat-sheet-v1.md` thay `beats-v1.md` (thêm các đường dẫn riêng của beat), một dòng trong `docs/contracts/README.md`.
6. Test: domain, application (AI giả), http, Postgres, stale của lời thoại khi sửa beat, web-gui không đổi.
7. Rebuild authoring-service (web-gui nếu có đổi).

## Phụ thuộc và trùng lặp

- Nhánh `fix/beat-sheet-from-story` còn thay đổi chưa commit ở `beat_desk.go`, `beat_writer.go`, `prompt_seed_beat.go`, `story.go`, `BeatCard.tsx` và hai file `beat_outline*`. Nó trùng phần với CR này và với CR-096 (lộ trình đã ghi). Chưa có quyết định của Creator về nhánh đó; CR này không chạm vào nó. Khi nó được giao sau CR này sẽ xung đột ở các file beat.
- CR-098 (bước 6) và CR-099 (bước 7) chưa bắt đầu; CR này giữ lời thoại và bàn giao chạy được qua lớp băm ở trên.

## Ngoài phạm vi

Chuyển lời thoại bước 6 và ghi chú bàn giao bước 7 sang bảng riêng (CR-098, CR-099); sửa brief rỗng ở bước 6 và 7 (thuộc CR-098/099); chuyển 9 beat cũ.
