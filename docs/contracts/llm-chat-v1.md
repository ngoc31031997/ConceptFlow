# Hợp đồng `llm-chat.v1` · cuộc gọi AI và nhật ký ở llm-service

Quyết định: [ADR-0045](../../aidlc-docs/decisions/ADR-0045-llm-service-owns-call-log.md). Yêu cầu: `aidlc-docs/inception/requirements/cr-100-llm-call-log.md`. Route thật ở `services/llm-service/app/main.py`.

llm-service là nơi duy nhất nói chuyện với model và là nơi duy nhất giữ nhật ký các cuộc gọi. Các service khác không lưu gì về LLM.

## `POST /v1/chat` (authoring-service → llm-service)

Trả luồng NDJSON: các sự kiện `progress`, rồi đúng một `result` hoặc `error`.

Thân yêu cầu (trường mới so với `cr-039`: `expect_json`, `project_id`, `role`, `step`, `phase`):

| Trường | Kiểu | Ý nghĩa |
|---|---|---|
| `provider` | string, mặc định `hive` | |
| `system`, `user` | string | lượt hỏi gửi cho model; **không được lưu** |
| `model`, `max_tokens`, `temperature`, `json_mode` | | như cũ |
| `expect_json` | bool, mặc định false | đọc khối JSON **đầu tiên** của reply (có hoặc không có hàng rào ```json, bỏ phần thừa phía sau); không đọc được thì một lượt gọi model để sửa; vẫn hỏng thì lỗi `malformed` |
| `project_id`, `role`, `step`, `phase` | string, mặc định rỗng | nguồn của cuộc gọi, ghi vào nhật ký; không tới model. `role` là vai AI của bước (`story_frame`, `beat_sheet`…) |

Sự kiện `result`:

| Trường | Ý nghĩa |
|---|---|
| `content` | reply thô (khi `expect_json` có sửa: reply của lượt sửa) |
| `json` | object JSON đã đọc; **chỉ có khi** `expect_json` |
| `usage` | token; khi có lượt sửa là tổng hai lượt |
| `provider` | |
| `call_id` | dòng `llm_calls` của cuộc gọi (khi có lượt sửa: của lượt sửa) |

Sự kiện `error`: `error` gồm `kind`, `provider`, `message`, `usage`, `partial`, `diag` và **`call_id`**. Khi `expect_json` không đọc được JSON, `kind = "malformed"`, `message` nêu lỗi đọc, `partial` là reply cuối; reply đó đã nằm trong `llm_calls` theo `call_id`.

Phía authoring-service, các hàm `Parse…Reply` nhận `json` (chuỗi object sạch), không tự cắt `{`…`}`. Kiểm tra nghiệp vụ (dữ kiện, ô Creator đã điền, `Validate`) vẫn ở authoring-service.

## `POST /v1/chat/external` (authoring-service → llm-service)

Nhận một câu trả lời Creator dán từ AI bên ngoài hệ thống làm câu trả lời của một lượt hỏi, và ghi nó vào `llm_calls` như một cuộc gọi. Không gọi model. Trả JSON một lần (không phải luồng NDJSON). Quyết định: [ADR-0047](../../aidlc-docs/decisions/ADR-0047-external-ai-through-the-chat-gate.md).

Thân yêu cầu: `system`, `user` (prompt của lượt hỏi, lưu vào nhật ký), `reply` (đoạn dán), `expect_json`, `project_id`, `role`, `step`, `phase` (như `/v1/chat`).

- `200`: `content` (= `reply`), `usage` (`model = "external"`, `usage_reported = false`, token 0), `provider = "external"`, `call_id`; có `json` khi `expect_json`.
- `422`: chỉ khi `expect_json` mà `reply` không có object JSON đọc được. Thân `{"error": {"kind": "malformed", "provider": "external", "message", "call_id"}}`. Không có lượt sửa: Creator dán lại. Dòng nhật ký vẫn được ghi (`ok = false`).
- Dòng `llm_calls`: `provider = "external"`, `model = "external"`, `response = reply`, `usage_reported = false`.

## Prompt quá dài với Ollama

Với Ollama, llm-service gửi `options.num_ctx = OLLAMA_NUM_CTX` kèm mọi cuộc gọi. Trước khi gọi, nó ước lượng số token của prompt (2 ký tự một token); nếu vượt 85% cửa sổ thì **cắt phần giữa** (giữ đầu và đuôi, chèn dòng `[... phần giữa bị cắt ...]`) rồi gọi bình thường, không báo lỗi, và ghi cảnh báo ra log của service. `system_prompt`/`user_prompt` trong `llm_calls` là bản đã cắt, đúng cái model nhận. Hive không bị cắt.

## Nhật ký `llm_calls` (database `llm` của `llm-db`, `127.0.0.1:5435`)

Một dòng cho mỗi `Provider.chat` (kể cả thử lại, gộp thành một dòng), ghi mọi đường gọi model: `/v1/chat`, `/v1/suggest-metadata`, `/v1/storyboard/finalize`, `/v2/code/generate`.

| Cột | Ý nghĩa |
|---|---|
| `call_id` (uuid, khoá), `created_at` | |
| `provider`, `model` | |
| `role`, `step`, `phase` | nguồn; `''` khi người gọi không gửi. Pipeline code ghi `step = code`, `phase` = `layout`/`chunk`/`repair`; lượt sửa JSON ghi `phase = json_repair` |
| `project_id` | không khoá ngoại |
| `response` | nội dung thô (hoặc phần đã tới khi lỗi); nén lz4 |
| `system_prompt`, `user_prompt` | đúng prompt đã gửi cho model; nén lz4; có cả ở lần gọi lỗi và lần bị từ chối vì quá dài ([ADR-0046](../../aidlc-docs/decisions/ADR-0046-call-log-keeps-the-prompt.md)) |
| `prompt_chars` | số ký tự system + user |
| `num_ctx` | cửa sổ ngữ cảnh gửi cho Ollama (`OLLAMA_NUM_CTX`, mặc định 16384); `0` với nhà cung cấp tự đặt |
| `ok`, `error_kind`, `error_message` | |
| `prompt_tokens`, `completion_tokens`, `reasoning_tokens`, `cached_tokens`, `reasoning_chars`, `usage_reported` | `usage_reported = false`: token không phải số đã bị tính |
| `duration_ms` | |

Ghi nhật ký lỗi (DB sập) không làm hỏng cuộc gọi: llm-service ghi lỗi ra log của nó và trả kết quả bình thường.

## Đọc và xoá (web-gui → api-gateway → llm-service)

| Route | Ý nghĩa |
|---|---|
| `GET /v1/llm/calls?limit&before&project_id&role&ok` | các dòng thô, mới nhất trước, **không kèm prompt**; `{calls, next_before}`; `before` = `next_before` của trang trước; `limit` ≤ 500 |
| `GET /v1/llm/calls/{call_id}` | một dòng đầy đủ, kèm `system_prompt` và `user_prompt`; 404 khi không có |
| `POST /v1/llm/calls/delete` | thân đúng một trong `{"ids":[…]}`, `{"older_than_days":N}`, `{"all":true}` (422 nếu không đúng một); trả `{"deleted":n}` |
| `GET /v1/llm/stats?step&phase&days=30` | theo model: `{calls, ok, failures:{kind:n}, avg_completion_tokens, avg_duration_ms}`; trung bình chỉ tính các cuộc gọi thành công |

Xoá dòng làm mất số liệu thống kê tính trên chính các dòng đó. Không có tác vụ nền tự xoá; xoá theo ngày là nút bấm tay ở màn Nhật ký.

## Cấu hình

`LLM_DATABASE_URL` (bắt buộc; llm-service không khởi động thiếu nó). api-gateway: `LLM_SERVICE_URL`.
