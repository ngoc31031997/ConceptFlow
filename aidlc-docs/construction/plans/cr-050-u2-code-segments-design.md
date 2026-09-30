# CR-050 Unit 2 — Thiết kế: bước Code lưu từng đoạn, chạy tiếp khi lỗi, AI ngoài từng đoạn

## Date
2026-09-30

## Trạng thái
Application Design + Functional Design cho Unit 2. **Creator duyệt 2026-09-30 ("ok duyệt").**

Nhánh: `feature/cr-050-llm-call-chunking-resume`. Nhánh đã merge `main` @ `39f1f26` (CR-051..055, chưa commit, chờ duyệt cùng bản thiết kế này).

## Yêu cầu gốc (nguyên văn)
- "kiểm tra lại các tác vụ có gọi api llm xem có hợp lý hay chưa, hiện nay ở đoạn code hay bị ngắt và lỗi, ở đoạn đó có thể chia nhỏ ra để chạy song song hoặc tuần tự để khi có lỗi chỉ cần chạy lại đúng đoạn lỗi ko phải toàn bộ như hiện tại"
- Các quyết định đã duyệt (xem `aidlc-docs/inception/requirements/cr-050-llm-call-chunking-resume.md`):
  - Q2: không tự thử lại.
  - Q3: một đoạn lỗi thì các đoạn khác vẫn chạy; đoạn lỗi chạy riêng được bằng AI hoặc AI ngoài.
  - Q4: lưu từng đoạn trong DB.
  - Q8: mỗi đoạn một kết quả cuối, ghi đè.
  - C2: chỉ bỏ đoạn khi prompt đổi, đổi model vẫn giữ.
  - C3/C3b: số shot mỗi đoạn chỉnh được, mặc định 3.
  - C4: giao diện liệt kê đủ các đoạn.
- 2026-09-30: "tiếp tục feature/cr-050-llm-call-chunking-resume".

Unit 2 gồm: FR-1..FR-9 áp cho **bước Code** (Remotion và Manim), FR-21, FR-22, và NFR-1..NFR-4, NFR-6.
Phần Storyboard theo cảnh (FR-10/11) là Unit 3. Unit 3 dùng lại bảng và cơ chế của unit này.

## Hiện trạng (đã đọc code)
- `llm-service/app/pipeline/run.py:281-467` `CodePipeline.run`:
  1. Khung: LAYOUT lấy từ storyboard, hoặc một lượt AI; Manim thì là cast.
  2. Chia các đoạn `CODE_CHUNK_SHOTS` (biến môi trường, mặc định 3) và chạy song song, tối đa 10.
  3. Mỗi đoạn Remotion được kiểm sớm và sửa riêng (`_settle_chunk`).
  4. Ghép, kiểm cả file, sửa các shot lỗi (`_repair`) tối đa `CODE_REPAIR_MAX_ROUNDS` vòng.
- **Một đoạn lỗi** (`run.py:331-422`): đặt `stopping`, không cho đoạn mới bắt đầu, và cả lượt ném `PipelineFailure`. Đoạn đã xong chỉ còn trong `ChunkCache` trong RAM (`run.py:157`, tạo một lần ở `main.py:172`).
- Khoá cache (`run.py:304,341`) chứa `req.system`, mà `req.system` đã được ghép `library_section` ở `run.py:298`.
- `authoring-service` gọi `POST /v1/code/generate` (`adapters/llm/llm_service_client.go:396`). Nó nhận `calls` **ở cuối** lượt rồi mới ghi `llm_usage` (`application/generate_authoring_code.go:83-96`). Tiến độ chỉ có `chunks_done/total` trong RAM (`generate_authoring_code.go:207`).
- Diagnostic của `rendering` (`application/check_script.py:40`) chỉ có `message/line/kind`. Mã luật (`LayoutFinding.rule`, mã lỗi tsc) bị bỏ khi đổi kiểu (`check_script.py:104,135`).
- Giao diện bước Code (`web-gui/src/pages/ManimEngineerStepPage.tsx`) chỉ có thanh chạy AI và một ô code cho cả file.

## Yêu cầu và tiêu chí nghiệm thu
- **FR-1**:
  - Bước Code có một đoạn khung (`frame`) và các đoạn shot (`1.1-1.3`, …), lưu trong `authoring_segments`.
  - Mỗi đoạn có: trạng thái (`pending|running|done|failed`), nội dung cuối, nguồn (`ai|external|manual|storyboard`), dấu vân tay, lỗi (loại, câu), thời gian.
  - *Nghiệm thu*: sau một lượt, bảng có đủ các đoạn với trạng thái đúng.
