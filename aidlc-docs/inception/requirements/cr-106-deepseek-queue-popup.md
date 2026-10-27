# CR-106 · Hàng đợi chạy DeepSeek nằm trong popup của extension

## Yêu cầu
1. Bấm icon extension thì sổ ra một popup nhỏ (như nhiều extension khác) thay vì mở tab cài đặt. Popup có hai tab: **Hàng đợi** và **Cài đặt**.
2. Hàng đợi **nằm trong extension** và được quản lý ở popup. Ở mọi nơi có "Chạy bằng DeepSeek" (bước 1–7, gồm các nút làm lại và gợi ý) có thêm "Thêm vào hàng đợi". Các mục chạy tuần tự qua DeepSeek, mỗi mục tự đọc kết quả, thay vì Creator bấm từng lần.
3. Mục nào lỗi thì hàng đợi dừng và giữ mục đó; Creator sửa tay qua hộp thoại AI ngoài (đã dán sẵn câu trả lời) rồi cho chạy tiếp.

## Quyết định thiết kế
1. **Một mục trong hàng đợi là một request HTTP thường**, không phải hàm của trang. Mọi lời gọi AI ngoài của web-gui đều là `POST <gateway><path>` với body JSON và header `X-External-AI: prompt | paste` (`api/client.ts`, [`external-ai-v1.md`](../../../docs/contracts/external-ai-v1.md)). Mục gồm: tên hiển thị, `method`, `path`, `body`, địa chỉ gateway.
2. **Ghi lại request thay vì gửi**: thêm chế độ `describe` cho lời gọi AI ngoài. `apiFetch` gặp chế độ này thì ném `ExternalRequestDescribed` mang request mà không gửi (cùng kiểu với `ExternalPromptReady`). Nhờ vậy mọi nơi đang có `run(external)` thêm được vào hàng đợi mà không phải viết lại từng lời gọi: "Thêm vào hàng đợi" chỉ gọi `run({mode:"describe"})`.
3. **Extension tự chạy hàng đợi, kể cả khi tab ConceptFlow đã đóng hoặc tải lại.** Tới lượt một mục, service worker:
   1. gọi lại request với `X-External-AI: prompt` để lấy prompt, dựng đúng lúc đó từ trạng thái hiện tại của dự án (nên beat sau đọc được beat trước vừa viết);
   2. chạy prompt trên DeepSeek như hiện nay (mở tab, gửi, chờ, lấy câu trả lời, đóng tab);
   3. gọi lại request với `X-External-AI: paste` và `external_reply`; server đọc câu trả lời như khi Creator dán tay.
4. **Lưu bền** trong `chrome.storage.local` (hàng đợi, trạng thái từng mục, câu trả lời của mục lỗi). Tải lại trang hay khởi động lại Chrome không làm mất hàng đợi; mục đang chạy dở được chạy lại từ đầu.
5. **Trạng thái mục**: `waiting` → `running` → `done` hoặc `failed` (kèm lý do và câu trả lời nếu có). Lỗi dừng hàng đợi; các mục sau giữ nguyên cho tới khi Creator chọn Chạy tiếp, Thử lại mục lỗi hoặc Bỏ mục lỗi. Mục trùng (cùng method, path và body) chỉ được thêm một lần.
6. **Popup**:
   - Tab Hàng đợi: nút Chạy / Tạm dừng, từng mục với trạng thái và Huỷ / Bỏ, Xoá hàng; mục lỗi hiện lý do và nút "Sửa tay trong ConceptFlow".
   - Tab Cài đặt: dùng lại phần cài đặt selector hiện có (`options.js`); trang cài đặt đầy đủ vẫn mở được từ popup.
7. **"Sửa tay trong ConceptFlow"**: popup nhờ một tab ConceptFlow (mở mới nếu chưa có) mở hộp thoại AI ngoài của mục lỗi. Hộp thoại dựng từ chính request đã lưu, kèm câu trả lời và lý do lỗi; đọc xong thì mục được đánh dấu `done` và hàng đợi chạy tiếp được.
8. **Cho màn hình biết dữ liệu đã đổi**: mỗi khi một mục `done`, extension báo cho các tab ConceptFlow đang mở; các màn hình bước 1–7 nạp lại dữ liệu của dự án tương ứng.
9. **Giao thức** (thêm vào `docs/contracts/ai-bridge-v1.md`, message cũ giữ nguyên nên `v` vẫn là 1): trang → extension `queue_add` (`item`), `queue_open` (không bắt buộc); extension → trang `queue_item_done`, `queue_edit_failed` (`item_id`), `queue_state` (số mục, để hiện huy hiệu trên nút). Trang cũng hiện số mục đang chờ cạnh nút "Thêm vào hàng đợi".
10. **Quyền extension**: thêm `host_permissions` cho địa chỉ gateway, để service worker gọi API (cùng máy, `localhost`). Địa chỉ lấy từ `VITE_API_BASE_URL` của web-gui và đi kèm mỗi mục.

11. **Nút "viết tất cả"** (bàn giao tất cả beat, viết tất cả beat ở Beat sheet và Kịch bản) không có AI ngoài vì server chỉ nhận một prompt mỗi lượt ([`external-ai-v1.md`](../../../docs/contracts/external-ai-v1.md)). Cạnh mỗi nút đó thêm "Chạy bằng DeepSeek (hàng đợi)": nó thêm một mục cho mỗi beat chưa viết (đúng request một-beat đã có), theo thứ tự beat, bỏ qua beat đã có nội dung.

## Giới hạn đã biết (Creator cần biết trước)
- Hàng đợi chạy bằng chính tài khoản DeepSeek đã đăng nhập trên Chrome này; Chrome phải đang mở.
- Mỗi mục mở một tab DeepSeek rồi đóng, nên chạy nhiều mục thì màn hình nhảy sang DeepSeek nhiều lần.
- Mỗi lúc chỉ một lượt DeepSeek.
- Prompt dựng lúc tới lượt, nên nếu Creator sửa dự án trong lúc mục còn chờ thì mục đọc theo bản mới. Muốn ngăn thì tạm dừng hàng đợi.
- Popup Chrome đóng khi bấm ra ngoài; hàng đợi vẫn chạy.
- Hàng đợi dùng chung cho mọi dự án; mỗi mục ghi tên dự án.

## Thứ tự làm
1. Web-gui: chế độ `describe`, nút "Thêm vào hàng đợi" ở mọi nơi có "Chạy bằng DeepSeek", hộp thoại sửa tay dựng từ request đã lưu, nạp lại màn hình khi mục `done`, test.
2. Extension: hàng đợi và bộ chạy trong service worker (lưu bền), giao thức mới, test node.
3. Popup hai tab, quyền gateway, contract, README.

## Kiểm thử
Unit test web-gui với `fakeAiBridge`: `describe` ghi lại đúng request mà không gửi, nút thêm hàng đợi ở các nơi, hộp thoại sửa tay. Test node cho extension: chạy tuần tự đúng thứ tự, prompt dựng đúng lúc tới lượt, mục lỗi dừng hàng đợi, chạy tiếp sau khi sửa, huỷ và bỏ mục, không thêm trùng, hàng đợi còn sau khi worker khởi động lại. Không gọi AI thật.
