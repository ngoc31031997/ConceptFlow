# CR-056 — Thiết kế: điểm nghẽn bước Code và gần 2 triệu token cho một video lỗi

Trạng thái: **DESIGN — đã duyệt**.

Creator đã chọn:
- FR-2: không cần giữ dự án cũ;
- FR-3: phương án A;
- thêm FR-5: xoá `/v1/code/generate` (task trong backlog).

Service bị ảnh hưởng: `llm-service`, `authoring-service`. `orchestrator`, `rendering`, `web-gui` không đổi.

## Yêu cầu gốc (nguyên văn)

> phân tích dự án Tìm hiểu về AI Agentic? chúng thực sự là gì và đóng vai trò gì trong development tìm điểm nghễn gây lỗi khi render code đi

> phân tích dự án "Tìm hiểu về AI Agentic? chúng thực sự là gì và đóng vai trò gì trong development" tìm điểm nghễn gây lỗi khi render code đi tại sao sau khi tối ưu lại tốn gần 2 triệu token cho 1 video lỗi vậy?

Trả lời đề xuất:

> 1.backlog còn task xoá /v1 có thể làm luôn không?
> 2. FR2 => không cần quan tâm đến các dự án cũ nữa làm gì.
> 3. chọn A
> giai đoạn ở dự án thử nghiêm nên ko cần quan tâm quá nhiều đến dự án cũ nếu chúng tốn qua nhiều effort

## Hiện trạng (đo trên dữ liệu thật và code)

Dự án `92192079-839c-4805-980a-8c98e3ffbf96`, engine Remotion, model `deepseek-ai/deepseek-v4.1-flash`, 3 shot mỗi đoạn, 16 đoạn shot và khung LAYOUT lấy từ storyboard.

### 1. Video không render được vì bước Code chưa xong

Dự án vẫn ở `draft`. Trong `authoring_segments` có 12 đoạn `done` và 4 đoạn `failed`, cả 4 đều lỗi `budget`:

| Đoạn | Lỗi |
|---|---|
| `3.3-3.5` | `chunk 3.3-3.3: hive: budget: model suy nghĩ quá 60000 ký tự mà chưa viết được chữ nào — dừng sớm` |
| `5.2-5.4` | như trên, ở shot `5.2` |
| `7.5-7.7` | như trên, ở shot `7.5` |
| `8.1-8.3` | như trên, ở shot `8.2` |

