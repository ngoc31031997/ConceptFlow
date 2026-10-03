# authoring-service → llm-service: bước Code theo đoạn (`/v2/code/*`)

CR-050 Unit 2, [ADR-0030](../../aidlc-docs/decisions/ADR-0030-code-segments-and-v2-code-contract.md). `/v1/code/generate` đã bỏ ở CR-056 (xem mục cuối).

`llm-service` không lưu gì. `authoring-service` lưu từng đoạn (bảng `authoring_segments`) và gửi lại các đoạn đã xong trong mỗi lượt.

Route thật nằm ở `services/llm-service/app/main.py`. Client nằm ở `services/authoring-service/internal/adapters/llm/llm_service_client.go`.

## Đoạn và dấu vân tay
- **Khoá**: `frame` cho khung (LAYOUT với Remotion, `setup_cast` với Manim), `"<shot đầu>-<shot cuối>"` cho mỗi nhóm `chunk_shots` shot, theo thứ tự storyboard qua các cảnh. Đoạn một shot có khoá `1.1-1.1`.
- **Vân tay** do `llm-service` tính; `authoring-service` chỉ so bằng nhau:
  - khung Remotion lấy từ `layout` của storyboard: `sha256("frame-sb", layout)`;
  - khung AI: `sha256("frame", engine, sha256(system), storyboard)`;
  - đoạn shot: `sha256("shots", engine, sha256(system), vân tay khung, {hero, world, palette}, JSON các shot của đoạn)`.
- Vân tay **không** chứa model, danh sách hình thư viện, hay shot liền trước/liền sau.
- `system` là prompt `*_engineer_ai` đã render, trước khi `llm-service` ghép phần hình thư viện.

## Body chung
```json
{
  "engine": "remotion | manim",
  "topic": "…",
  "storyboard": "<JSON storyboard chuẩn hoá>",
  "system": "<prompt *_engineer_ai đã render>",
  "model": "",
  "max_tokens": 0,
  "illustrations": [{"name": "…", "usage": "…", "description": "…", "code": "…"},
                    {"name": "MeadowBackdrop", "description": "…", "code": "", "kind": "backdrop", "shots": ["1.1", "1.2"]}],
  "frame": {"width": 1920, "height": 1080},
  "subtitle_band": {"edge": "top | bottom", "px": 240},
  "video_font": "",
  "chunk_shots": 3,
  "segments": [{"key": "frame", "fingerprint": "…", "content": {"code": "…"}}],
  "only": ["1.4-1.6"]
}
```
- `chunk_shots`: từ 0 đến 10; 0 nghĩa là dùng `CODE_CHUNK_SHOTS`.
- `segments`: các đoạn `done` đã lưu, và các đoạn shot `failed` có **nội dung dở** (thiếu một số shot). Nội dung khung là `{"code": "…"}`, nội dung đoạn shot là `{"shots": {"1.1": "…"}}`. Với đoạn dở có vân tay khớp, `llm-service` chỉ viết các shot còn thiếu rồi trả `segment_done` với đủ shot.
- `only`: nếu có thì chỉ chạy các đoạn này; bỏ trống thì chạy mọi đoạn còn thiếu.
- `subtitle_band` và `video_font` chỉ dùng cho Remotion. `subtitle_band.px` nhỏ hơn chiều cao khung lớn nhất (1920).
- `frame`: khung video được dựng, chỉ nhận `1920×1080` (video dài) hoặc `1080×1920` (short dọc); không có thì là khung ngang. Merger viết `<Composition>` đúng kích thước này, storyboard được kiểm toạ độ `layout` theo nó, prompt LAYOUT/sửa lỗi nêu đúng vùng an toàn của nó. Khung dọc thêm kích thước khung vào vân tay khung (nên mọi đoạn được viết lại khi đổi khung); khung ngang giữ nguyên vân tay cũ. Rendering đọc khung từ `<Composition>` của script khi kiểm bố cục, không cần trường riêng. Xem ADR-0031.
- `illustrations[].kind`: `"figure"` (mặc định) hoặc `"backdrop"`. Một nền mang `shots` (các shot dựng trong nền đó); nền dựng sẵn trong kit có `code` rỗng. Prompt liệt kê nền theo shot (mục C5); merger dán code của nền thư viện khi shot dùng nó qua `backdrop={Tên}`.
- Storyboard: mỗi cảnh có thể có `setting` (bối cảnh kín khung). Có `setting` thì nó vào JSON của shot dưới tên `scene_setting`; không có thì JSON shot (và vân tay đoạn) như cũ.
- Storyboard: mỗi shot có thể có `lines: [{"say", "show"}]` (1–4 câu thoại, mỗi `say` ≤ 30 từ). Khi có, `narration` của shot do llm-service ghép từ các `say` lúc chuẩn hoá, và `lines` vào JSON của shot; không có thì shot là một câu (`narration`) và JSON shot như cũ. Xem ADR-0032.

