# ConceptFlow AI Bridge

Extension Chrome giúp ConceptFlow chạy prompt trên **DeepSeek**, **Gemini** hoặc **Grok** (bản web) thay cho việc chép/dán tay hoặc gọi API. Mọi lượt chạy đều đi qua **hàng đợi AI**:

- Ở bước 1–7, bấm **Thêm vào hàng đợi AI** (hộp thoại "Dùng AI ngoài", form làm lại, thanh chọn nơi sinh), hoặc **Thêm tất cả vào hàng đợi AI** (beat, ghi chú bàn giao, kế hoạch hình…).
- Ở bước 3 · Nghiên cứu, bấm **Thêm vào hàng đợi AI** ở ô ② để AI tra web và trả bảng dữ kiện.

Extension lấy prompt, mở tab chat mới của nhà cung cấp, chờ câu trả lời rồi gọi API ConceptFlow để lưu kết quả. ConceptFlow chỉ việc tải lại màn hình khi mục xong.

- Dùng tài khoản web của bạn ở từng nhà cung cấp, không cần API key, không tốn tiền API.
- Hỗ trợ: **DeepSeek** (`chat.deepseek.com`, mặc định), **Gemini** (`gemini.google.com/app`), **Grok** (`grok.com`). DeepSeek tự bật hoặc tắt **DeepThink** và **Search** theo từng mục (DeepThink cho hầu hết bước, DeepThink + Search cho bước Nghiên cứu). Gemini và Grok: extension **không đổi** chế độ hay model, mục chạy theo lựa chọn hiện có trong trang của chúng.
- Dùng được cả khi ConceptFlow chưa cấu hình AI (không có API key).
- Không gửi dữ liệu đi đâu khác ngoài các tab nhà cung cấp và API của ConceptFlow. Extension không có server riêng, không đọc trang nào khác ngoài ConceptFlow (`localhost:3000`) và ba trang AI trên.

## Cài đặt

Cần Google Chrome 111 trở lên (hoặc Edge, Brave, Arc, tức là trình duyệt dùng nhân Chromium).

1. Mở `chrome://extensions`.
2. Bật **Developer mode** (góc trên bên phải).
3. Bấm **Load unpacked** và chọn thư mục này: `tools/ai-bridge-extension` trong repo ConceptFlow.
4. Extension **ConceptFlow AI Bridge** hiện trong danh sách. Nên ghim nó lên thanh công cụ: bấm icon để mở popup **Hàng đợi AI** và **Cài đặt**.
5. Mở trang của từng nhà cung cấp bạn định dùng (`chat.deepseek.com`, `gemini.google.com`, `grok.com`) và **đăng nhập** một lần. Extension dùng phiên đăng nhập này.
6. Tải lại tab ConceptFlow nếu nó đang mở, để extension được nạp vào trang.

Cập nhật extension sau khi kéo code mới (bản 2.0 đổi giao thức, nên **bắt buộc**): vào `chrome://extensions`, bấm nút ↻ trên thẻ của extension, rồi tải lại tab ConceptFlow và các tab nhà cung cấp. ConceptFlow gặp extension cũ thì hiện dòng "Cập nhật extension ConceptFlow AI Bridge lên bản mới" và không bật nút chạy.

ConceptFlow chạy ở cổng khác `3000`? Sửa hai dòng `matches` của `bridge.js` trong `manifest.json` rồi tải lại extension.

## Sử dụng

### Thêm một mục vào hàng đợi (bước 1–7)

Mọi lượt AI ở bước 1–7 (viết ý tưởng, đánh giá, brief, câu hỏi nghiên cứu, câu chuyện, beat, kịch bản, rà soát…) có nút hoặc lựa chọn **Dùng AI ngoài**.

1. Bấm **Dùng AI ngoài**: hộp thoại hiện prompt để chép tay (cách cũ vẫn dùng được). Hoặc bấm thẳng **Thêm vào hàng đợi AI** (cạnh nút, trong hộp thoại và trong form làm lại).
2. Màn hình hiện trạng thái của mục: "Đang chờ trong hàng đợi AI · 2 mục trước · sẽ chạy trên Gemini", rồi "Đang chạy trên Gemini · đang trả lời…". Nút **Huỷ** đưa mục ra khỏi hàng đợi (mục đang chạy bị dừng).
3. Khi mục xong, màn hình tự tải lại dữ liệu. Hộp thoại hay form làm lại đóng lại.
4. Nếu bước không đọc được câu trả lời, hộp thoại mở ra với câu trả lời trong ô dán kèm lý do. Bạn sửa rồi bấm **Đọc kết quả**, hoặc thêm lại vào hàng đợi. Lỗi không có câu trả lời (chưa đăng nhập…) hiện ngay dưới nút.

