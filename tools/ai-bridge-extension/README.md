# ConceptFlow AI Bridge

Extension Chrome giúp ConceptFlow chạy prompt trên DeepSeek web thay cho việc chép/dán tay hoặc gọi API:

- **Mọi hộp thoại "Dùng AI ngoài" ở bước 1–7**: viết ý tưởng, đánh giá, brief, câu chuyện, beat, kịch bản…
- **Bước 3 · Nghiên cứu**: DeepSeek tra web và trả bảng dữ kiện.

Bấm một nút; extension làm phần còn lại và ConceptFlow tự đọc kết quả.

- Dùng tài khoản DeepSeek web của bạn, không cần API key, không tốn tiền API.
- Hiện hỗ trợ: **DeepSeek** (`chat.deepseek.com`). Hộp thoại "Dùng AI ngoài" bật **DeepThink**; ô prompt nghiên cứu ở bước 3 bật **DeepThink + Search**. Extension tự bật hoặc tắt hai nút này theo từng lượt.
- Dùng được cả khi ConceptFlow chưa cấu hình AI (không có API key).
- Không gửi dữ liệu đi đâu khác ngoài tab DeepSeek. Extension không có server riêng, không đọc trang nào khác ngoài ConceptFlow (`localhost:3000`) và DeepSeek.

## Cài đặt

Cần Google Chrome 111 trở lên (hoặc Edge, Brave, Arc, tức là trình duyệt dùng nhân Chromium).

1. Mở `chrome://extensions`.
2. Bật **Developer mode** (góc trên bên phải).
3. Bấm **Load unpacked** và chọn thư mục này: `tools/ai-bridge-extension` trong repo ConceptFlow.
4. Extension **ConceptFlow AI Bridge** hiện trong danh sách. Có thể ghim nó lên thanh công cụ nhưng không bắt buộc, vì extension không có popup.
5. Mở `https://chat.deepseek.com` và **đăng nhập** một lần. Extension dùng phiên đăng nhập này.
6. Tải lại tab ConceptFlow nếu nó đang mở, để extension được nạp vào trang.

Cập nhật extension sau khi kéo code mới: vào `chrome://extensions`, bấm nút ↻ trên thẻ của extension, rồi tải lại tab ConceptFlow và tab DeepSeek.

ConceptFlow chạy ở cổng khác `3000`? Sửa hai dòng `matches` của `bridge.js` trong `manifest.json` rồi tải lại extension.

## Sử dụng

### Mọi nút "Dùng AI ngoài" (bước 1–7)

Mọi lượt AI ở bước 1–7 (viết ý tưởng, đánh giá, brief, câu hỏi nghiên cứu, câu chuyện, beat, kịch bản, rà soát…) đều có nút hoặc lựa chọn **Dùng AI ngoài**.

1. Bấm **Dùng AI ngoài**. Hộp thoại hiện prompt cần chạy.
2. Bấm **Chạy bằng DeepSeek**, nằm ngay dưới prompt. Hộp thoại hiện tiến độ ("DeepSeek đang suy nghĩ…") và nút Huỷ.
3. Khi xong, câu trả lời được dán vào ô "Dán kết quả" và hộp thoại tự bấm **Đọc kết quả**. Bước chạy tiếp như khi gọi AI của hệ thống, rồi hộp thoại đóng.
4. Nếu bước không dùng được câu trả lời, hộp thoại báo lý do và câu trả lời vẫn nằm trong ô. Bạn sửa rồi bấm **Đọc kết quả**, hoặc chạy lại.

Ở các hộp thoại này DeepSeek chỉ bật **DeepThink**, không tra web. Đóng hộp thoại thì ConceptFlow ngừng chờ; tab DeepSeek vẫn mở.

Ví dụ bước 1 · Ý tưởng: tạo project, rồi ở thẻ ② bấm **Dùng AI ngoài** để viết ý tưởng; làm lại thì chọn AI ngoài trong modal **Làm lại kèm ghi chú**.

### Bước 3 · Nghiên cứu

1. Mở project, vào bước **3 · Nghiên cứu**, tạo câu hỏi ở ô ①.
2. Ở ô ② **Prompt nghiên cứu**, bấm **Chạy bằng DeepSeek**. Nếu không thấy nút này mà thấy dòng "Cài extension…", xem phần Xử lý lỗi.
3. Extension mở một tab DeepSeek **ở nền** (bạn vẫn ở lại ConceptFlow), bật DeepThink + Search, dán prompt và gửi.
4. Ô ② hiện tiến độ:
   - "Đang mở DeepSeek trong một tab nền…"
   - "Đang bật DeepThink + Search và gửi prompt…"
   - "DeepSeek đang suy nghĩ và tra web…": thường mất vài phút, tối đa 15 phút.
   - "Đang lấy câu trả lời…"