## Khung script Remotion do merger viết
- `export const narrations: string[]` — **mỗi câu thoại một dòng** (các câu của mọi shot, theo thứ tự). Mỗi dòng là một đoạn TTS riêng và một mốc `wait_offsets` riêng.
- `export const shotLineCounts: number[]` — số câu của từng shot, theo thứ tự `SHOTS`; tổng bằng số dòng `narrations`. Rendering dùng nó để gom câu thành đoạn của shot; script không có nó thì mỗi dòng là một shot (script cũ, luồng thủ công).
- `type ShotProps = {duration: number; lines: number[]}` — `lines[i]` là frame bắt đầu câu i, tính từ đầu shot (`lines[0] = 0`). `CreatorComposition` truyền `segment.lines`, hoặc `evenLines(duration, shotLineCounts[i])` khi đoạn không có mốc thật (layout probe).
- Shot stub của một lượt kiểm một lô: `function ShotN_M({duration, lines}: ShotProps) { return null; }`.

## `POST /v2/code/generate` (NDJSON)
Sự kiện theo thứ tự thời gian. Mỗi dòng là một JSON có trường `type`.

| `type` | Trường | Bên nhận làm gì |
|---|---|---|
| `plan` | `segments: [{key, kind: "frame"\|"shots", shots, fingerprint}]` | Xoá đoạn có khoá không còn trong plan; thêm đoạn mới ở trạng thái `pending`; đoạn lệch vân tay về `pending` và xoá nội dung. |
| `segment_start` | `key` | Đoạn thành `running`. |
| `segment_done` | `key, fingerprint, content, source ("ai"\|"storyboard"), repaired, duration_ms` | Lưu `done`. Với `repaired: true`, giữ nguyên `source` và `duration_ms` cũ. |
| `segment_failed` | `key, failed_shots, error: {kind, message, …}`; đoạn shot thêm `fingerprint, content` | Lưu `failed` và `failed_shots`. `content` (`{"shots": …}` các shot đã viết, `null` khi không có) được lưu làm nội dung dở, trừ khi nội dung đang lưu cùng vân tay đã đủ mọi shot (lượt chạy lại lỗi không vứt kết quả đầy đủ). Không có `content` thì giữ nội dung cũ. |
| `call` | `phase, label, segment, ok, usage, error_kind, error_message, duration_ms` | Ghi một dòng `llm_usage` ngay. `usage` = `{model, prompt_tokens, completion_tokens, reasoning_tokens, cached_tokens, reasoning_chars, usage_reported}`; hai trường cuối thêm ở CR-056 (xem dưới). `label` của lượt `repair` là id các shot được sửa trong lượt, nối bằng dấu phẩy (`1.3,1.4`) khi một lượt sửa nhiều shot của cùng một đoạn; một shot thì chỉ có id đó. |
| `check` | `phase ("chunk"\|"final"), round, segment, diagnostics: [{message, line, kind, rule, shot, segment}]` | Ghi vào `code_check_diagnostics`. |
| `phase`, `chunk_start`, `chunk_done`, `chunk_split`, `chunk_repair` | như CR-039/CR-048 | Chỉ dùng cho thanh tiến độ. |
| `result` | `status ("done"\|"incomplete"), code, check_ok, diagnostics, repair_rounds, warnings, scene_class_name, failed, missing` | `done`: lưu code đã ghép. `incomplete`: bước kết thúc "chưa xong". |
| `error` | `error: {kind, message, …}`, `calls: []` | Lỗi dừng cả lượt: `auth`, `balance`, `not_configured`, storyboard không đọc được, lỗi bất ngờ. Không ghi lại `calls`, vì mọi lượt gọi đã có sự kiện `call` riêng. |

