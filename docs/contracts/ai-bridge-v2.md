# AI Bridge v2 · web-gui ↔ extension trình duyệt

Extension `tools/ai-bridge-extension/` chạy prompt trên trang AI mà Creator đã đăng nhập (DeepSeek, Gemini, Grok), rồi đưa câu trả lời về ConceptFlow. **Chỉ còn hàng đợi**: web-gui không chạy trực tiếp một lượt nào; mọi lần chạy là một mục của hàng đợi. Extension không nói chuyện với service nào ngoài API của ConceptFlow: nó tự gọi API để lấy prompt và để đưa câu trả lời về.

| Nơi dùng | Mục hàng đợi | Prompt | Câu trả lời |
|---|---|---|---|
| Nút "Thêm vào hàng đợi AI" ở bước 1–7 (hộp thoại "Dùng AI ngoài", form làm lại, thanh chọn nơi sinh) | `kind: "request"` | route của bước với `X-External-AI: prompt` | cùng route với `X-External-AI: paste` ([`external-ai-v1.md`](external-ai-v1.md)) |
| "Thêm tất cả vào hàng đợi AI" (beat, ghi chú bàn giao, kế hoạch hình, sequence, concept) | `kind: "request"`, nhiều mục | như trên | như trên |
| Bước 3 · Nghiên cứu, ô ② | `kind: "prompt"` | prompt có sẵn trong mục (trang đã dựng từ `GET /research/prompt`) | `POST /research/paste` với body `{raw}`, không có header ([`research-v1.md`](research-v1.md)) |

Mã tương ứng: `services/web-gui/src/hooks/useAiBridge.ts` (phát hiện extension), `useAiQueue.ts` (hàng đợi, theo dõi từng mục), `useQueuedExternalRun.ts` (một lượt của một bước); `tools/ai-bridge-extension/src/protocol.js` (phía extension).

## Kênh

`window.postMessage` trên chính tab ConceptFlow, cùng origin. Content script `bridge.js` của extension nghe và trả lời trên cùng `window`. Hai bên bỏ qua message có `event.source` khác `window` hoặc `event.origin` khác origin của trang.

Mọi message có:

| Trường | Giá trị |
|---|---|
| `source` | `"conceptflow-ai-bridge"` |
| `v` | `2` |
| `from` | `"page"` hoặc `"extension"` |
| `type` | xem bảng dưới |

Extension bản v1 (≤ 1.4.x) chỉ trả lời `ping` có `v: 1`. Trang gửi cả hai ping khi mở màn hình; thấy `pong` có `v: 1` thì hiện dòng "Cập nhật extension ConceptFlow AI Bridge lên bản mới" và không bật nút chạy.

## Nhà cung cấp

| `provider` | Tên | Chat mới mở ở | Chế độ |
|---|---|---|---|
| `deepseek` | DeepSeek | `https://chat.deepseek.com/` | DeepThink và Search theo `modes` của mục |
| `gemini` | Gemini | `https://gemini.google.com/app` | extension không đổi chế độ hay model |
| `grok` | Grok | `https://grok.com/` | extension không đổi chế độ hay model |

Nhà cung cấp mặc định do Creator chọn ở Cài đặt của extension (`chrome.storage.local`, khoá `defaultProvider`, mặc định `deepseek`). Một mục có `provider: null` chạy trên mặc định **tại lúc tới lượt**.

## Message

| Hướng | `type` | Trường thêm | Ý nghĩa |
|---|---|---|---|
| trang → ext | `ping` | | Hỏi có extension không. Trang gửi khi mở màn hình. |
| ext → trang | `pong` | `providers: string[]`, `default_provider`, `version` | Extension có mặt. Cũng tự gửi một lần khi được nạp vào trang. Ngay sau đó là một `queue_state`. |
| trang → ext | `queue_add` | `request_id`, `item` (xem dưới) | Thêm một mục. |
| ext → trang | `queue_added` | `request_id`, `item_id`, `added` | Đã nhận. `added: false` khi một mục cùng loại, cùng request (và cùng prompt với `kind: "prompt"`) đang chờ, chạy hoặc lỗi; `item_id` là mục đó. |
| ext → trang | `error` | `request_id`, `code: "queue_bad_item"` hoặc `"bad_request"`, `message` | Mục không hợp lệ, không thêm. |
| trang → ext | `cancel` | `item_id` | Đưa một mục ra khỏi hàng đợi; mục đang chạy bị dừng và tab của nó đóng. |
| ext → trang | `queue_state` | `waiting`, `running`, `done`, `failed`, `paused` | Số mục theo trạng thái; gửi tới mọi tab ConceptFlow mỗi khi hàng đợi đổi và ngay sau `pong`. |
| ext → trang | `queue_item` | `item_id`, `path`, `project`, `status`, `stage`, `provider`, `error`, `reply` | Một mục vừa đổi trạng thái; gửi tới mọi tab ConceptFlow. |

