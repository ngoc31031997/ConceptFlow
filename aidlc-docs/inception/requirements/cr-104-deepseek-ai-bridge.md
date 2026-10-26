# CR-104 · Extension AI Bridge: chạy prompt trên DeepSeek web — yêu cầu và thiết kế

Nhánh `feature/cr-104-deepseek-ai-bridge`. Trạng thái: Creator duyệt 2026-10-26. Ban đầu đánh số CR-102; đổi sang CR-104 vì CR-102 "AI ngoài" ([`cr-102-external-ai-paste.md`](cr-102-external-ai-paste.md)) đã merge trước với cùng số.

## Bối cảnh (2026-10-26)

Bước 3 · Nghiên cứu còn ba thao tác tay: chép prompt ở ô ②, dán vào một công cụ AI có tra web, rồi chép câu trả lời và dán vào ô ③. Các bước khác của web-gui đã gọi AI qua server nên không có luồng tay này. Creator muốn tự động hoá ba thao tác đó bằng một extension trình duyệt, dùng tài khoản web DeepSeek của Creator (không tốn tiền API). DeepSeek làm trước; các trang AI khác sẽ làm sau.

## Chốt của Creator (2026-10-26)

1. Dùng cách 1: một extension Chrome tự điều khiển tab AI.
2. Trang AI đầu tiên là DeepSeek (`chat.deepseek.com`).
3. Khi DeepSeek trả lời xong: điền vào ô ③ **và tự bấm "Đọc kết quả"**.
4. Tab DeepSeek mở ở nền, không chuyển tab; tab được giữ lại để Creator xem hoặc hỏi tiếp.
5. Bật **Search + DeepThink (R1)**.

## Phạm vi

### A. Extension `tools/ai-bridge-extension/` (Chrome, Manifest V3)

Extension là một source riêng, không phải service Docker và không phụ thuộc code của các service. Creator cài bằng "Load unpacked" trong `chrome://extensions`. Thư mục có `README.md` riêng hướng dẫn cài đặt, sử dụng, xử lý lỗi và cách sửa selector khi DeepSeek đổi giao diện (Creator yêu cầu 2026-10-26).

- `bridge.js` là content script chạy trên trang ConceptFlow (`http://localhost:3000/*`). Nó nhận yêu cầu từ trang qua `window.postMessage`, chuyển cho service worker, rồi gửi trạng thái và kết quả về trang. Script chỉ nhận message có `source` đúng và cùng origin.
- `background.js` là service worker. Nó mở một tab DeepSeek mới ở nền (`active: false`), đợi tab tải xong, gửi lệnh chạy cho adapter rồi chuyển tiếp tiến độ về đúng tab ConceptFlow đã gửi yêu cầu. Mỗi yêu cầu có một `request_id`.
- `adapters/deepseek.js` là content script chạy trên `chat.deepseek.com`. Các bước:
  1. Kiểm đã đăng nhập: có ô nhập chat thì coi là đã đăng nhập, không có thì báo lỗi `not_logged_in`.
  2. Bật nút DeepThink và Search nếu chúng đang tắt.
  3. Ghi prompt vào ô chat bằng native value setter cộng sự kiện `input`, để React của DeepSeek nhận được giá trị, rồi bấm gửi.
  4. Chờ trả lời xong. Hai điều kiện: nút dừng đã biến mất, và nội dung câu trả lời cuối không đổi trong 3 giây.
  5. Lấy câu trả lời: ưu tiên markdown gốc qua nút Copy của DeepSeek (đọc clipboard sau khi bấm); nếu không được thì lấy `innerText` của khối trả lời cuối, bỏ phần "suy nghĩ" của DeepThink.
- Mọi selector DOM của DeepSeek gom ở đầu `deepseek.js`. Khi DeepSeek đổi giao diện thì chỉ sửa chỗ này.
- Thời gian chờ tối đa là 15 phút, vì Search + R1 có thể chạy lâu. Hết giờ thì báo lỗi `timeout`, không trả kết quả dở.
- Lỗi được báo về bằng mã kèm câu tiếng Việt: `not_logged_in`, `ui_changed` (không thấy ô chat / nút gửi / câu trả lời), `timeout`, `tab_closed`, `busy` (đang có một yêu cầu khác chạy). Không đoán, không điền gì khi có lỗi.

### B. Giao thức trang ↔ extension

Mọi message có `source: "conceptflow-ai-bridge"` và `v: 1`.

| Hướng | `type` | Dữ liệu |
|---|---|---|
| trang → ext | `ping` | |
| ext → trang | `pong` | `providers: ["deepseek"]`, `version` |
| trang → ext | `run` | `request_id`, `provider`, `prompt` |
| ext → trang | `progress` | `request_id`, `stage`: `opening` / `sending` / `answering` / `reading` |
| ext → trang | `result` | `request_id`, `text` |
| ext → trang | `error` | `request_id`, `code`, `message` |

Trang bỏ qua mọi message có `request_id` không khớp với yêu cầu đang chờ. Hợp đồng này được ghi thành `docs/contracts/ai-bridge-v1.md` và thêm vào chỉ mục `docs/contracts/README.md`.

### C. web-gui

