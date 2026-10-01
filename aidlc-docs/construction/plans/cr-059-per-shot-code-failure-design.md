# CR-059 — Thiết kế: lỗi bước Code theo từng shot, giữ shot đã viết

Trạng thái: **DESIGN — đã duyệt**.

Creator đã chọn: làm mục 1, 2, 4 của đề xuất; **bỏ** mục 3 (lượt chẩn đoán tự động).

Service bị ảnh hưởng: `llm-service`, `authoring-service`, `web-gui`. `orchestrator`, `rendering` không đổi code (orchestrator chỉ chép nguyên câu lỗi của authoring vào `project_errors`).

## Yêu cầu gốc (nguyên văn)

> xem dự án này Các cấp độ của nền văn minh sao có shot 3.3-3.5 lại không chạy được nhỉ.

> làm sao có thể tránh những kịch bản kiểu này nhỉ, hoặc ít nhất trả lỗi cụ thể cho shot đó như cách bạn đã phân tích.

> làm mục 1 trước đi

Trả lời đề xuất:

> chưa cần
> *(kèm trích đoạn mục 3 — chẩn đoán tự động)*
> llàm 1 2 4 nhé

FR-3 (chẩn đoán) bị bỏ; các FR còn lại giữ số cũ để khớp với đề xuất.

"Mục 1" là phần đã đề xuất trong hội thoại: (a) giữ và lưu các shot đã viết xong, chỉ đánh lỗi shot thật sự lỗi, chạy lại chỉ shot đó; (b) thông báo lỗi chỉ đích danh shot; (c) chẩn đoán tự động — **Creator đã bỏ, chưa làm**. **Không** gồm siết prompt storyboard hay bước kiểm tra tính khả thi storyboard.

## Hiện trạng (đo trên dữ liệu thật và code)

### Dự án `6deec24f-8030-4423-8a27-b5c52791c997` — "Các cấp độ của nền văn minh"

Remotion, `deepseek-ai/deepseek-v4.1-flash`, 3 shot mỗi đoạn, 17 đoạn. 16 đoạn `done`; đoạn `3.3-3.5` `failed`:

```
error_kind = budget
error_message = chunk 3.5-3.5: hive: budget: model suy nghĩ quá 60000 ký tự mà chưa viết được chữ nào — dừng sớm
```

Lượt chạy mất 756 giây. Câu lỗi Creator thấy: "Bước Code chưa xong: 1/17 đoạn lỗi (3.3-3.5: budget)…". Câu này không nói shot nào lỗi, không nói vì sao, và không cho biết 3.3, 3.4 đã viết được hay chưa.

Visual của shot 3.5 ("Người đàn ông gật đầu, đặt bút xuống. Ba vòng tròn lơ lửng trên bàn, mỗi vòng sáng lên khi được nhắc tới") mâu thuẫn với 3.2 (ba vòng vẽ **trên tờ giấy**, lồng nhau) và 3.4 (mỗi vòng "có một chấm vàng ở giữa"). Shot này còn đòi dựng động tác nhân vật và canh ba lần sáng theo lời đọc dài. Đây là phân tích thủ công; CR này không tự động hoá phần phân tích.

CR-056 đã gặp đúng kiểu lỗi này: 4 đoạn `budget` trên một dự án khác. Lỗi này lặp lại, không phải ngẫu nhiên.

### Code