- **FR-2**:
  - Đoạn lỗi không dừng các đoạn khác.
  - Ba lỗi vẫn dừng ngay: `auth`, `balance`, `not_configured`. Creator huỷ cũng dừng ngay.
  - *Nghiệm thu*: provider giả lỗi ở đoạn 2/5; các đoạn 1, 3, 4, 5 được lưu `done`, đoạn 2 `failed`; bước kết thúc "chưa xong: 1 đoạn lỗi".
- **FR-3**: không tự thử lại. Riêng lượt hỏng định dạng vẫn được hỏi lại một lần như hiện nay (`EXTRACT_ATTEMPTS=2`); đây là sửa đầu ra, không phải thử lại lỗi mạng.
- **FR-4**:
  - Chạy bước Code: chỉ gọi model cho đoạn chưa `done` hoặc có dấu vân tay lệch.
  - "Chạy lại đoạn này": chỉ đoạn đó.
  - "Sinh lại toàn bộ": xoá mọi đoạn rồi chạy, có hỏi xác nhận.
  - *Nghiệm thu*: chạy lại sau khi đoạn 2 lỗi thì chỉ có lượt `chunk` cho đoạn 2 (đếm qua `llm_usage`).
- **FR-5**:
  - "Sao chép prompt" trả đúng `system + user` của đoạn.
  - "Dán kết quả" kiểm đủ shot và đúng khuôn, rồi lưu `done/external`.
  - "Sửa tay" lưu `done/manual`.
  - Dán sai thì báo lỗi cụ thể, không lưu.
- **FR-6**:
  - Đổi prompt Engineer thì mọi đoạn lệch vân tay.
  - Sửa storyboard thì chỉ đoạn có shot đổi mới lệch; khung đổi thì cả khung và các đoạn lệch.
  - Đổi model thì giữ nguyên.
  - Đoạn lệch được đưa về `pending` khi lượt sau bắt đầu.
- **FR-7**: `project_authoring.code_chunk_shots` (1–10, mặc định 3) chỉnh trên giao diện, trước nút chạy.
- **FR-8**:
  - Khi mọi đoạn `done`, lượt đó tự ghép, kiểm, sửa như hiện nay.
  - Shot được sửa thì ghi đè vào đoạn chứa nó (nguồn giữ nguyên).
  - Code ghép được lưu vào `code_content` như hiện nay.
- **FR-9**:
  - Panel danh sách đoạn trên trang bước Code ở chế độ AI.
  - Mỗi đoạn có trạng thái, lý do lỗi, thời gian và các nút FR-4/FR-5. Đoạn `done` thu gọn được.
- **FR-21**: mỗi lượt gọi có một dòng `llm_usage` ngay khi lượt đó xong, kể cả khi lượt chạy sau đó bị ngắt.
- **FR-22**:
  - Mỗi lần kiểm không đạt (kiểm sớm từng đoạn và kiểm cả file) ghi vào `code_check_diagnostics`: loại, luật, shot, vòng, câu lỗi.
  - Unit này không làm màn thống kê; thống kê bằng SQL, xem phần Kiểm tra.
- **NFR-3**: restart `authoring-service` hoặc `llm-service` không mất đoạn đã xong; đoạn đang chạy thành `failed` với lý do "bị ngắt".
- **NFR-6**: project cũ không có đoạn nào. Code đã lưu giữ nguyên. Lượt AI đầu tiên sau khi nâng cấp sinh mới toàn bộ.

**Ngoài phạm vi**:
- Storyboard theo cảnh (Unit 3).
- Shot Spec (Unit 4).
- Màn thống kê chẩn đoán.
- Nhớ "đoạn này quá lớn" giữa các lượt (xem Rủi ro).

## Giải pháp đề xuất