**Đóng hộp thoại không huỷ mục**: mục vẫn nằm trong hàng đợi và vẫn chạy. Màn hình chỉ tự tải lại khi nó còn mở; đã đóng thì tải lại màn hình sau khi mục xong.

### Bước 3 · Nghiên cứu

1. Mở project, vào bước **3 · Nghiên cứu**, tạo câu hỏi ở ô ①.
2. Ở ô ② , bấm **Thêm vào hàng đợi AI**. Mục này mang sẵn prompt nghiên cứu; trên DeepSeek, extension bật DeepThink + Search. Trên Gemini hoặc Grok, hãy tự chọn chế độ có tra web trong trang đó trước khi chạy.
3. Khi xong, bảng dữ kiện tự tải lại và hệ thống bắt đầu kiểm link như khi bạn dán tay.
4. Nếu ConceptFlow không đọc được câu trả lời, câu trả lời được đưa vào ô ③ kèm lý do để bạn sửa rồi bấm **Đọc kết quả**.

### Chọn nhà cung cấp

- **Mặc định**: trang **Cài đặt** của extension (popup → Cài đặt → Mở cài đặt) có ô **Nhà cung cấp mặc định**. Mục không chọn riêng chạy trên nhà cung cấp mặc định **tại lúc tới lượt**.
- **Từng mục**: ở popup, mỗi mục chờ hoặc lỗi có ô chọn "Mặc định (DeepSeek) / DeepSeek / Gemini / Grok". Khoá khi mục đang chạy hoặc đã xong. Mục lỗi vì một nhà cung cấp (hết hạn đăng nhập, đổi giao diện) có thể đổi sang nhà cung cấp khác rồi **Thử lại**.

## Hàng đợi AI

Bấm icon extension để mở popup. Tab **Hàng đợi AI** liệt kê các mục ConceptFlow đã thêm; chúng chạy lần lượt, mỗi lúc một mục, kể cả khi tab ConceptFlow đã đóng hay tải lại (hàng đợi lưu trong `chrome.storage.local`).

Với mỗi mục, extension: (1) gọi API của ConceptFlow với `X-External-AI: prompt` để lấy prompt dựng từ dự án đúng lúc đó (nên beat sau đọc được beat trước vừa viết; mục của bước Nghiên cứu mang sẵn prompt), (2) chạy prompt trên nhà cung cấp của mục, (3) gọi lại API với `X-External-AI: paste` kèm câu trả lời để ConceptFlow đọc và lưu (bước Nghiên cứu: `POST /research/paste` với `{raw}`).

- **Tạm dừng / Chạy tiếp**: dừng hay tiếp tục lấy mục kế tiếp; mục đang chạy vẫn chạy nốt.
- **Mục lỗi** (nhà cung cấp không trả lời, ConceptFlow không đọc được câu trả lời, tab bị đóng) dừng cả hàng đợi. Popup hiện lý do và câu trả lời của AI nếu có. **Thử lại** chạy lại mục từ đầu, **Bỏ** xoá mục rồi hàng đợi chạy tiếp.
- **Huỷ** một mục đang chạy ở popup coi như mục lỗi "Đã huỷ". Huỷ từ màn hình ConceptFlow thì mục bị đưa ra khỏi hàng đợi.
- Khởi động lại Chrome giữa lúc một mục đang chạy thì mục đó chạy lại từ đầu.
- Chrome phải đang mở và nhà cung cấp đã đăng nhập. Mỗi mục mở một tab chat mới rồi đóng nó khi lấy được câu trả lời, nên màn hình nhảy sang tab đó mỗi lần (tab ẩn bị Chrome làm chậm). Prompt được dựng lúc tới lượt, nên sửa dự án trong lúc mục còn chờ thì mục đọc theo bản mới; muốn tránh thì bấm Tạm dừng.
- Extension gọi API ConceptFlow ở `http://localhost:8080`. Gateway ở địa chỉ khác thì sửa `host_permissions` trong `manifest.json`.