- **Chia đôi khi hết budget**: `write()` trong `CodePipeline.run` ([run.py:562-584](../../../services/llm-service/app/pipeline/run.py#L562-L584)). Một đoạn lỗi `budget`/`truncated` (`SPLIT_KINDS`, [run.py:63](../../../services/llm-service/app/pipeline/run.py#L63)) được viết lại thành hai nửa, đệ quy tới một shot. Các nửa chạy **tuần tự**, kết quả gộp vào biến cục bộ `out`. Nửa nào lỗi thì exception bay ra khỏi vòng `for`:
  - `out` (shot đã viết xong, đã trả tiền) bị bỏ;
  - nửa sau chưa chạy cũng bị bỏ qua.
- **Báo lỗi đoạn**: `do_chunk` → `segment_failed(seg, exc)` ([run.py:504-508](../../../services/llm-service/app/pipeline/run.py#L504-L508), [run.py:586-595](../../../services/llm-service/app/pipeline/run.py#L586-L595)). Sự kiện chỉ có `key` và `error {kind, message}`; message là chuỗi `"chunk 3.5-3.5: <lỗi provider>"` do `_ask` ghép ([run.py:459-461](../../../services/llm-service/app/pipeline/run.py#L459-L461)).
- **Đoạn đã lưu được dùng lại** chỉ khi nội dung có **đủ mọi shot** ([run.py:547-554](../../../services/llm-service/app/pipeline/run.py#L547-L554)).
- **Provider** chỉ đếm `reasoning_chars`, không giữ chữ reasoning ([provider.py:201-210](../../../services/llm-service/app/provider.py#L201-L210)). Khi vượt trần thì ném `LLMError(BUDGET)` không retry ([provider.py:240-252](../../../services/llm-service/app/provider.py#L240-L252)).
- **authoring-service**:
  - `codeRun.onEvent` `"segment_failed"` → `SaveSegmentFailed(key, kind, message, duration)` ([generate_authoring_code.go:319-322](../../../services/authoring-service/internal/application/generate_authoring_code.go#L319-L322)). Hàm này chỉ đổi `status/error_*`, giữ nội dung cũ ([code_segment_repository.go:128-132](../../../services/authoring-service/internal/adapters/postgres/code_segment_repository.go#L128-L132)).
  - `sendStoredSegments` chỉ gửi lại đoạn `done` ([generate_authoring_code.go:177-189](../../../services/authoring-service/internal/application/generate_authoring_code.go#L177-L189)).
  - `ErrSegmentsIncomplete.Error()` ghép "key: kind" ([generate_authoring_code.go:240-262](../../../services/authoring-service/internal/application/generate_authoring_code.go#L240-L262)).
- **web-gui** `CodeSegmentsPanel`: đoạn `failed` hiện `error_kind: error_message` và nút "Chạy đoạn này" ([CodeSegmentsPanel.tsx:325-339](../../../services/web-gui/src/components/CodeSegmentsPanel.tsx#L325-L339)).
- **Hợp đồng**: [authoring-llm-code-v2.md](../../../docs/contracts/authoring-llm-code-v2.md), [ADR-0030](../../decisions/ADR-0030-code-segments-and-v2-code-contract.md).

`graphify affected CodePipeline`: `app/main.py` (`create_app`, `code_generate`) và `tests/test_pipeline.py`. Không service nào khác gọi trực tiếp.

## Yêu cầu

- **FR-1 — Giữ shot đã viết.**
  - Khi một đoạn bị chia nhỏ, **mọi phần** đều được chạy, kể cả khi một phần trước đó lỗi.
  - Shot viết xong được gửi về cùng sự kiện `segment_failed` và lưu như **nội dung dở** của đoạn: đoạn vẫn `failed`, nhưng có `content.shots` cho các shot đã xong.
  - Lần chạy lại đoạn đó (tự động hoặc "Chạy đoạn này") **chỉ gọi model cho shot còn thiếu**, rồi ghép với shot đã lưu thành đoạn `done`.
  - Nội dung dở chỉ được dùng lại khi vân tay đoạn khớp. Storyboard đổi thì nó bị xoá như mọi nội dung lệch vân tay.
- **FR-2 — Lỗi chỉ đích danh shot.**
  - Mỗi đoạn lỗi mang danh sách `failed_shots`.
  - Câu lỗi bằng tiếng Việt, có: shot nào; vì sao (`budget` → "model suy nghĩ quá N ký tự mà chưa viết được chữ nào"; `truncated` → "câu trả lời bị cắt giữa chừng"; loại khác → câu lỗi gốc); đã thử những gì ("đã thử cả đoạn 3.3-3.5, rồi riêng shot 3.5"); shot nào đã lưu.
  - Ví dụ: `Shot 3.5: model suy nghĩ quá 60000 ký tự mà chưa viết được chữ nào (đã thử cả đoạn 3.3-3.5, rồi riêng shot 3.5). Đã lưu shot 3.3, 3.4.`
  - Câu tổng của bước dùng shot, không dùng khoá đoạn: `Bước Code chưa xong: lỗi ở shot 3.5 (budget)…`.
- **FR-4 — Hiển thị.** Ở dòng đoạn lỗi, `CodeSegmentsPanel` hiện:
  - câu lỗi theo shot;
  - "Đã lưu: shot 3.3, 3.4" (nếu có);
  - Nút "Chạy đoạn này" đổi nhãn thành "Chạy lại shot 3.5" khi đoạn có nội dung dở.
- **NFR**:
  - Không tăng số lượt gọi khi không có lỗi.
  - Không đổi vân tay, nên đoạn đã `done` của mọi dự án vẫn dùng lại được.

**Nghiệm thu**

1. Test pipeline giả lập dự án này: đoạn `3.3-3.5` lỗi `budget`, nửa `[3.3, 3.4]` xong, `[3.5]` lỗi `budget`. Kỳ vọng: `segment_failed` có `content.shots` gồm 3.3, 3.4; `failed_shots == ["3.5"]`; câu lỗi như FR-2; không có lượt gọi nào ngoài các lượt `chunk`.
2. Chạy lại với nội dung dở đó: chỉ một lượt `chunk` cho `3.5-3.5`, rồi `segment_done` có đủ 3 shot.
3. Nửa đầu lỗi thì nửa sau vẫn chạy.
4. Màn Code của dự án thật hiện đúng FR-4 sau khi chạy lại đoạn 3.3-3.5. Nếu shot 3.5 vẫn lỗi thì thấy câu lỗi theo shot và "Đã lưu: shot 3.3, 3.4"; nếu qua thì đoạn `done`.

**Ngoài phạm vi**

- Chẩn đoán tự động shot lỗi (Creator đã bỏ; có thể làm ở CR sau).
- Siết prompt storyboard.
- Bước kiểm tra tính khả thi storyboard.
- Đổi `CODE_MAX_REASONING_CHARS`.
- Nút "áp gợi ý vào storyboard" tự động: Creator tự sửa storyboard.
- Prompt "Sao chép prompt" cho riêng shot thiếu: vẫn là prompt cả đoạn, dán vào vẫn thay cả đoạn như hiện nay.

## Giải pháp đề xuất

### 1. Nội dung dở nằm trong đoạn, không đổi cách chia đoạn

Có hai cách giữ shot đã viết:

| | A. Đoạn lỗi mang nội dung dở (**đề xuất**) | B. Tách đoạn thành đoạn con mới (`3.3-3.4`, `3.5-3.5`) |
|---|---|---|
| Khoá, vân tay, `plan` | Không đổi | Plan đổi giữa lượt; `ApplySegmentPlan` phải biết đoạn con; `/v2/code/plan` không còn là nơi duy nhất giữ cách chia (ADR-0030) |
| Lưu trữ | Thêm 2 cột, nội dung dùng lại cột `content` | Thêm/xoá dòng giữa lượt chạy |
| Chạy lại | `only=[key]`, pipeline tự biết shot nào thiếu | Creator thấy đoạn lạ, khoá thay đổi theo lỗi |

Chọn **A**, vì nó giữ nguyên hợp đồng ADR-0030 (một đoạn = một khoá ổn định) và chỉ **thêm trường**.

### 2. `write()` không ném mất phần đã xong

`write()` trả về `ChunkOutcome(shots: dict[str, str], failures: list[ShotFailure])`, không ném lỗi với các lỗi thuộc về shot:
- Gọi `_ask` cho `ids`. Thành công thì trả `shots` đầy đủ.
- Lỗi thuộc `SPLIT_KINDS` và `len(ids) >= 2`: chia đôi, chạy **cả hai** nửa (tuần tự như hiện nay để giữ thứ tự gọi), gộp `shots` và `failures`. Mỗi nửa ghi thêm `tried` (chuỗi các khoá đã thử) để câu lỗi nói được "đã thử cả đoạn … rồi …".
- Lỗi khác, hoặc lỗi một shot: trả `failures=[ShotFailure(ids, kind, message, tried)]`.
- `STOP_NOW_KINDS` vẫn ném ra như cũ để dừng cả lượt.

`do_chunk` nhận outcome:
- không có `failures` → như hiện nay (`_settle_chunk`, `segment_done`);
- có `failures` → `segment_failed` với `fingerprint`, `content: {"shots": shots}` (`null` khi rỗng), `failed_shots`, `error.message` theo FR-2.

**Lấy lại nội dung dở**: `stored(seg)` trả về nội dung có vân tay khớp. Ở bước dựng `todo`:
- đoạn có đủ shot → dùng luôn;
- đoạn có **một phần** shot và được `wanted` → vào `todo` cùng phần đã có;
- `write()` chỉ được gọi cho các shot còn thiếu, ghép với phần đã có rồi đi tiếp như một đoạn bình thường (kiểm tra sớm Remotion, `segment_done` với đủ shot).

### 3. Hợp đồng `segment_failed` (chỉ thêm trường)

```json
{"type": "segment_failed", "key": "3.3-3.5",
 "fingerprint": "…",
 "content": {"shots": {"3.3": "…", "3.4": "…"}},
 "failed_shots": ["3.5"],
 "error": {"kind": "budget", "message": "Shot 3.5: model suy nghĩ quá 60000 ký tự … Đã lưu shot 3.3, 3.4."}}
```

- `content` là `null` khi không có shot nào xong.
- `segments` của body chung nay **có thể chứa đoạn lỗi có nội dung dở**: `content.shots` thiếu shot. `llm-service` chỉ viết shot còn thiếu.
- Hai bên deploy cùng lúc.
  - `authoring-service` cũ bỏ qua trường mới (`json.Unmarshal`), nên vẫn chạy như hôm nay.
  - `llm-service` cũ nhận đoạn thiếu shot thì coi như chưa có, viết lại cả đoạn như hôm nay.

### 4. authoring-service

- **Lược đồ** (`db.go`), thêm vào `authoring_segments`:
  - `failed_shots TEXT[] NOT NULL DEFAULT '{}'`.
- **`domain.CodeSegment`** thêm:
  - `FailedShots []string \`json:"failed_shots"\``.
- **`SaveSegmentFailed`** nhận `domain.SegmentFailure{Kind, Message, DurationMS, Fingerprint, Content, FailedShots}`. Cập nhật trong một transaction:
  - nội dung đang lưu có cùng vân tay và **đủ mọi shot** (đoạn từng `done`, Creator bấm chạy lại rồi lỗi) → giữ nguyên nội dung cũ, đúng quy tắc hiện nay "lượt chạy lại lỗi không được vứt nội dung đang có";
  - còn lại, nếu sự kiện có `content` → lưu `content` + `fingerprint` mới;
  - không có `content` → giữ nội dung cũ.
- **Xoá `failed_shots`**: `MarkSegmentRunning`, `SaveSegmentDone`, `ApplySegmentPlan` (khi đưa về `pending`) và `FailRunningSegments` đặt `failed_shots='{}'`.
- **`sendStoredSegments`** gửi thêm đoạn `failed` có `content`, kể cả khi đó là đoạn `skip`. `skip` chỉ bỏ nội dung của đoạn `done` mà Creator muốn viết lại.
- **`ErrSegmentsIncomplete.Error()`** dùng `FailedShots` khi có: `"Bước Code chưa xong: lỗi ở shot 3.5 (budget)"`. Đoạn không có `FailedShots` (dữ liệu cũ, lỗi khung) giữ cách ghi `key: kind`.
- **Client** `llm_service_client.go` đọc thêm `failed_shots` vào `CodeEvent.FailedShots`.

### 5. web-gui

- Thêm vào `CodeSegment` (`api/client.ts`):
  - `failed_shots: string[]`.
- `CodeSegmentsPanel`, dòng đoạn `failed`, theo thứ tự trên xuống: câu lỗi → "Đã lưu: shot …" (dùng token `--success`) → các nút.
  - Nhãn nút chạy: "Chạy lại shot 3.5" khi có `failed_shots` và nội dung dở; còn lại giữ như cũ.

## Phạm vi

| Service | File | Thay đổi |
|---|---|---|
| llm-service | `app/pipeline/run.py` | `ChunkOutcome`, `ShotFailure`, `write()`/`do_chunk()`, dùng nội dung dở, câu lỗi theo shot, sự kiện `segment_failed` mở rộng |
| llm-service | `tests/test_pipeline.py` | test mới (xem Kế hoạch) |
| authoring-service | `internal/adapters/postgres/db.go`, `code_segment_repository.go` | cột `failed_shots`, `SaveSegmentFailed` mới, xoá trường khi chạy/xong/pending |
| authoring-service | `internal/domain/code_segment.go` | `FailedShots`, `Diagnosis`, `SegmentFailure` |
| authoring-service | `internal/application/code_pipeline.go`, `generate_authoring_code.go` | `CodeEvent` mở rộng, `onEvent`, `sendStoredSegments`, `ErrSegmentsIncomplete` |
| authoring-service | `internal/adapters/llm/llm_service_client.go` | đọc trường mới |
| authoring-service | test tương ứng | |
| web-gui | `src/api/client.ts`, `src/components/CodeSegmentsPanel.tsx` (+ `.module.css`, test) | FR-4 |
| docs | `docs/contracts/authoring-llm-code-v2.md` | trường mới của `segment_failed`, đoạn dở trong `segments` |

Không đổi DB của orchestrator, không đổi RabbitMQ. Không cần migrate dữ liệu cũ: cột mới có mặc định. Không thêm biến môi trường.

## Kế hoạch thực hiện (cho `/code`)

1. **run.py**:
   - `ShotFailure(ids, kind, message, tried)` và `ChunkOutcome(shots, failures)`.
   - `write()` trả `ChunkOutcome` theo mục 2 của giải pháp: chạy hết các nửa; `STOP_NOW_KINDS` vẫn ném.
   - `_shot_error_message(failures, saved_ids, max_reasoning_chars)` sinh câu FR-2.
   - Bước dựng `todo`: đoạn có nội dung dở (vân tay khớp, thiếu shot) và `wanted` → `todo` kèm `partial`. `do_chunk` chỉ viết shot thiếu, ghép `partial`.
   - `segment_failed(...)` phát thêm `fingerprint`, `content`, `failed_shots`. Lỗi khung giữ như cũ, `failed_shots` rỗng.
   - Cập nhật docstring đầu module (đoạn "A chunk that ran out of token budget…") cho đúng hành vi mới.
2. **test_pipeline.py** (dùng fake provider sẵn có):
   - `test_a_split_chunk_keeps_the_shots_written_before_one_shot_fails`: nghiệm thu 1.
   - `test_both_halves_run_when_the_first_half_fails`: nghiệm thu 3.
   - `test_a_partly_written_segment_writes_only_its_missing_shots`: nghiệm thu 2.
   - `test_a_stop_now_error_inside_a_split_still_stops_the_run`.
   - Sửa `test_a_one_shot_chunk_over_budget_fails_its_segment_with_budget` và `test_splitting_recurses_down_to_one_shot_on_truncation` cho câu lỗi/sự kiện mới.
3. **authoring-service domain + postgres**:
   - `SegmentFailure`, trường `FailedShots`, cột `failed_shots` trong `db.go` (`ALTER TABLE … ADD COLUMN IF NOT EXISTS`, có comment SQL), `segmentColumns`/`scanSegment`.
   - `SaveSegmentFailed` theo mục 4 (transaction `SELECT … FOR UPDATE` rồi `UPDATE`).
   - Xoá `failed_shots` ở `MarkSegmentRunning`, `SaveSegmentDone`, `ApplySegmentPlan`, `FailRunningSegments`.
   - Test repository nếu có test DB sẵn; nếu không thì test ở tầng application bằng `fakeRepo`.
4. **authoring-service application + client**:
   - `CodeEvent.FailedShots`; `onEvent("segment_failed")` dựng `SegmentFailure`.
   - `sendStoredSegments` gửi đoạn `failed` có nội dung.
   - `ErrSegmentsIncomplete.Error()` theo FR-2.
   - Client parse trường mới.
   - Test: `generate_authoring_code_test.go` (gửi đoạn dở; câu lỗi theo shot), `llm_service_client_test.go` (parse `segment_failed` mở rộng).
5. **web-gui**: `client.ts` types; `CodeSegmentsPanel` theo FR-4; test vitest (đoạn lỗi có nội dung dở hiển thị "Đã lưu: shot …", nút "Chạy lại shot 3.5").
6. **Hợp đồng**: cập nhật `docs/contracts/authoring-llm-code-v2.md`.
7. **Lint/format** theo `docs/code-standards-rules.md` §3: ruff, gofmt + go vet, eslint + prettier + `tsc --noEmit`.
8. **Rebuild** `llm-service`, `authoring-service`, `web-gui`; xác nhận healthy.
9. **Kiểm tra trực tiếp**: dự án `6deec24f…`, bấm "Chạy đoạn này" cho `3.3-3.5`, ghi lại kết quả thật (xem Kiểm tra).

## Kiểm tra

- `llm-service`: `pytest` toàn bộ (`tests/`), ruff sạch.
- `authoring-service`: `go test ./...`, `go vet ./...`, gofmt.
- `web-gui`: `tsc --noEmit`, eslint, vitest.
- Rebuild ba service bằng `docker compose build <svc>` + `docker compose up -d <svc>`, rồi xác nhận healthy.
- Chạy thật trên dự án `6deec24f…`, đoạn `3.3-3.5`:
  - Đoạn này hiện **không** có nội dung dở, vì lượt cũ đã bỏ mất. Lượt chạy lại sẽ viết cả 3 shot.
  - Nếu lại chia tới 3.5 rồi lỗi: kiểm DB `authoring_segments` (`content` có 3.3, 3.4; `failed_shots={3.5}`) và màn hình như FR-4. Bấm chạy lại thì chỉ còn một lượt `chunk 3.5-3.5` trong `llm_usage`.
  - Nếu lần này qua: ghi rõ là chưa quan sát được nhánh lỗi trên dữ liệu thật; nhánh đó chỉ được chứng minh bằng test.

## Rủi ro

- **Chi phí**: không thêm lượt gọi nào. Lượt chạy lại một đoạn dở rẻ hơn hôm nay vì chỉ viết shot thiếu.
- **Nội dung dở + "Sao chép prompt"**: prompt vẫn cho cả đoạn; dán vào sẽ thay cả đoạn, kể cả shot đã lưu. Giữ như hiện nay, ghi trong ngoài phạm vi.
- **Tuần tự hai nửa**: không đổi so với hôm nay (ADR-0030 / CR-048 T2). Chạy song song hai nửa nhanh hơn nhưng chiếm thêm slot; không đổi trong CR này.
- **Dữ liệu cũ**: đoạn `failed` đã lưu không có `failed_shots`. Câu lỗi tổng dùng cách ghi cũ cho chúng, không đoán shot.