### Chọn nơi điều phối (ADR-0030)
| | A. `llm-service` điều phối, không trạng thái; `authoring-service` lưu (**khuyến nghị**) | B. `authoring-service` điều phối, `llm-service` chỉ làm thao tác đơn lẻ |
|---|---|---|
| Code đổi | Sửa `run.py` (bỏ cache, không dừng khi một đoạn lỗi, stream kết quả đoạn); thêm bảng và use case bên Go | Viết lại bằng Go phần song song, chia đôi khi hết budget, kiểm sớm, vòng sửa; thêm 3–4 endpoint Python |
| Rủi ro | Thấp: logic sinh, kiểm, sửa đã chạy thật giữ nguyên | Cao: viết lại phần đã ổn định, hai nơi cùng hiểu định dạng shot |
| Số request | Một stream mỗi lượt, như hiện nay | Nhiều request mỗi lượt |

Chọn A, ghi trong `aidlc-docs/decisions/ADR-0030-code-segments-and-v2-code-contract.md`.

### Mô hình dữ liệu (`authoring-service`)
```sql
CREATE TABLE IF NOT EXISTS authoring_segments (
    project_id    TEXT NOT NULL,
    step          TEXT NOT NULL,             -- 'code' (Unit 3 thêm 'storyboard')
    key           TEXT NOT NULL,             -- 'frame' | '1.1-1.3'
    position      INTEGER NOT NULL,          -- thứ tự hiển thị
    kind          TEXT NOT NULL,             -- 'frame' | 'shots'
    shots         TEXT[] NOT NULL DEFAULT '{}',
    status        TEXT NOT NULL DEFAULT 'pending',  -- pending|running|done|failed
    source        TEXT NOT NULL DEFAULT '',  -- ai|external|manual|storyboard
    fingerprint   TEXT NOT NULL DEFAULT '',
    content       JSONB,                     -- frame: {"code": "..."}; shots: {"shots": {"1.1": "..."}}
    error_kind    TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    duration_ms   INTEGER NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, step, key)
);
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS code_chunk_shots INTEGER NOT NULL DEFAULT 3;

CREATE TABLE IF NOT EXISTS code_check_diagnostics (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    project_id  TEXT NOT NULL,
    engine      TEXT NOT NULL,
    phase       TEXT NOT NULL,               -- 'chunk' (kiểm sớm) | 'final'
    round       INTEGER NOT NULL,
    segment_key TEXT NOT NULL DEFAULT '',
    shot_id     TEXT NOT NULL DEFAULT '',    -- '' = khung hoặc không gắn được shot
    kind        TEXT NOT NULL,               -- compile | layout
    rule        TEXT NOT NULL DEFAULT '',    -- vd. safe_area, subtitle_zone, TS2322
    message     TEXT NOT NULL,
    line        INTEGER
);
CREATE INDEX IF NOT EXISTS code_check_diagnostics_created_idx ON code_check_diagnostics (created_at DESC);
```
- Không có khoá ngoại, theo cùng lối với `project_illustrations`. `DeleteAuthoring` xoá đoạn của project.
- `code_check_diagnostics` được giữ khi xoá project, giống `llm_usage`, để thống kê.

### Dấu vân tay (tính ở `llm-service`, `authoring-service` chỉ so bằng nhau)
- `P = sha256(system)`: `system` là prompt `*_engineer_ai` đã render, **trước** khi ghép phần hình thư viện. Đây là "prompt của bước" (C2).
- Khung Remotion lấy từ storyboard: `sha256("frame-sb", layout)`, nguồn `storyboard`, không gọi AI.
- Khung AI: `sha256("frame", engine, P, storyboard)`.
- Đoạn shot: `sha256("shots", engine, P, vân tay khung, {hero, world, palette}, JSON chuẩn hoá các shot của đoạn)`. Xem mục "Thay đổi khi code".
- Không chứa: model (C2), hình thư viện (sửa lỗi C3 của review), shot liền trước/liền sau.
  - Shot kề chỉ là gợi ý nối cảnh; nếu đưa vào thì sửa một shot kéo theo ba đoạn.

### Luồng một lượt bước Code
1. `runCode` (Go):
   - Kiểm hình minh hoạ như hiện nay.
   - Đọc `code_chunk_shots` và các đoạn đã lưu.
   - Với "Sinh lại toàn bộ" thì xoá đoạn trước khi gọi.
   - Gọi `POST /v2/code/generate`, gửi các đoạn `done` (khoá, vân tay, nội dung) và `only` (khi chạy lại một đoạn).
2. `llm-service` phát sự kiện `plan` với danh sách đoạn và dấu vân tay mới. `authoring-service` áp plan:
   - xoá khoá không còn;
   - thêm khoá mới ở trạng thái `pending`;
   - đoạn lệch vân tay về `pending` và xoá nội dung;
   - ghi `position`.