## Xử lý lỗi

| Hiện tượng | Nguyên nhân thường gặp | Cách xử lý |
|---|---|---|
| Không có nút "Thêm vào hàng đợi AI" (hộp thoại hiện dòng "Cài extension…") | Extension chưa cài, bị tắt, hoặc tab ConceptFlow mở trước khi cài | Kiểm tra `chrome://extensions`, rồi tải lại tab ConceptFlow |
| "Cập nhật extension ConceptFlow AI Bridge lên bản mới…" | Extension còn là bản 1.x | Bấm ↻ trên thẻ extension ở `chrome://extensions`, tải lại tab ConceptFlow |
| "Extension không phản hồi" | Extension vừa được tải lại nhưng trang thì chưa | Tải lại tab ConceptFlow |
| "Chưa đăng nhập Gemini hoặc giao diện Gemini đã đổi (không thấy ô chat)" (hoặc DeepSeek, Grok) | Phiên hết hạn, hoặc trang hiện hộp thoại chặn (CAPTCHA, đồng ý điều khoản), hoặc giao diện đổi | Mở trang nhà cung cấp, đăng nhập và xử lý hộp thoại, chạy lại. Nếu vẫn lỗi, kiểm selector **Ô chat** ở Cài đặt |
| "Giao diện Grok đã đổi… (nút Copy)" hoặc tên phần khác | Nhà cung cấp đổi giao diện, hoặc Grok không để tiếng Anh (nút Copy tìm theo nhãn "Copy response") | Dán selector mới ở trang Cài đặt (xem dưới) |
| Tiến độ đứng ở "đang trả lời…" rất lâu | Bạn chuyển sang tab khác nên tab nhà cung cấp bị ẩn và Chrome làm chậm nó | Quay lại tab đó; câu trả lời sẽ được lấy khi xong |
| "… chưa trả lời xong sau 15 phút" | Câu hỏi quá dài, mạng chậm, nhà cung cấp quá tải | Xem tab nhà cung cấp; có câu trả lời thì chép và dán tay |

Xem log chi tiết:

- Phía nhà cung cấp: mở DevTools (F12) trong tab của nó, tab Console, tìm dòng `ai-bridge`.
- Phía extension: ở `chrome://extensions`, bấm **service worker** trên thẻ extension.

## Cách hoạt động

```
ConceptFlow (localhost:3000)                    Chrome extension                         tab nhà cung cấp (mới)
  useAiQueue ──postMessage──▶ bridge.js ──runtime──▶ background.js ──tabs.create──▶ adapter (core + providers/<id>)
            ◀──postMessage── bridge.js ◀──tabs.sendMessage── background.js ◀──runtime── adapter
```