5. Khi xong, câu trả lời được điền vào ô ③ và ConceptFlow tự bấm **Đọc kết quả**. Dữ kiện hiện ở bảng dữ kiện và hệ thống bắt đầu kiểm link như khi bạn dán tay.
6. Tab DeepSeek được giữ lại để bạn xem lại hoặc hỏi tiếp. Đóng nó khi không cần nữa.

Muốn dừng: bấm **Huỷ** cạnh dòng tiến độ. ConceptFlow ngừng chờ; tab DeepSeek vẫn mở và DeepSeek có thể vẫn trả lời tiếp trong tab đó. Nếu muốn, bạn chép và dán tay câu trả lời đó.

Mỗi lần chỉ chạy được một lượt. Mỗi lượt là một cú bấm của bạn; extension không tự chạy lại hay chạy hàng loạt.

Nếu ConceptFlow không đọc được câu trả lời (ví dụ DeepSeek không trả khối JSON), câu trả lời vẫn nằm trong ô ③ kèm lý do. Bạn sửa rồi bấm **Đọc kết quả** lại.

## Xử lý lỗi

| Hiện tượng | Nguyên nhân thường gặp | Cách xử lý |
|---|---|---|
| Không có nút "Chạy bằng DeepSeek" (bước Nghiên cứu hiện dòng "Cài extension…") | Extension chưa cài, bị tắt, hoặc tab ConceptFlow mở trước khi cài | Kiểm tra `chrome://extensions`, rồi tải lại tab ConceptFlow |
| "Extension không phản hồi" | Extension vừa được tải lại nhưng trang thì chưa | Tải lại tab ConceptFlow |
| "Chưa đăng nhập DeepSeek" | Phiên DeepSeek hết hạn | Mở chat.deepseek.com, đăng nhập, chạy lại |
| "Giao diện DeepSeek đã đổi… (nút Search)" hoặc tên phần khác | DeepSeek đổi giao diện | Dán tay như cũ; sửa selector (xem dưới) |
| "Đang có một lượt chạy DeepSeek khác" | Một lượt trước chưa xong | Chờ, hoặc bấm Huỷ ở tab đã chạy lượt đó |
| Tiến độ đứng ở "DeepSeek đang suy nghĩ…" rất lâu | Chrome làm chậm tab nền | Bấm sang tab DeepSeek một lần rồi quay lại; câu trả lời sẽ được lấy khi xong |
| "DeepSeek chưa trả lời xong sau 15 phút" | Câu hỏi quá dài, mạng chậm, DeepSeek quá tải | Xem tab DeepSeek; có câu trả lời thì chép và dán tay |

Xem log chi tiết:

- Phía DeepSeek: mở DevTools (F12) trong tab DeepSeek, tab Console, tìm dòng `ai-bridge`.
- Phía extension: ở `chrome://extensions`, bấm **service worker** trên thẻ extension.

## Cách hoạt động

```
ConceptFlow (localhost:3000)                    Chrome extension                       chat.deepseek.com (tab nền)
  useAiBridge ──postMessage──▶ bridge.js ──runtime──▶ background.js ──tabs.create──▶ adapter.js
             ◀──postMessage── bridge.js ◀──tabs.sendMessage── background.js ◀──runtime── adapter.js
```