3. `llm-service` chạy các đoạn cần chạy. Với `only`: khung chỉ chạy khi nằm trong `only`; khung chưa `done` thì mọi đoạn shot không chạy được (báo "chạy đoạn khung trước").
   - `segment_start`: đoạn thành `running`.
   - `segment_done`: đoạn thành `done`, lưu nội dung và vân tay.
   - `segment_failed`: đoạn thành `failed` với lỗi.
   - `call`: ghi `llm_usage` ngay (FR-21).
   - `check`: ghi `code_check_diagnostics` (FR-22).
4. Khi mọi đoạn `done`, `llm-service` ghép, kiểm và sửa như hiện nay. Mỗi đoạn có shot được sửa phát lại `segment_done` (`repaired: true`), rồi phát `result` `status=done` cùng code.
   - `authoring-service` lưu `code_content` như hiện nay: `check_failed` vẫn lưu và gắn cờ.
5. Nếu còn đoạn lỗi hoặc chưa chạy, `result` là `status=incomplete` kèm `failed`/`pending`.
   - `runCode` trả `ErrSegmentsIncomplete`. HTTP 422, câu tiếng Việt: "Bước Code chưa xong: 2/18 đoạn lỗi (1.4-1.6: timeout; …). Các đoạn đã xong đã được lưu — chạy lại đoạn lỗi hoặc dán kết quả AI ngoài."
   - Chuỗi tự chạy dừng ở trạng thái `Failed` với câu đó. `code_content` cũ giữ nguyên.
6. Stream đứt (restart `llm-service`, huỷ, timeout): `FailRunning(project, "code", kind, câu)` đưa đoạn `running` về `failed` với lỗi "bị ngắt" hoặc "đã huỷ".
7. Khởi động `authoring-service`: `FailAllRunning` ("bị ngắt khi authoring-service khởi động lại") (NFR-3).

### Thay đổi trong `run.py` (`llm-service`)
- Bỏ `ChunkCache` cùng mọi `cache_key`/`used_keys`/`split_key`.
- `CodeRequest` thêm:
  - `chunk_shots: int` (0 = dùng cấu hình);
  - `done: dict[key, (fingerprint, content)]`;
  - `only: set[str] | None`.
- **Plan**:
  - `ordered` và `chunks` như cũ, nhưng chia theo `chunk_shots` của request.
  - Khoá đoạn là `f"{ids[0]}-{ids[-1]}"`; đoạn một shot là `"1.1-1.1"`.
  - Tính vân tay xong thì phát `plan`.
- **Khung**:
  - Dùng lại nội dung `done` nếu vân tay khớp.
  - Nếu không, chạy khi `only is None or "frame" in only`.
  - Khung lỗi: phát `segment_failed`, các đoạn shot không chạy, trả `incomplete`.
- **Đoạn shot**:
  - `do_chunk` không còn `stopping` cho lỗi thường. Lỗi chỉ phát `segment_failed`, các đoạn khác chạy tiếp.
  - `STOP_NOW_KINDS` và huỷ vẫn huỷ mọi task và ném lỗi (stream `error`).
  - Chia đôi khi `budget`/`truncated` giữ nguyên, trong phạm vi một đoạn.
  - Kiểm sớm Remotion (`_settle_chunk`) giữ nguyên và chạy trước `segment_done`.
- **Gọi model**: mọi `Call` được tạo đi qua `_record(call)`, hàm này vừa thêm vào danh sách vừa phát `{"type":"call", "segment": key, ...}`. `Call.cached` bị bỏ.
- **Kiểm**:
  - Mỗi lần kiểm không đạt phát `{"type":"check", "phase":"chunk"|"final", "round", "segment", "diagnostics":[{message,line,kind,rule,shot}]}`.
  - `shot` lấy từ `merged.shot_at(line)`.
- **Ghép**: chỉ khi mọi đoạn `done`. Sau `_repair`, đoạn chứa shot đổi phát lại `segment_done` (repaired).
- **Hàm mới dùng chung**:
  - `segment_prompt(req, key) -> (system, user)`: dùng các `prompts.remotion_chunk/manim_chunk/remotion_layout/manim_cast` hiện có, không có `problem`.
  - `parse_segment(req, key, reply) -> content`: `_parse_layout`/`_parse_cast`/`_parse_shots`.
