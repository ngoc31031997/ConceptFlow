# CR-118 · AI Bridge: chỉ chạy qua hàng đợi, thêm Gemini và Grok

**Trạng thái:** chờ Creator duyệt · **Ngày:** 2026-11-20 · **Thiết kế:** [cr-118-ai-bridge-providers-design.md](../../construction/plans/cr-118-ai-bridge-providers-design.md)

## Bối cảnh

Extension `tools/ai-bridge-extension` hiện có hai đường chạy: bấm "Chạy bằng DeepSeek" (chạy ngay, một lượt, câu trả lời về tab đang mở) và hàng đợi (chạy lần lượt, kể cả khi tab ConceptFlow đã đóng). Chỉ có DeepSeek.

## Yêu cầu (đã chốt với Creator)

| # | Yêu cầu |
|---|---|
| FR-1 | Mọi lượt AI ngoài (mọi hộp thoại "Dùng AI ngoài" ở bước 1–7, các nút chạy bằng DeepSeek, bước 3 · Nghiên cứu) đều **thêm một mục vào hàng đợi**. Bỏ đường chạy trực tiếp. |
| FR-2 | Ba nhà cung cấp: **DeepSeek** (`chat.deepseek.com`), **Gemini** (`gemini.google.com`), **Grok** (`grok.com`). Không có Kimi. |
| FR-3 | **Cài đặt** của extension chọn nhà cung cấp mặc định cho mọi mục; ban đầu là DeepSeek. |
| FR-4 | **Popup Hàng đợi** cho đổi nhà cung cấp của từng mục khi mục đang chờ hoặc lỗi; mục đang chạy thì khoá. Mục chưa chọn riêng theo mặc định **tại lúc tới lượt**; popup ghi rõ "Gemini (mặc định)" hay "Grok". |
| FR-5 | Mỗi lượt mở **một chat mới** trên trang của nhà cung cấp, chạy xong thì đóng tab (giữ tab khi lỗi), như DeepSeek hiện nay. |
| FR-6 | Kết quả **tự về ConceptFlow**: extension gửi câu trả lời thẳng vào API (`X-External-AI: paste`). Màn hình đang mở tự tải lại phần vừa lưu; hộp thoại đang mở theo dõi mục của nó và tự đóng khi xong. |
| FR-7 | Mục lỗi (không đọc được câu trả lời, hết giờ, tab bị đóng, giao diện đổi) dừng hàng đợi như hiện nay; hộp thoại đang mở hiện lý do và câu trả lời gốc trong ô dán để Creator sửa tay. |
| FR-8 | DeepSeek giữ cách bật DeepThink / Search theo từng bước. **Gemini và Grok không bị extension đổi chế độ hay model**: chạy với chế độ Creator đặt sẵn trên trang. |
| FR-9 | Mọi chỗ ghi "DeepSeek" trên giao diện ConceptFlow đổi thành chữ trung tính ("AI ngoài", "Thêm vào hàng đợi AI") hoặc tên nhà cung cấp thật của mục. |

## Ngoài phạm vi

- Chọn nhà cung cấp ngay trong hộp thoại ConceptFlow (chỉ chọn ở popup và Cài đặt).
- Gọi API của các nhà cung cấp (vẫn dùng trang web, tài khoản của Creator).
- Chạy song song nhiều mục.

## Rủi ro đã biết

- Selectors của Gemini và Grok lấy từ HTML Creator gửi ngày 2026-11-20; trang đổi giao diện thì phải sửa selector (trang Cài đặt cho ghi đè, như DeepSeek).
- Test tự động giả trang của nhà cung cấp; chạy thật trên Gemini và Grok cần Creator kiểm tay sau khi cài lại extension.
