# authoring-service → llm-service: bước Code theo đoạn (`/v2/code/*`)

CR-050 Unit 2, [ADR-0030](../../aidlc-docs/decisions/ADR-0030-code-segments-and-v2-code-contract.md). Đi kèm `/v1/code/generate`, vẫn được giữ để hai service deploy độc lập (xem mục cuối). Bỏ `/v1` là một mục backlog.

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
  "illustrations": [{"name": "…", "usage": "…", "description": "…", "code": "…"}],
  "subtitle_band": {"edge": "top | bottom", "px": 240},
  "video_font": "",
  "chunk_shots": 3,
  "segments": [{"key": "frame", "fingerprint": "…", "content": {"code": "…"}}],
  "only": ["1.4-1.6"]
}
```
- `chunk_shots`: từ 0 đến 10; 0 nghĩa là dùng `CODE_CHUNK_SHOTS`.
- `segments`: các đoạn `done` đã lưu. Nội dung khung là `{"code": "…"}`, nội dung đoạn shot là `{"shots": {"1.1": "…"}}`.
- `only`: nếu có thì chỉ chạy các đoạn này; bỏ trống thì chạy mọi đoạn còn thiếu.
- `subtitle_band` và `video_font` chỉ dùng cho Remotion.

## `POST /v2/code/generate` (NDJSON)
Sự kiện theo thứ tự thời gian. Mỗi dòng là một JSON có trường `type`.

| `type` | Trường | Bên nhận làm gì |
|---|---|---|
| `plan` | `segments: [{key, kind: "frame"\|"shots", shots, fingerprint}]` | Xoá đoạn có khoá không còn trong plan; thêm đoạn mới ở trạng thái `pending`; đoạn lệch vân tay về `pending` và xoá nội dung. |
| `segment_start` | `key` | Đoạn thành `running`. |
| `segment_done` | `key, fingerprint, content, source ("ai"\|"storyboard"), repaired, duration_ms` | Lưu `done`. Với `repaired: true`, giữ nguyên `source` và `duration_ms` cũ. |
| `segment_failed` | `key, error: {kind, message, …}` | Lưu `failed`, giữ nội dung cũ nếu có. |
| `call` | `phase, label, segment, ok, usage, error_kind, error_message, duration_ms` | Ghi một dòng `llm_usage` ngay. |
| `check` | `phase ("chunk"\|"final"), round, segment, diagnostics: [{message, line, kind, rule, shot, segment}]` | Ghi vào `code_check_diagnostics`. |
| `phase`, `chunk_start`, `chunk_done`, `chunk_split`, `chunk_repair` | như CR-039/CR-048 | Chỉ dùng cho thanh tiến độ. |
| `result` | `status ("done"\|"incomplete"), code, check_ok, diagnostics, repair_rounds, warnings, scene_class_name, failed, missing` | `done`: lưu code đã ghép. `incomplete`: bước kết thúc "chưa xong". |
| `error` | `error: {kind, message, …}`, `calls: []` | Lỗi dừng cả lượt: `auth`, `balance`, `not_configured`, storyboard không đọc được, lỗi bất ngờ. Không ghi lại `calls`, vì mọi lượt gọi đã có sự kiện `call` riêng. |

- **Lỗi một đoạn** không dừng các đoạn khác. Lượt chạy kết thúc với `result.status = "incomplete"` và `failed` liệt kê các đoạn lỗi.
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

## `POST /v1/code/generate` (giữ lại, sẽ bỏ)
- Contract trước CR-050, không đổi: sự kiện tiến độ (`phase`, `chunk_*`); `result` có `calls` ở cuối; đoạn lỗi đầu tiên làm cả lượt lỗi, `error.calls` liệt kê các lượt đã tính tiền.
- Chạy trên pipeline mới, không nhận `segments`/`only`. Không còn cache trong bộ nhớ, nên lượt chạy lại trả tiền lại cho các đoạn đã xong.
- **Fallback của `authoring-service`**: khi `/v2/code/*` trả 404 mà body không phải `{"error": …}` của `llm-service` (tức là route không có), bước Code chạy qua `/v1`, và mỗi phần tử `calls` được ghi `llm_usage`. Panel đoạn báo 501 "llm-service chưa hỗ trợ lưu theo đoạn".
- Bỏ route này và đường fallback: mục backlog trong `aidlc-docs/aidlc-state.md`.