- `main.py`:
  - Bỏ `/v1/code/generate` và `cache`.
  - Thêm `/v2/code/generate` (stream), `/v2/code/segment-prompt`, `/v2/code/segment-parse` (JSON).
  - `CodeBody` thêm `chunk_shots`, `segments`, `only` (cùng `key`, `reply` cho hai route sau).

### Thay đổi ở `rendering`
- `CheckDiagnostic` thêm `rule: str = ""`:
  - lỗi bố cục lấy `f.rule` (`check_script.py:135`);
  - lỗi tsc lấy `d.code` (`check_script.py:104`);
  - lỗi lint lấy mã luật nếu kiểu issue có, nếu không để rỗng;
  - Manim để rỗng.
- `check_server.py:109` trả thêm `rule`. Chỉ thêm trường: `llm-service` phiên bản cũ bỏ qua nó.
- `llm-service/app/pipeline/checker.py`: `Diagnostic` thêm `rule`, đọc `d.get("rule", "")`.

### API `authoring-service` (qua gateway, dưới `/v1/projects/{id}/authoring/**`, gateway không cần sửa)
| Route | Việc |
|---|---|
| `POST .../authoring/code/generate` (đã có) | Body tuỳ chọn `{"segment": "<key>"}` để chạy lại một đoạn, hoặc `{"fresh": true}` để sinh lại toàn bộ. Không có body thì chạy các đoạn còn thiếu (cả chuỗi tự chạy). |
| `GET .../authoring/code/segments` | `{chunk_shots, segments: [{key, kind, shots, status, source, error_kind, error_message, duration_ms, updated_at, content}]}` |
| `GET .../authoring/code/segments/{key}/prompt` | Render prompt Engineer rồi gọi `/v2/code/segment-prompt`; trả `{system, user}`. 409 nếu khung chưa `done`. |
| `PUT .../authoring/code/segments/{key}` | Body `{reply, source: "external"\|"manual"}`. Gọi `/v2/code/segment-parse`, lưu `done`. 422 kèm lý do nếu sai khuôn. 409 khi bước Code đang chạy. |
| `PUT .../authoring/code/chunk-shots` | Body `{chunk_shots: 1..10}`. 409 khi đang chạy. |

Mọi thao tác ghi lấy cùng khoá `acquire(project, "code")`, nên không chen được vào một lượt đang chạy (`ErrGenerateBusy`, 409).

### Giao diện (`web-gui`), theo `docs/ux-ui-design-rules.md`
- Trang bước Code, chế độ AI, sắp từ trên xuống:
  1. Ô "Số shot mỗi đoạn" nằm cạnh bộ chọn model, **trước** nút chạy (luật 1).
  2. Thanh chạy AI đã có.
  3. **Panel "Các đoạn code"** (`CodeSegmentsPanel`).
  4. Ô code của cả file như hiện nay.
- `CodeSegmentsPanel`:
  - Tóm tắt "12/18 đoạn xong · 2 lỗi · 4 chưa chạy".
  - Mỗi đoạn một hàng:
    - tên ("Khung (LAYOUT)" / "Khung (cast)" / "Shot 1.1–1.3");
    - huy hiệu trạng thái (Chờ / Đang chạy / Xong / Lỗi, dùng màu `--run`/`--success`/`--danger`);
    - nhãn nguồn (AI / AI ngoài / Sửa tay / Từ storyboard);
    - thời gian; dòng lý do lỗi.
  - Nút theo hàng:
    - "Chạy lại đoạn này";
    - "Sao chép prompt";
    - "Dán kết quả", mở `Disclosure` với ô dán và nút Lưu;
    - "Sửa tay" (đoạn đã xong), mở ô soạn có sẵn nội dung.
  - Đoạn `done` thu gọn; mở ra thì xem nội dung (`--font-mono`). Mở/đóng dùng `Disclosure`/`usePresence` (luật 3).
  - Cuối panel, sau danh sách (luật 1):
    - "Chạy các đoạn còn thiếu (N)": gọi generate không body;
    - "Sinh lại toàn bộ": hộp xác nhận, rồi `{"fresh": true}`.
  - Khi đang chạy, panel tải lại danh sách 2 giây một lần (cùng nhịp poll tiến độ), và các nút ghi bị khoá.
- Chế độ thủ công (copy prompt cả file) không đổi.