1. Trang gửi `run` (có `request_id`) qua `window.postMessage`. Giao thức đầy đủ ở `docs/contracts/ai-bridge-v1.md`.
2. `background.js` (service worker) lưu lượt chạy vào `chrome.storage.session` và mở tab DeepSeek ở nền. Lưu vào storage vì Chrome có thể tắt service worker trong lúc DeepSeek đang trả lời.
3. `deepseek/adapter.js` nạp xong thì xin việc (`ready`). Nó chờ ô chat, đặt DeepThink và Search đúng chế độ lượt chạy yêu cầu (`modes`), ghi prompt vào ô chat rồi nhấn Enter.
4. Câu trả lời được coi là xong khi hàng nút dưới câu trả lời (Copy, Regenerate…) đã hiện và nội dung đứng yên 5 giây. Nếu không tìm thấy hàng nút thì cần nội dung đứng yên 60 giây.
5. Lấy câu trả lời bằng nút **Copy** của DeepSeek để giữ nguyên markdown và khối ```` ```json ````. `deepseek/clipboard-hook.js` chạy trong trang DeepSeek và chặn đúng lần copy đó, nên **clipboard thật của bạn không bị ghi đè**. Các lần bạn tự copy vẫn đi qua bình thường. Nếu không lấy được qua nút Copy, adapter đọc chữ trên trang, bỏ phần suy nghĩ của DeepThink và dựng lại các khối code.
6. Kết quả hoặc lỗi đi ngược về tab ConceptFlow đã gửi yêu cầu.

| File | Vai trò |
|---|---|
| `manifest.json` | Khai báo extension (Manifest V3), quyền `tabs`, `storage`, trang được nạp script |
| `src/protocol.js` | Định dạng message, mã lỗi và câu báo lỗi tiếng Việt, giới hạn thời gian |
| `src/jobs.js` | Luật của lượt chạy đang lưu (còn chặn lượt mới không, message có thuộc lượt này không) |
| `src/bridge.js` | Content script trên ConceptFlow: chuyển message trang ↔ service worker |
| `src/background.js` | Service worker: mở tab, giao prompt, chuyển tiến độ và kết quả |
| `src/deepseek/selectors.js` | **Mọi selector của giao diện DeepSeek** |
| `src/deepseek/answer.js` | Luật "đã trả lời xong", khớp nhãn nút, dựng lại khối code |
| `src/deepseek/adapter.js` | Điều khiển trang DeepSeek |
| `src/deepseek/clipboard-hook.js` | Chặn lần copy của nút Copy trong lượt chạy |

## Khi DeepSeek đổi giao diện

Lỗi "Giao diện DeepSeek đã đổi… (tên phần)" cho biết phần nào không tìm thấy. Chỉ cần sửa `src/deepseek/selectors.js`:

| Phần trong lỗi | Khoá trong `selectors.js` | Cách tìm selector mới |
|---|---|---|
| ô chat | `input` | Chuột phải vào ô chat → Inspect, lấy id hoặc tag |
| nút DeepThink, nút Search | `deepThinkLabels`, `searchLabels`, `clickable`, `selectedClass` | Xem chữ trên nút; xem class nào thêm vào khi nút bật |
| gửi prompt | `input` (ô chat không tự xoá sau Enter) | Kiểm tra DeepSeek còn gửi bằng Enter không |
| nội dung câu trả lời | `markdown`, `thinking` | Inspect câu trả lời: class bao khối markdown, và class bao phần suy nghĩ |

Không có hàng nút Copy thì vẫn chạy được, chỉ chậm hơn (đợi 60 giây). Muốn sửa thì xem khoá `iconButton`.

Dùng id, `role`, class `ds-…` hoặc chữ hiện trên nút. Đừng dùng class dạng mã băm (ví dụ `_7a8b9c`) vì chúng đổi mỗi lần DeepSeek cập nhật. Sửa xong, tải lại extension và tab DeepSeek.

## Phát triển và test

Phần logic thuần (giao thức, lượt chạy, luật đọc câu trả lời) có test chạy bằng Node 18+, không cần cài gì:

```bash
cd tools/ai-bridge-extension
node --test test/
```

Phía ConceptFlow, hook `useAiBridge`, hộp thoại `ExternalAiDialog` và bước Nghiên cứu được test trong `services/web-gui` (`npx vitest run tests/hooks/useAiBridge.test.tsx tests/components/ExternalAiDialog.test.tsx tests/pages/ResearchStepPage.test.tsx`), với một extension giả.

Phần điều khiển trang DeepSeek thật **không có test tự động**, vì cần đăng nhập và giao diện DeepSeek thay đổi. Mỗi lần sửa adapter hoặc selector, chạy thử một lượt thật trên bước Nghiên cứu.

Một lượt AI mới của ConceptFlow tự có nút DeepSeek khi nó có hộp thoại "Dùng AI ngoài" (`ExternalAiDialog`). Chỗ khác thì gọi `useAiBridge().run("deepseek", prompt, modes)`.

Thêm trang AI khác (ChatGPT, Gemini…): viết `src/<tên>/adapter.js` + `selectors.js` theo mẫu DeepSeek, thêm tên vào `PROVIDERS` trong `protocol.js`, thêm `content_scripts` và `host_permissions` trong `manifest.json`, và cho `background.js` mở đúng URL theo `provider`.

## Lưu ý

Extension tự động thao tác trên giao diện web DeepSeek thay bạn. Hãy dùng ở mức một lượt mỗi lần bấm, đúng như cách bạn tự làm bằng tay, và tự kiểm tra điều khoản sử dụng của DeepSeek.