`item` của `queue_add`:

| Trường | |
|---|---|
| `label` | Tên hiện ở popup, bắt buộc. |
| `project` | Id dự án, hiện ở popup. |
| `request` | `{gateway, method, path, body?}`; `method` phải là `POST`, `path` bắt đầu bằng `/v1/`, `gateway` là địa chỉ http(s). Trang dựng nó bằng chế độ `describe` của lời gọi AI ngoài (`ExternalRequestDescribed`). |
| `modes` | `{deepThink, search}`; thiếu thì DeepThink bật, Search tắt. Chỉ DeepSeek dùng. |
| `provider` | `null` / thiếu (chạy trên mặc định) hoặc `"deepseek"` / `"gemini"` / `"grok"`. Sai id thì mục bị từ chối. |
| `kind` | `"request"` (mặc định) hoặc `"prompt"`. |
| `prompt` | Bắt buộc với `kind: "prompt"`: prompt cố định gửi cho AI. |

`queue_item`:

- `status`: `waiting` → `running` → `done` hoặc `failed`; thêm `cancelled` khi mục rời hàng đợi mà chưa xong (trang `cancel`, hoặc Creator bỏ ở popup). Sau `cancelled` không còn message nào cho mục đó.
- `stage` (khi `running`): `preparing` (lấy prompt), `opening` (mở tab), `sending`, `answering`, `reading`, `saving` (gọi API đưa câu trả lời về).
- `provider`: nhà cung cấp mục đang chạy, đã chạy, hoặc sẽ chạy (mặc định hiện tại) khi còn chờ.
- `error`, `reply` (khi `failed`): lý do bằng tiếng Việt có tên nhà cung cấp ("Chưa đăng nhập Gemini…"), và câu trả lời của AI nếu API không đọc được nó.
- `status: "done"` nghĩa là API đã nhận và lưu kết quả; màn hình của dự án đó nạp lại dữ liệu. Extension đọc câu trả lời, nên trang không nhận được nội dung câu trả lời của mục `done`.

## Mã lỗi

`not_logged_in` (không thấy ô chat sau 30 giây: chưa đăng nhập hoặc giao diện đổi; nêu tên nhà cung cấp), `ui_changed` (kèm tên phần không tìm thấy), `timeout` (15 phút), `tab_closed`, `bad_request`, `queue_bad_item`. Mã `busy` của v1 không còn: hàng đợi tự xếp lượt.

## Cách một mục chạy

Với mỗi mục, đến lượt (không tạm dừng, không có mục lỗi phía trước), extension:

1. Chọn nhà cung cấp: `provider` của mục, hoặc mặc định lúc đó. Ghi lại vào `ranOn`.
2. `kind: "request"`: gọi `path` với `X-External-AI: prompt` để lấy `external_prompt` dựng từ dự án đúng lúc đó. `kind: "prompt"`: dùng `prompt` của mục.
3. Mở tab chat mới của nhà cung cấp, chờ ô chat, gõ prompt, gửi, chờ câu trả lời **mới** xong (số lượt trả lời của AI tăng so với lúc gửi), bấm Copy dưới câu trả lời cuối để lấy nguyên markdown.
4. `kind: "request"`: gọi `path` với `X-External-AI: paste` và `external_reply` cộng vào body. `kind: "prompt"`: `POST path` với body `{raw: <câu trả lời>}`, không header. Thành công thì mục `done`.

Mục lỗi dừng hàng đợi cho tới khi Creator thử lại (có thể đổi nhà cung cấp trước) hoặc bỏ nó ở popup. Mục đang chạy dở khi extension khởi động lại được chạy lại từ đầu. Đổi nhà cung cấp của một mục chỉ được khi mục `waiting` hoặc `failed`.

## Luật phía trang

- Một `queue_add` không nhận được `queue_added` / `error` trong 5 giây thì coi là extension không phản hồi.
- Theo dõi một mục bằng `item_id` từ `queue_added`; bỏ qua `queue_item` của mục khác.
- `done`: nạp lại dữ liệu. `failed` kèm `reply`: đưa `reply` vào ô dán của bước để Creator sửa, kèm `error`. `failed` không có `reply`: hiện `error`. `cancelled`: không hiện gì.
- Đóng hộp thoại hay rời màn hình không huỷ mục; chỉ nút "Huỷ" gửi `cancel`.

## Trong extension (không thuộc hợp đồng với trang)

`bridge.js` ↔ service worker ↔ adapter dùng `chrome.runtime` message `{kind: ready | progress | result | error | queue_add | page_cancel | queue_*}`. Lượt đang chạy được lưu trong `chrome.storage.session`, vì service worker có thể bị Chrome tắt trong lúc AI đang trả lời. Popup nói với worker bằng `queue_pause`, `queue_resume`, `queue_retry`, `queue_drop`, `queue_clear`, `queue_cancel`, `queue_set_provider`. Chi tiết ở README của extension.