### Hợp đồng
Tạo `docs/contracts/authoring-llm-code-v2.md` với body, sự kiện NDJSON (`plan`, `segment_start`, `segment_done`, `segment_failed`, `call`, `check`, `phase`, `result`, `error`) và mã lỗi. Sửa dòng tương ứng trong `docs/contracts/README.md`.

## Phạm vi
- **`llm-service`**: `app/pipeline/run.py`, `app/pipeline/checker.py`, `app/main.py`, `app/config.py` (`CODE_CHUNK_SHOTS` chỉ còn là mặc định), `tests/test_pipeline.py`, `tests/test_main*.py` (test route code).
- **`rendering`**: `application/check_script.py`, `adapters/http/check_server.py`, cùng test tương ứng.
- **`authoring-service`**:
  - `adapters/postgres/db.go`, `adapters/postgres/code_segment_repository.go` (mới), `adapters/postgres/check_diagnostic_repository.go` (mới), `adapters/postgres/prompt_template_repository.go` (`DeleteAuthoring`);
  - `application/code_pipeline.go` (kiểu mới), `application/generate_authoring_code.go`, `application/generate_authoring.go` (`Execute` nhận tuỳ chọn), `application/code_segments.go` (mới: list/prompt/paste/chunk-shots);
  - `adapters/llm/llm_service_client.go`, `adapters/http/router.go`, `cmd/authoring/main.go` (nối dây và dọn đoạn `running` khi khởi động).
- **`web-gui`**: `api/client.ts`, `components/CodeSegmentsPanel.tsx` (+ `.module.css`, mới), `pages/ManimEngineerStepPage.tsx`, test vitest.
- **Tài liệu**: ADR-0030, `docs/contracts/authoring-llm-code-v2.md`, `docs/contracts/README.md`, `aidlc-docs/decisions/README.md`.
- **Không đổi**: RabbitMQ, orchestrator, gateway, api `/v1/check/*` (chỉ thêm trường).
- Phụ thuộc cần kiểm bằng `graphify affected` trước khi sửa: `CodePipelinePort`, `CodeGenResult`, `CodeCall`, `recordCall`, `GenerateAuthoringUseCase.Execute` (chuỗi tự chạy, HTTP), `CodePipeline.run`, `ChunkCache`.

## Kế hoạch thực hiện (cho `/code`)
1. **rendering**:
   - Thêm `rule` vào `CheckDiagnostic` và phần trả của `check_server`.
   - Test: một lỗi bố cục trả `rule="safe_area"`; một lỗi tsc trả `rule="TS…"`.
2. **llm-service, `checker.py`**: `Diagnostic.rule`; đọc `rule`.
3. **llm-service, `run.py`** (theo mục "Thay đổi trong `run.py`"):
   - Plan, vân tay, `done`/`only`, không dừng khi một đoạn lỗi, `_record` phát `call`, sự kiện `check`, `segment_*`, `result.status`.
   - Bỏ `ChunkCache`.
   - Thêm `segment_prompt`, `parse_segment`.
4. **llm-service, `main.py`**: route `/v2/code/*`, bỏ `/v1/code/generate`. `_stream` giữ nguyên.
5. **llm-service, test** (`pytest`):
   - plan và vân tay: đổi model thì giữ; đổi system thì đổi; đổi hình thư viện thì giữ; sửa một shot thì chỉ đoạn đó đổi;
   - đoạn 2/5 lỗi thì 4 đoạn còn lại `done` và `status=incomplete`;
   - `only` chỉ chạy đúng đoạn; `done` khớp thì không gọi model;
   - `auth` dừng ngay;
   - mỗi `Call` có đúng một sự kiện `call`;
   - repair phát lại `segment_done`;
   - `segment_prompt`/`parse_segment`: thiếu shot bị báo lỗi.
   - Sửa các test cũ còn dựa vào `ChunkCache`.
6. **authoring-service, DB và repo**:
   - DDL ở trên.
   - `CodeSegmentRepository`: `List`, `ApplyPlan`, `MarkRunning`, `SaveDone`, `SaveFailed`, `FailRunning(project, step)`, `FailAllRunning`, `Delete`.
   - `CheckDiagnosticRepository.Insert`.
   - `DeleteAuthoring` xoá đoạn.
   - `GetCodeChunkShots`/`SaveCodeChunkShots`.
   - Test Postgres (`TEST_DATABASE_URL`).