1. Trang gửi `queue_add` (có `request_id`) qua `window.postMessage`; mỗi lần mục đổi trạng thái, extension gửi `queue_item` tới mọi tab ConceptFlow. Giao thức đầy đủ ở `docs/contracts/ai-bridge-v2.md`.
2. `background.js` (service worker) giữ hàng đợi trong `chrome.storage.local`. Khi tới lượt một mục nó chọn nhà cung cấp, lưu lượt chạy vào `chrome.storage.session` (Chrome có thể tắt service worker trong lúc AI đang trả lời) và mở tab chat mới của nhà cung cấp.
3. Adapter của nhà cung cấp (`providers/<id>/adapter.js` trên `adapter/core.js`) nạp xong thì xin việc (`ready`). Nó chờ ô chat (tối đa 30 giây, không có thì báo chưa đăng nhập), làm bước chuẩn bị riêng của nhà cung cấp (DeepSeek đặt DeepThink và Search), ghi prompt vào ô chat rồi bấm gửi (không được thì Enter).
4. Câu trả lời được coi là xong chỉ khi **lượt trả lời mới** (số lượt trả lời trên trang tăng so với lúc gửi) có dấu hiệu "đã xong" của nhà cung cấp: DeepSeek, hàng nút dưới câu trả lời hiện và nội dung đứng yên 5 giây (20 giây nếu không thấy hàng nút); Gemini, chân trang `.response-footer.complete` và nội dung hết `aria-busy`; Grok, hàng nút `response-footer-actions`.
5. Lấy câu trả lời bằng nút **Copy** dưới lượt cuối để giữ nguyên markdown và khối ```` ```json ````. `adapter/clipboard-hook.js` chạy trong trang nhà cung cấp và chặn đúng lần copy đó, nên **clipboard thật của bạn không bị ghi đè**. Các lần bạn tự copy vẫn đi qua bình thường. Nếu Copy không đưa được văn bản, DeepSeek đọc chữ trên trang (bỏ phần suy nghĩ, dựng lại khối code); Gemini và Grok báo lỗi `ui_changed` thay vì đoán.
6. Worker đóng tab, gọi API đưa câu trả lời về ConceptFlow, rồi báo `queue_item` `done` (hoặc `failed` kèm lý do và câu trả lời).

| File | Vai trò |
|---|---|
| `manifest.json` | Khai báo extension (Manifest V3), quyền `tabs`, `storage`, trang được nạp script |
| `src/providers.js` | Bảng nhà cung cấp: id, tên, URL chat mới, host; id mặc định |
| `src/protocol.js` | Định dạng message (v2), mã lỗi và câu báo lỗi tiếng Việt (có tên nhà cung cấp), giới hạn thời gian |
| `src/jobs.js` | Luật của lượt chạy đang lưu (còn chặn lượt mới không, message có thuộc lượt này không) |
| `src/queue.js` | Hàng đợi: thêm, gộp trùng, chọn mục kế tiếp, đổi nhà cung cấp, trạng thái |
| `src/queue-http.js` | Các lời gọi API quanh một lượt: `prompt`, `paste`, `raw` |
| `src/bridge.js` | Content script trên ConceptFlow: chuyển message trang ↔ service worker |
| `src/background.js` | Service worker: chạy hàng đợi, mở tab, giao prompt, báo trạng thái mục |
| `src/adapter/core.js` | Phần chung của adapter: chờ ô chat, gõ, gửi, chờ xong, Copy, báo tiến độ, kiểm selector |
| `src/adapter/answer.js` | Luật "đã trả lời xong" của DeepSeek, khớp nhãn nút, dựng lại khối code |
| `src/adapter/clipboard-hook.js` | Chặn lần copy của nút Copy trong lượt chạy (cả ba nhà cung cấp) |
| `src/adapter/selector-kit.js` | Dựng bộ selector cho nhà cung cấp chỉ dùng CSS selector (Gemini, Grok) |
| `src/providers/<id>/selectors.js` | **Selector mặc định** của giao diện nhà cung cấp, ghép với phần bạn dán ở trang cài đặt |
| `src/providers/<id>/adapter.js` | Khai báo của nhà cung cấp: các lượt trả lời, `isDone`, nút Copy; DeepSeek thêm bước đặt DeepThink/Search |
| `src/popup/` | Popup: hàng đợi, nhà cung cấp từng mục |
| `src/options/` | Trang **Cài đặt**: nhà cung cấp mặc định; dán, kiểm tra, lưu selector cho từng nhà cung cấp |

## Khi một trang AI đổi giao diện: tự dán selector

Lỗi "Giao diện … đã đổi (tên phần)" cho biết phần nào không tìm thấy. Bạn sửa được mà không cần sửa code:

1. Bấm **icon ConceptFlow AI Bridge** trên thanh công cụ Chrome, sang tab **Cài đặt**, bấm **Mở cài đặt**. Nếu chưa ghim icon thì vào `chrome://extensions` → **Chi tiết** → **Tuỳ chọn tiện ích**.
2. Ở ô **Selector của**, chọn nhà cung cấp cần sửa.
3. Mở một tab của nhà cung cấp đó (đã đăng nhập, nên đang có một câu trả lời đã xong), rồi bấm **Kiểm tra trên tab của nhà cung cấp**. Mỗi ô hiện "Tìm thấy" hoặc "Không tìm thấy"; phần tìm thấy được viền đỏ vài giây trên tab (nút Copy viền xanh). Với DeepSeek, trang cũng cho biết nút DeepThink và Search đang bật hay tắt và class của nút.
4. Ô nào "Không tìm thấy", hoặc tìm thấy sai phần tử:
   - Sang tab nhà cung cấp, nhấn **F12**.
   - Bấm biểu tượng mũi tên chọn phần tử (Cmd + Shift + C), rồi bấm vào phần cần tìm.
   - Trong tab Elements, chuột phải dòng HTML đang chọn → **Copy** → **Copy selector**.
   - Dán vào ô tương ứng.