- Hook mới `useAiBridge` gửi `ping` khi mở màn hình. Có `pong` thì coi như có extension. Hook cung cấp hàm `run(prompt)` trả về kết quả hoặc lỗi, kèm `stage` hiện tại.
- Ô ② (`PromptCard`) thêm nút **"Chạy bằng DeepSeek"**. Nút chỉ hiện khi đã phát hiện extension. Không có extension thì giữ nguyên giao diện hiện nay và có một dòng gợi ý cách cài.
- Trong lúc chạy: nút chuyển thành trạng thái đang chạy, có dòng tiến độ ("Đang mở DeepSeek…", "DeepSeek đang suy nghĩ và tra web…", "Đang lấy câu trả lời…"), và có nút Huỷ (bỏ chờ ở phía trang; tab DeepSeek vẫn được giữ).
- Khi có kết quả: điền vào ô ③ rồi tự gọi đúng đường "Đọc kết quả" hiện nay (`research.pasteResults`), nên log lần dán, thông báo và luật gộp giữ nguyên. Ô ③ được nâng thành controlled từ trang để nhận text từ ô ②. Nếu đọc lỗi thì text vẫn nằm lại trong ô ③ để Creator sửa.
- Có lỗi: hiện câu lỗi cạnh nút, không điền gì.
- Nút bị khoá khi bước đã khoá (`locked`), đang bận, hoặc prompt chưa dựng xong.
- Theo `docs/ux-ui-design-rules.md` và `docs/code-standards-rules.md`.

### D. Backend

Không đổi gì. `POST /research/paste` nhận text như khi dán tay.

## Mở rộng: mọi bước có "Dùng AI ngoài" (Creator yêu cầu 2026-10-26)

Creator muốn bước 1 cũng chạy được bằng DeepSeek: viết, làm lại kèm ghi chú và đánh giá, chỉ bật DeepThink. Lúc đó CR-102 "AI ngoài" đã merge vào `main`. CR đó đã cho mọi lượt AI ở bước 1–7 một hộp thoại "Dùng AI ngoài" (`ExternalAiDialog`, hợp đồng [`external-ai-v1.md`](../../../docs/contracts/external-ai-v1.md)): server dựng prompt, Creator chép sang AI ngoài, dán câu trả lời, server đọc như câu trả lời của model.

Creator chốt làm một lần rồi merge, không tách CR. Cách làm:

- Không thêm route nào ở server. Bước 1 và mọi bước khác dùng luồng `X-External-AI` sẵn có.
- `ExternalAiDialog` có thêm nút **Chạy bằng DeepSeek**, đặt giữa ô prompt và ô dán, chỉ hiện khi có extension. Bấm vào thì extension chạy prompt của hộp thoại trên DeepSeek, câu trả lời được điền vào ô dán và hộp thoại tự bấm "Đọc kết quả". Đọc không được thì lý do hiện như khi dán tay và câu trả lời vẫn nằm trong ô.
- Mặc định chỉ bật DeepThink. Hộp thoại nhận `deepSeekModes` cho bước nào cần tra web.
- Bước 3 · Nghiên cứu giữ nút ở ô ② (DeepThink + Search) như phần A–C.
- Đóng hộp thoại thì trang ngừng chờ lượt chạy (gửi `cancel`).
- Extension: message `run` thêm `modes {deep_think, search}`; thiếu thì bật cả hai. Adapter đặt hai nút về đúng trạng thái được yêu cầu, kể cả tắt Search nếu đang bật.

## Ngoài phạm vi

- ChatGPT, Claude.ai, Gemini, Perplexity: chỉ chuẩn bị chỗ cắm adapter mới (`providers`), chưa viết.
- Firefox / Safari.
- Chạy hàng loạt hoặc tự chạy lại khi lỗi. Mỗi lần chạy do Creator bấm một lần, để giảm rủi ro vi phạm điều khoản dùng web của DeepSeek.
- Đăng extension lên Chrome Web Store.
- Các nút copy prompt khác (ví dụ prompt ảnh thumbnail): không phải luồng dán kết quả dạng text.

## Rủi ro

- DeepSeek đổi giao diện thì adapter hỏng. Khi đó extension báo `ui_changed`, Creator dán tay như cũ, và chỉ cần sửa bảng selector.
- Tự động hoá giao diện web có thể trái điều khoản của DeepSeek. Creator đã biết và chấp nhận ở mức mỗi lần một cú bấm.
- Đọc clipboard từ tab nền có thể bị Chrome chặn. Khi đó adapter dùng đường `innerText` (có thể mất định dạng markdown nhưng giữ được khối JSON).

## Kiểm thử

- Unit test web-gui (vitest) với một extension giả lập bằng `postMessage`: phát hiện có / không có extension, các trạng thái tiến độ, kết quả được điền và tự đọc, lỗi không điền gì, bỏ qua `request_id` lạ, Huỷ.
- Unit test logic thuần của extension: chọn message hợp lệ, điều kiện "trả lời xong", lọc phần suy nghĩ. Phần này chạy được ngoài trình duyệt.
- Adapter DeepSeek chạy trên trang thật **không tự động test được**. Creator kiểm bằng tay sau khi cài; báo cáo sẽ ghi rõ đây là kiểm tra chưa chạy tự động. Không gọi DeepSeek thật trong test.
- Unit test `ExternalAiDialog` với extension giả: không có extension thì không có nút; chạy, điền và đọc; câu trả lời không dùng được thì giữ trong ô; lỗi extension không điền gì.
- Rebuild `web-gui` sau khi code và xác nhận healthy.