7. **authoring-service, application**:
   - `CodePipelinePort.GenerateCode` nhận `CodeGenRequest` mới (`ChunkShots`, `Done []CodeSegment`, `Only []string`) và `onEvent` với sự kiện mới. `CodeGenResult` thêm `Status`, `Failed`, `Pending` và bỏ `Calls`.
   - Thêm `SegmentPrompt` và `ParseSegment` vào port.
   - `runCode` xử lý sự kiện như "Luồng một lượt", ghi usage và chẩn đoán ngay, và trả `ErrSegmentsIncomplete` khi chưa xong.
   - `Execute(ctx, projectID, step)` giữ nguyên chữ ký cho chuỗi tự chạy; thêm `ExecuteCode(ctx, projectID, CodeRunOptions{Segment, Fresh})`, dùng chung `run`.
   - `CodeSegmentsUseCase`: `List`, `Prompt`, `Paste`, `SetChunkShots`.
   - Test với pipeline giả: không body thì chỉ chạy đoạn còn thiếu; lỗi một đoạn; stream đứt thì đoạn `running` thành `failed`; usage ghi theo từng sự kiện; `fresh` xoá trước.
8. **authoring-service, adapter và HTTP**:
   - Client `/v2/code/*`.
   - Route mới, body tuỳ chọn cho generate.
   - `DescribeGenerateError` thêm `ErrSegmentsIncomplete` (422).
   - `main.go` gọi `FailAllRunning` lúc khởi động.
   - `router_test.go` cho các route mới.
9. **web-gui**:
   - `client.ts`: `getCodeSegments`, `getCodeSegmentPrompt`, `putCodeSegment`, `putCodeChunkShots`, và `generateAuthoring(step, opts)`.
   - `CodeSegmentsPanel` và ô số shot mỗi đoạn trên `ManimEngineerStepPage`.
   - Vitest: hiển thị các trạng thái; nút chạy lại gửi `{segment}`; "Sinh lại toàn bộ" hỏi xác nhận; dán sai hiện lỗi 422; nút bị khoá khi đang chạy.
10. **Tài liệu**:
    - Thêm dòng ADR-0030 vào `decisions/README.md`.
    - Tạo `docs/contracts/authoring-llm-code-v2.md` và sửa `docs/contracts/README.md`.
    - Cập nhật trạng thái U2 trong `cr-050-workflow-plan.md` và `aidlc-state.md`.
11. **Build**: `go test ./...`, `pytest` (llm-service, rendering), `npx vitest run`, `tsc --noEmit`; rebuild `rendering`, `llm-service`, `authoring-service`, `web-gui`.

## Kiểm tra
- **Test tự động**: như bước 5, 6, 7, 8, 9. Test Postgres chạy với DB tạm (`TEST_DATABASE_URL`).
- **Rebuild và kiểm healthy**: `rendering`, `llm-service`, `authoring-service`, `web-gui`.
- **Kiểm trực tiếp**, cần Creator đồng ý vì tốn token thật (một video 2–3 cảnh là đủ):
  1. chạy bước Code;
  2. trong lúc chạy, restart `llm-service`, rồi xác nhận: các đoạn đã xong còn trong panel, đoạn đang chạy báo "bị ngắt", và `llm_usage` có dòng cho mọi lượt đã xong;
  3. bấm chạy các đoạn còn thiếu, rồi xác nhận chỉ đoạn thiếu có lượt `chunk` mới;
  4. thử "Sao chép prompt" và "Dán kết quả" cho một đoạn.
- **Truy vấn thống kê mẫu (FR-22)**: `SELECT rule, kind, count(*) FROM code_check_diagnostics WHERE created_at > now() - interval '30 days' GROUP BY 1,2 ORDER BY 3 DESC;`

