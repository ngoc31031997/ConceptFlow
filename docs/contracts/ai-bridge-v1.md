# AI Bridge v1 · web-gui ↔ extension trình duyệt

Extension `tools/ai-bridge-extension/` chạy một prompt trên trang AI mà Creator đã đăng nhập (hiện chỉ có DeepSeek), rồi đưa câu trả lời về web-gui. Extension không nói chuyện với service nào; web-gui lấy prompt từ server và gửi câu trả lời về server:

| Bước | Prompt | Câu trả lời | Chế độ DeepSeek |
|---|---|---|---|
| 3 · Nghiên cứu | `GET /research/prompt` | `POST /research/paste` rồi tự đọc như khi dán tay ([`research-v1.md`](research-v1.md)) | DeepThink + Search |
| Mọi hộp thoại "Dùng AI ngoài" ở bước 1–7 | route của bước với `X-External-AI: prompt` | cùng route với `X-External-AI: paste` ([`external-ai-v1.md`](external-ai-v1.md)) | DeepThink (hộp thoại đổi được qua `deepSeekModes`) |

Mã tương ứng: `services/web-gui/src/hooks/useAiBridge.ts` (phía trang; dùng ở `ExternalAiDialog.tsx` và `ResearchStepPage.tsx`), `tools/ai-bridge-extension/src/protocol.js` (phía extension).

## Kênh

`window.postMessage` trên chính tab ConceptFlow, cùng origin. Content script `bridge.js` của extension nghe và trả lời trên cùng `window`. Hai bên bỏ qua message có `event.source` khác `window` hoặc `event.origin` khác origin của trang.

Mọi message có:

| Trường | Giá trị |
|---|---|
| `source` | `"conceptflow-ai-bridge"` |
| `v` | `1` |
| `from` | `"page"` hoặc `"extension"` |
| `type` | xem bảng dưới |

## Message

| Hướng | `type` | Trường thêm | Ý nghĩa |
|---|---|---|---|
| trang → ext | `ping` | | Hỏi có extension không. Trang gửi khi mở màn hình. |
| ext → trang | `pong` | `providers: string[]`, `version` | Extension có mặt. Cũng tự gửi một lần khi được nạp vào trang. |
| trang → ext | `run` | `request_id`, `provider` (`"deepseek"`), `prompt`, `modes` tuỳ chọn `{deep_think, search}` (bool, thiếu thì coi là `true`) | Chạy prompt với các chế độ đã chọn. Mỗi lúc chỉ một lượt. |
| trang → ext | `cancel` | `request_id` | Bỏ lượt đang chạy; tab của trang AI vẫn giữ. |
| ext → trang | `progress` | `request_id`, `stage`: `opening` / `sending` / `answering` / `reading` | Tiến độ. Message đầu tiên (`opening`) là xác nhận đã nhận lượt chạy. |
| ext → trang | `result` | `request_id`, `text` | Toàn bộ câu trả lời, giữ khối ```` ```json ````. |
| ext → trang | `error` | `request_id`, `code`, `message` | Lượt chạy kết thúc không có câu trả lời. `message` là câu tiếng Việt để hiện cho Creator. |

Mã lỗi: `not_logged_in`, `ui_changed` (kèm tên phần không tìm thấy), `timeout` (15 phút), `tab_closed`, `busy`, `bad_request`.

## Luật phía trang

- Bỏ qua `progress` / `result` / `error` có `request_id` khác lượt đang chờ.
- Không nhận được message nào cho lượt chạy trong 5 giây sau `run` thì coi là extension không phản hồi.
- `result` được điền vào ô ③ rồi đọc ngay như khi Creator bấm "Đọc kết quả". Server đọc không được thì text vẫn nằm trong ô để Creator sửa.
- `error` không điền gì.

## Trong extension (không thuộc hợp đồng với trang)

`bridge.js` ↔ service worker ↔ adapter dùng `chrome.runtime` message `{kind: run | cancel | ready | progress | result | error}`. Lượt chạy được lưu trong `chrome.storage.session`, vì service worker có thể bị Chrome tắt trong lúc AI đang trả lời. Chi tiết ở README của extension.