Khi còn đoạn lỗi, `CodePipeline.run` trả `status="incomplete"` ([run.py:612-614](../../../services/llm-service/app/pipeline/run.py#L612-L614)). Không có file code hoàn chỉnh thì không bao giờ tới bước render. Nên "lỗi khi render" thực chất là **bước Code dừng ở trạng thái chưa đủ đoạn**.

### 2. Token đã tiêu (bảng `llm_usage` của authoring-service)

| Bước / pha | Lượt ok | Lượt lỗi `budget` | Prompt | Completion (trong đó reasoning) | Prompt cache-hit |
|---|---|---|---|---|---|
| story | 1 | 0 | 7 487 | 37 744 (33 337) | 0 |
| storyboard | 1 | 0 | 10 520 | 47 470 (37 285) | 0 |
| illustration | 1 | 0 | 21 665 | 16 235 (13 388) | 0 |
| code / chunk | 19 | 18 | 406 392 | 293 589 (255 200) | 225 536 |
| code / **repair** | **52** | 2 | **1 089 384** | 126 429 (72 696) | 869 120 |
| **Tổng** | | | | | **≈ 2 057 000 token** |

- **Repair chiếm ≈ 1,22 triệu token, tức 59%.** Mỗi lượt repair chỉ sửa một shot và viết ra khoảng 1 000 token, nhưng vẫn gửi đi khoảng 21 000 token prompt. Lý do: `_repair` gọi `_ask` với `ask_req`, mà `ask_req` mang nguyên system prompt ([run.py:489](../../../services/llm-service/app/pipeline/run.py#L489), [run.py:727](../../../services/llm-service/app/pipeline/run.py#L727)). System prompt gồm `remotion_engineer_ai` (28 503 ký tự) cộng mục C4 liệt kê 30 hình thư viện.
- Khoảng 18 000 token trong mỗi prompt là cache-hit: 1,09 triệu trên tổng 1,5 triệu token prompt của bước code. Con số "gần 2 triệu" cộng cả phần cache. Tiền thật phải trả thấp hơn nếu Hive tính giá cache rẻ hơn, điều này tuỳ bảng giá Hive và chưa kiểm.
- **Còn một phần chi phí không hiện ra trong con số trên.** 20 lượt bị ngắt vì `budget` được ghi **0 token** ([provider.py](../../../services/llm-service/app/provider.py), nhánh `aborted`: gói usage luôn đến cuối stream, nên stream bị cắt thì không có số liệu). Thực tế mỗi lượt đó đã gửi khoảng 21 000 token prompt và đã suy nghĩ hơn 60 000 ký tự. Nếu lấy tỉ lệ khoảng 4 ký tự mỗi token (**chưa đo**), 20 lượt này khoảng **0,7 triệu token** mà màn hình không hiện.

### 3. Nguyên nhân gốc của số repair: key PALETTE bị viết thường

282 lỗi kiểm tra của dự án (`code_check_diagnostics`) phân bố như sau:

| Loại | Số lỗi | Số shot | Ví dụ |
|---|---|---|---|
| `TS2551` | **228 (81%)** | 29 | `Property 'conNguoi' does not exist on type '{ nenpanel: string; mohinh: string; … connguoi: string …}'` |
| `safe_area` | 35 | 15 | hình ra ngoài vùng an toàn |
| `TS2304` | 12 | 9 | `Cannot find name 'PALETTE_CONNGUOI_PLACEHOLDER'` (model lúng túng vì chính lỗi trên) |
| `TS2322`, `TS1117` | 7 | 6 | lỗi thường |

Chuỗi gây lỗi:

1. Storyboard đặt role màu kiểu camelCase: `conNguoi`, `nenPanel`, `moHinh`, `hanhDong`, `quanSat`. Trường `visual` của các shot nhắc `conNguoi` hàng chục lần.
2. `merger.palette_keys` đổi role thành key bằng `naming.camel` ([merger.py:51-54](../../../services/llm-service/app/pipeline/merger.py#L51-L54)).
3. `naming.camel` chạy `words[0].lower()`. Role chỉ có một "từ" ASCII như `conNguoi` vì thế thành `connguoi`: chữ hoa ở giữa bị xoá ([naming.py](../../../services/llm-service/app/pipeline/naming.py)). Đã chạy thử trong container: `conNguoi→connguoi`, `nenPanel→nenpanel`, `moHinh→mohinh`.
4. File ghép ra khai báo `PALETTE = { connguoi: … }`. Bảng màu trong prompt ghi `- connguoi = #… — conNguoi: …`, nên có hai tên cho cùng một màu. Model làm theo phần `visual` và viết `PALETTE.conNguoi`, dẫn tới lỗi `TS2551` ở gần như mọi shot có màu này. Mỗi shot lỗi tốn một lượt repair khoảng 21 000 token, có khi tới 3 vòng.

Lỗi này có ở nhiều dự án, không chỉ dự án này:

| Dự án | Role palette | Lượt repair |
|---|---|---|
| Thuyết tiến hoá (`69948fcb`) | `coreGlow, darwinEra, …` (camelCase) | 59 |
| Buffet (`24fa121a`) | `nenPhong, doRe, canhBao, …` (camelCase) | 60 |
| Vũ trụ (`f7103848`) | `vàng khao khát, xanh thành phố, …` (có dấu cách, `camel` xử lý đúng) | 21 |

Prompt `visual_director_ai` chỉ yêu cầu `"role": "<tên vai trò>"`. Trong khi đó key `layout` bắt buộc camelCase, nên model storyboard hay đặt role camelCase luôn.

### 4. Nghẽn thứ hai: trần suy nghĩ 60 000 ký tự nằm giữa mức bình thường

- Trần được đặt ở CR-048 (`CODE_MAX_REASONING_CHARS=60000`, [config.py:72](../../../services/llm-service/app/config.py#L72)) để tránh lượt treo 13 phút.
- Các lượt chunk **thành công** của dự án này dùng 4 300 đến 18 800 token reasoning, phần lớn 13 000 đến 17 000. Với khoảng 4 ký tự mỗi token, mức đó là khoảng 52 000 đến 70 000 ký tự, tức trần 60 000 cắt vào giữa mức bình thường.
- Một đoạn 3 shot bị cắt thì được chia đôi thành 2 shot và 1 shot ([run.py:562-583](../../../services/llm-service/app/pipeline/run.py#L562-L583)). Mỗi nửa lại gửi lại 21 000 token prompt và có thể bị cắt tiếp. 18 lượt chunk bị cắt, mỗi lượt chạy 3 đến 18 phút.
- **Chưa đo được** tỉ lệ ký tự trên token thật, vì `reasoning_chars` chỉ ghi vào log. Log `llm-service` của lượt này đã mất do container khởi động lại. Có thể sự mâu thuẫn `conNguoi`/`connguoi` trong prompt cũng làm model suy nghĩ lâu hơn. Đây là **giả thuyết, chưa chứng minh**.

## Yêu cầu

- **FR-1**: key PALETTE giữ nguyên camelCase đã có trong role.
  - `conNguoi→conNguoi`, `nenPanel→nenPanel`.
  - Các trường hợp cũ không đổi: `màu nhấn→mauNhan`, `Đã xong→daXong`, `vàng khao khát→vangKhaoKhat`.
  - Một từ viết hoa toàn bộ, dài hơn 1 ký tự, thì viết thường (`RED→red`).
  - `unique` và tiền tố `c` cho key bắt đầu bằng số giữ như cũ.
  - Chấp nhận: bảng màu trong prompt chỉ có một tên cho mỗi màu, trùng với tên storyboard dùng.
- **FR-2 (Creator: không cần giữ dự án cũ)**: không thêm bí danh. Để code cũ không bị lặng lẽ dùng lại rồi lỗi `TS2551`, danh sách key PALETTE được đưa vào fingerprint của đoạn shot. Khi đó, đoạn nào viết theo key cũ sẽ bị coi là cũ và được viết lại khi chạy lại. Thay đổi chỉ 1 dòng.
  - Với dự án Agentic: đoạn nào có role camelCase sẽ phải viết lại, tức cả 16 đoạn. Chỉ tốn token khi Creator bấm chạy lại.
- **FR-3 (Creator chọn A)**: mọi lượt gọi LLM ghi số ký tự suy nghĩ đếm thật (`reasoning_chars`) và cờ `usage_reported`.
  - Lượt bị ngắt vì `budget` có `usage_reported=false` và `reasoning_chars>0`, không còn "0 token" im lặng.
  - Không bịa số token.
- **FR-4**: trần 60 000 giữ nguyên trong CR này. Quyết định lại khi có số đo thật từ FR-3: tỉ lệ `reasoning_chars / reasoning_tokens` của các lượt thành công.
- **FR-5 (backlog 2026-09-30, Creator yêu cầu làm luôn)**: bỏ `POST /v1/code/generate` của `llm-service`, cùng đường dự phòng `/v1` trong `authoring-service`.
  - Điều kiện của backlog đã thoả: `authoring-service` đang chạy (main) gọi `/v2/code/generate` trước.
  - Nếu `llm-service` không có `/v2` thì trả lỗi rõ ràng `ErrSegmentsUnsupported` (HTTP 501, đã có sẵn), không âm thầm chạy đường khác.

Ngoài phạm vi (ghi backlog):
- Gộp nhiều shot lỗi của cùng một đoạn vào một lượt repair.
- Thu gọn system prompt 28 500 ký tự.
- Hiện `reasoning_chars` và `usage_reported` trên giao diện. CR này chỉ ghi DB; giao diện đổi phải theo `docs/ux-ui-design-rules.md`, làm CR riêng nếu Creator muốn.
- Đổi `naming.pascal`.
- Tự chạy lại AI cho dự án Agentic.

## Giải pháp

### FR-1: `naming.camel`
Với mỗi từ ASCII:
- nếu viết hoa toàn bộ và dài hơn 1 ký tự thì viết thường;
- từ đầu: `w[0].lower() + w[1:]`;
- các từ sau: `w[0].upper() + w[1:]`.

`camel` chỉ có một chỗ gọi là `merger.palette_keys` ([merger.py:53](../../../services/llm-service/app/pipeline/merger.py#L53)). Prompt manim không dùng bảng màu.

### FR-2: fingerprint
Trong `make_plan` ([run.py:302-304](../../../services/llm-service/app/pipeline/run.py#L302-L304)), `shared` thêm `"palette_keys": merger.palette_keys(sb)`.
- Fingerprint của **mọi** đoạn shot đổi, kể cả dự án có role không camelCase, vì chuỗi `shared` khác.
- Nghĩa là mọi đoạn đã lưu trước CR-056 sẽ phải viết lại khi chạy lại. Creator đã chấp nhận không giữ dự án cũ.
- Đoạn khung (frame) không đổi fingerprint.

### FR-3: `reasoning_chars`, `usage_reported`
- **llm-service**:
  - `errors.Usage` thêm `reasoning_chars: int = 0` và `usage_reported: bool = True`, có trong `to_dict` và `__add__` (cộng `reasoning_chars`, AND `usage_reported`).
  - `provider._once` điền `reasoning_chars` cho mọi kết quả: thành công, `budget`/`truncated`/`empty` sau stream. Nhánh `aborted` đặt `usage_reported=False`.
  - Sự kiện `call` và mọi `usage` khác (chat, suggest, …) mang thêm 2 trường. Thay đổi additive, client cũ bỏ qua được.
- **authoring-service**:
  - `wireUsage` và `application.TokenUsage`/`LLMUsageRecord` thêm 2 trường. JSON thiếu trường thì `usage_reported` mặc định `true`: dùng `*bool` hoặc giá trị mặc định khi decode.
  - Migration idempotent trong `db.go`: `ALTER TABLE llm_usage ADD COLUMN IF NOT EXISTS reasoning_chars INTEGER NOT NULL DEFAULT 0` và `usage_reported BOOLEAN NOT NULL DEFAULT true`.
  - `RecordLLMUsage` ghi 2 cột.
- **Contract**: cập nhật `docs/contracts/authoring-llm-code-v2.md`, phần usage.

### FR-5: bỏ `/v1/code/generate`
- **llm-service**: xoá route `code_generate_v1` và `v2_only_events` ([main.py:294-357](../../../services/llm-service/app/main.py#L294-L357)), cùng các test `/v1/code/generate` trong `tests/test_api.py`. `Call` và import dùng riêng cho v1 bỏ theo nếu không còn chỗ dùng.
- **authoring-service** (`adapters/llm/llm_service_client.go`):
  - `GenerateCode` chỉ gọi `/v2/code/generate`. Gặp `errRouteMissing` thì trả `application.ErrSegmentsUnsupported`, giống cách các lời gọi segment đang làm.
  - Bỏ nhánh `path == "/v1/code/generate"` và trường `Calls` trong kết quả.
  - Bỏ test fallback `/v1`, thay bằng test "404 route → ErrSegmentsUnsupported".
  - Kiểm các test router có nhắc `code/generate` (`code_segments_router_test.go`, `router_test.go`): sửa những test giả lập `/v1`.
- **Tài liệu**:
  - `docs/contracts/README.md` và `authoring-llm-code-v2.md`: ghi `/v1` đã bỏ ở CR-056;
  - ADR-0030: thêm ghi chú "CR-056 bỏ `/v1`";
  - `aidlc-state.md`: gạch dòng backlog.
- Từ nay **phải deploy `llm-service` trước `authoring-service`**, hoặc cùng lúc. Cả hai đều build lại trong CR này.

## Phạm vi

| Service | File | Thay đổi |
|---|---|---|
| llm-service | `app/pipeline/naming.py` | FR-1 |
| llm-service | `app/pipeline/run.py` (`make_plan`) | FR-2 |
| llm-service | `app/errors.py`, `app/provider.py` | FR-3 |
| llm-service | `app/main.py` | FR-5 |
| llm-service | `tests/test_merger.py`, `test_prompts.py`, `test_provider.py`, `test_pipeline.py`/`test_api.py` | test |
| authoring-service | `adapters/llm/llm_service_client.go` | FR-3 (`wireUsage`), FR-5 |
| authoring-service | `application` (`TokenUsage`, `LLMUsageRecord`, chỗ đổi usage → record) | FR-3 |
| authoring-service | `adapters/postgres/db.go`, `llm_usage_repository.go` | FR-3 migration + INSERT |
| docs | `docs/contracts/*`, ADR-0030, `aidlc-state.md` | FR-3, FR-5 |

- `graphify affected`: `camel` không có node duy nhất trong graph. Grep xác nhận chỉ `palette_keys` gọi nó.
- DB: thêm 2 cột vào `llm_usage` của `authoring-service-db`, idempotent, có default nên không mất dữ liệu.
- RabbitMQ: không đổi.

## Kế hoạch thực hiện (cho `/code`)

1. `naming.camel` theo FR-1.
   - Test `tests/test_merger.py`: giữ `mauNhan`/`daXong`; thêm `conNguoi`, `nenPanel`, `RED→red`, key bắt đầu bằng số.
   - `tests/test_prompts.py`: bảng màu `remotion_chunk` có `- conNguoi = `, không có `connguoi`.
2. `make_plan`: thêm `palette_keys` vào `shared`.
   - Test: đổi key (cùng role, thuật toán mới) làm đổi fingerprint đoạn shot, không đổi fingerprint khung.
   - Chạy lại test fingerprint hiện có; sửa các giá trị hash cứng nếu có.
3. `errors.Usage` và `provider._once` theo FR-3.
   - Test `tests/test_provider.py`: stream bị ngắt → `usage_reported=False`, `reasoning_chars>limit`; stream thành công → `reasoning_chars` đúng tổng độ dài `reasoning_content`, `usage_reported=True`; `__add__`.
4. Xoá `/v1/code/generate` và các test của nó (FR-5).
5. `authoring-service`:
   - `wireUsage`, `TokenUsage`, `LLMUsageRecord`, migration, `RecordLLMUsage` (FR-3). Test decode JSON thiếu trường → `usage_reported=true`.
   - `GenerateCode` bỏ fallback, 404 route → `ErrSegmentsUnsupported`; sửa các test.
6. Cập nhật contract, ADR-0030, `aidlc-state.md` (gạch backlog `/v1`, thêm backlog mới ở mục "Ngoài phạm vi").
7. Chạy `pytest` + `ruff` (`llm-service`); `go vet ./...` + `go test ./...` (`authoring-service`).
8. `docker compose build llm-service authoring-service`. `up -d llm-service` trước, rồi `authoring-service`. Xác nhận cả hai healthy.

## Kiểm tra

- Unit test như trên.
- Kiểm trực tiếp, không tốn token:
  - `\d llm_usage` có 2 cột mới.
  - Gọi `/v2/code/segment-prompt` cho đoạn `3.3-3.5` của dự án `92192079…`: bảng màu ghi `conNguoi`.
  - `GET` danh sách đoạn: các đoạn `done` cũ giờ lệch fingerprint (hiện là cần viết lại).
  - `POST /v1/code/generate` trả 404.
- Chạy lại dự án Agentic bằng AI thật: **chỉ khi Creator bấm**.

## Rủi ro

- **Mọi đoạn code đã lưu phải viết lại khi chạy lại.** Creator đã chấp nhận (giai đoạn thử nghiệm).
- Bỏ `/v1`: một `llm-service` cũ không có `/v2` sẽ làm bước Code lỗi 501 thay vì chạy. Cả hai service đều build lại trong CR này.
- FR-1 chưa chắc cứu được 4 đoạn `budget`. Nếu shot đơn vẫn suy nghĩ quá 60 000 ký tự thì đó là vấn đề trần (FR-4), sẽ quyết định bằng số đo từ FR-3.
- Nếu Hive vẫn sinh tiếp sau khi ta đóng stream, lượt bị ngắt tốn hơn số `reasoning_chars` ghi được. Chưa kiểm được.