## Rủi ro
- ~~**Hai phía phải deploy cùng lúc.**~~ Đã bỏ ràng buộc này, xem mục "Thay đổi khi code" điểm 6. Bỏ `/v1/code/generate` nghĩa là `authoring-service` bản cũ gọi `llm-service` bản mới sẽ nhận 404. Phải rebuild cùng lúc, và không có lượt Code nào đang chạy khi rebuild.
- **Sửa tay cả file code rồi chạy lại AI** thì code ghép từ các đoạn sẽ ghi đè bản sửa. Hành vi này giống hiện nay: lượt AI luôn ghi đè. Panel sẽ nói rõ "code sẽ được ghép lại từ các đoạn".
- **Manim, hoặc Remotion không có `layout` trong storyboard**: mọi lần sửa storyboard đều sinh lại khung, nên sinh lại mọi đoạn (ADR-0030).
- **Đổi số shot mỗi đoạn** làm đổi khoá, nên các đoạn không còn khoá trong cách chia mới sẽ bị bỏ. Giao diện cảnh báo trước khi lưu nếu đã có đoạn `done`.
- **Không còn cache "đoạn quá lớn"** giữa các lượt: một đoạn hết budget ở lượt trước sẽ thử lại cả khối một lần trước khi chia đôi.
- **Payload**: mỗi lượt gửi lại toàn bộ đoạn đã xong (khoảng 130 KB với 52 shot). Nếu vượt giới hạn body mặc định của FastAPI/uvicorn thì phải nâng giới hạn; cần kiểm khi code.
- Khi merge `main`, có một chỗ lệch chữ ký phải sửa: `summaries_cr051_test.go` gọi `MarkIllustrationsPlanned` theo chữ ký cũ. Đã sửa trong merge đang chờ commit.

## Thay đổi khi code (2026-09-30)
Những điểm dưới đây lệch so với bản thiết kế đã duyệt:
1. **Gateway phải sửa.** Thiết kế ghi "gateway không cần sửa", nhưng `api-gateway/src/routes/projects.js` liệt kê từng route authoring một.
   - Đã thêm 4 route: `GET .../code/segments`, `GET .../code/segments/:key/prompt`, `PUT .../code/segments/:key`, `PUT .../code/chunk-shots`, kèm test định tuyến.
   - Creator đã chọn phương án này.
2. **Chạy lại một đoạn và sinh lại toàn bộ đi qua chuỗi server.**
   - `POST .../authoring/chain` nhận thêm `{"segment": key}` hoặc `{"fresh": true}` với `steps: ["code"]` (`AuthoringChainRunner.StartWith`).
   - Lý do: một request `generate` trực tiếp từ trình duyệt sẽ bị huỷ khi đóng tab hoặc mất mạng. Chuỗi server thì chạy độc lập với trình duyệt, và dùng lại được thanh tiến độ cùng nút Dừng.
   - `POST .../code/generate` vẫn nhận body như thiết kế.
   - Creator đã chọn phương án này.
3. **Vân tay đoạn shot dùng vân tay khung, không dùng nội dung khung.**
   - Nếu dùng nội dung, lần sửa LAYOUT ở vòng repair sẽ làm mọi đoạn bị coi là cũ ở lượt sau.
   - Đưa thêm hero/world/palette vào vân tay, vì prompt đoạn có đọc chúng (FR-1: vân tay gồm mọi phần đầu vào của đoạn).
   - Hệ quả: chạy lại riêng đoạn khung (cùng vân tay) không tự đẩy các đoạn shot về trạng thái chờ. Code của chúng được kiểm ở bước ghép, và vòng sửa xử lý phần lệch LAYOUT.
4. **Danh sách đoạn hiện ngay cả trước lượt chạy đầu.**
   - `authoring-service` chia đoạn bằng Go (`domain.PlanCodeSegments`), theo đúng cách chia của `llm-service`, để panel liệt kê đủ đoạn và cho phép dán ngay (C4).
   - Vân tay vẫn chỉ do `llm-service` tính.
5. **Nút "Chạy các đoạn còn thiếu" vẫn bấm được khi không còn đoạn thiếu.** Nhãn đổi thành "Ghép và kiểm lại code", vì lượt chạy đó chỉ ghép, kiểm và sửa (FR-8). Trường hợp này xảy ra chẳng hạn khi mọi đoạn đều được dán từ AI ngoài.
6. **Hai service deploy độc lập** (Creator yêu cầu 2026-09-30, sau lần báo "code xong" đầu tiên):
   - `llm-service` giữ `/v1/code/generate` (hành vi cũ, chạy trên pipeline mới).
   - `authoring-service` quay về `/v1` khi `llm-service` chưa có `/v2`.
   - Thêm `/v2/code/plan`; bỏ bản chép cách chia đoạn bằng Go (`domain.PlanCodeSegments`, `StoryboardHasLayout`) nêu ở điểm 4. Panel lấy danh sách đoạn từ `llm-service`.
   - Panel báo 501 khi `llm-service` chưa hỗ trợ.
   - Quy tắc đổi contract được ghi vào `docs/contracts/README.md`.
   - Bỏ `/v1`: mục backlog trong `aidlc-state.md`.