- **Lỗi một đoạn** không dừng các đoạn khác. Lượt chạy kết thúc với `result.status = "incomplete"` và `failed` liệt kê các đoạn lỗi.
- **Chia đôi**: đoạn lỗi `budget`/`truncated` được viết lại thành hai nửa, đệ quy tới một shot; nửa nào lỗi thì nửa kia vẫn chạy. `segment_failed` của đoạn đó có `failed_shots` là các shot không viết được, `content` là các shot đã viết, và `error.message` (tiếng Việt) nêu shot, lý do, các lượt đã thử và các shot đã lưu, ví dụ: `Shot 3.5: model suy nghĩ quá 60000 ký tự mà chưa viết được chữ nào (đã thử cả đoạn 3.3-3.5, rồi riêng shot 3.5). Đã lưu shot 3.3, 3.4.` Lỗi khung có `failed_shots: []`, không có `content`.
- **Khung lỗi**: không đoạn shot nào chạy được; `missing` liệt kê các đoạn còn thiếu.

## `POST /v2/code/plan` (JSON)
- **Body**: body chung; chỉ `engine`, `storyboard`, `chunk_shots` có tác dụng (`system` có thể là `""`).
- **200**: `{"segments": [{key, kind, shots, fingerprint, source?}]}`. `source: "storyboard"` đánh dấu khung là `layout` của storyboard (không cần gọi model).
- **422**: storyboard không đọc được.
- Không gọi model. Đây là nơi duy nhất giữ cách chia đoạn; `authoring-service` không có bản chép.

## `POST /v2/code/segment-prompt` (JSON)
- **Body**: body chung, thêm `key`.
- **200**: `{"system", "user"}`, đúng lượt hỏi mà pipeline sẽ gửi cho đoạn đó (`system` đã ghép hình thư viện).
- **404**: khoá không có trong cách chia hiện tại.
- **409**: đoạn shot mà khung chưa có (hãy chạy hoặc dán khung trước), hoặc khung lấy từ storyboard nên không có gì để viết.
- **422**: storyboard không đọc được.

## `POST /v2/code/segment-parse` (JSON)
- **Body**: body chung, thêm `key` và `reply` (câu trả lời của AI ngoài, hoặc bản sửa tay).
- **200**: `{"fingerprint", "content"}`.
- **422**: `reply` sai khuôn (thiếu hàm shot, không phải `const LAYOUT = …` / `def setup_cast(self):`); câu lỗi nằm trong `error.message`.
- **404 / 409**: như `segment-prompt`.

## Diagnostic của `rendering`
`POST /v1/check/{remotion,manim}` trả thêm `rule` cho mỗi diagnostic (chỉ thêm trường, bên gọi cũ bỏ qua được):
- lỗi bố cục: tên luật, ví dụ `safe_area`, `subtitle_zone`, `text_overflow`;
- lỗi tsc: mã lỗi, ví dụ `TS2322`;
- lỗi lint và Manim: `""`.

## `usage`: `reasoning_chars`, `usage_reported` (CR-056)
Thêm trường, bên gọi cũ bỏ qua được. Áp dụng cho mọi `usage` mà `llm-service` trả về, không chỉ bước Code.
- `reasoning_chars`: số ký tự `reasoning_content` đếm được trên stream. Có cả khi provider không gửi usage.
- `usage_reported`: `false` khi provider không gửi usage. Ví dụ: stream bị ngắt vì vượt `CODE_MAX_REASONING_CHARS`, hoặc lượt gọi lỗi trước khi có usage. Khi đó các số token là `0` vì không biết, không phải vì miễn phí. Thiếu trường (bản cũ) thì hiểu là `true`.
- `authoring-service` ghi hai trường này vào cột `llm_usage.reasoning_chars` và `llm_usage.usage_reported`.

## `POST /v1/code/generate` — đã bỏ (CR-056)
- Route này trả 404.
- `authoring-service` chỉ gọi `/v2/code/generate`. Nếu route đó không có (`llm-service` cũ hơn CR-050), bước Code lỗi với `ErrSegmentsUnsupported` và không chạy theo cách khác.
- Vì vậy phải deploy `llm-service` trước hoặc cùng lúc với `authoring-service`.