5. Bấm **Kiểm tra** lại cho đến khi mọi ô đúng, rồi bấm **Lưu**. Lượt chạy sau dùng selector mới. **Về mặc định** xoá hết phần bạn dán của nhà cung cấp đang chọn.

Mẹo:

- Selector do "Copy selector" sinh ra thường chứa class mã băm, nên sẽ hỏng khi trang cập nhật. Nếu được, rút gọn lại còn id, `data-testid`, tên thẻ hoặc `role`.
- Gemini và Grok: ô **Lượt trả lời** khớp mọi lượt của AI trên trang; extension lấy lượt cuối. Ô **Dấu hiệu đã xong** và **Nút Copy** được tìm *bên trong* lượt cuối.
- Grok: nút Copy tìm theo nhãn tiếng Anh `aria-label="Copy response"`, selector duy nhất phụ thuộc chữ. Đổi ngôn ngữ giao diện Grok thì sửa ô **Nút Copy**.
- DeepSeek, ô **Class khi nút đang bật**: bật nút Search bằng tay, bấm Kiểm tra, xem class của nút. Phần chỉ xuất hiện khi nút bật (ví dụ `ds-toggle-button--selected`) là thứ cần điền; ô này là biểu thức chính quy, nhiều giá trị ngăn cách bằng `|`.
- DeepSeek, **Nút gửi** để trống thì extension bấm nút cuối cùng trong khung chat; nếu không gửi được thì thử Enter.

Selector mặc định nằm trong `src/providers/<id>/selectors.js`. Nếu một selector bạn dán chạy ổn, hãy báo để cập nhật mặc định cho mọi người.

### Xem log một lượt chạy

Mở tab nhà cung cấp mà extension đã tạo, nhấn F12, chọn tab **Console** và lọc theo `ai-bridge`. Mỗi bước đều có một dòng log: ô chat tìm thấy, trạng thái DeepThink/Search (DeepSeek), nút gửi đã bấm, số lượt trả lời trước và sau khi gửi, cách lấy câu trả lời, và lỗi nếu có. Gửi các dòng này khi cần hỗ trợ.

## Phát triển và test

Phần logic thuần (giao thức, lượt chạy, hàng đợi và các lời gọi API của nó, bảng nhà cung cấp, luật đọc câu trả lời, ghép selector) và adapter chạy trên trang giả của từng nhà cung cấp (dựng từ selector mặc định) có test chạy bằng Node 20+:

```bash
cd services/web-gui && npm ci     # một lần: test adapter dùng jsdom của web-gui
cd ../../tools/ai-bridge-extension
node --test test/*.test.js
```

Phía ConceptFlow, các hook `useAiBridge`, `useAiQueue`, `useQueuedExternalRun`, hộp thoại `ExternalAiDialog` và bước Nghiên cứu được test trong `services/web-gui` (`npx vitest run`), với một extension giả (`tests/helpers/fakeAiBridge.ts`).

Phần điều khiển trang thật của ba nhà cung cấp **không có test tự động**, vì cần đăng nhập và giao diện của họ thay đổi. Mỗi lần sửa adapter hoặc selector, chạy thử một mục thật ở từng nhà cung cấp.

Một lượt AI mới của ConceptFlow tự có nút hàng đợi khi nó có hộp thoại "Dùng AI ngoài" (`ExternalAiDialog`). Chỗ khác thì dùng `useQueuedExternalRun`, hoặc `useAiQueue().add` cho nhiều mục.

Thêm trang AI khác: thêm một dòng vào `src/providers.js`, viết `src/providers/<id>/selectors.js` + `adapter.js` theo mẫu Gemini/Grok (`adapter/core.js` lo phần chung), thêm `content_scripts` và `host_permissions` trong `manifest.json`, và thêm id vào `AiBridgeProvider` của web-gui.

## Lưu ý

Extension tự động thao tác trên giao diện web của DeepSeek, Gemini và Grok thay bạn. Hãy dùng ở mức Creator bấm thêm mục khi cần, đúng như cách bạn tự làm bằng tay, và tự kiểm tra điều khoản sử dụng của từng nhà cung cấp.
