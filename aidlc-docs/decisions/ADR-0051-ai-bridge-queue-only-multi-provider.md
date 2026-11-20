# ADR-0051: AI Bridge chạy mọi lượt qua hàng đợi, nhiều nhà cung cấp

**Trạng thái:** Proposed · **Ngày:** 2026-11-20

## Bối cảnh

Extension AI Bridge có hai đường: chạy trực tiếp (trang gửi prompt, nhận câu trả lời, tự dán) và hàng đợi (worker tự gọi API ConceptFlow ở chế độ `prompt` rồi `paste`). Hai đường làm cùng một việc nhưng có hai bộ code, hai cách báo lỗi. Creator muốn dùng thêm Gemini và Grok, chọn mặc định trong Cài đặt và đổi theo từng mục.

## Quyết định

1. Chỉ còn hàng đợi. Trang ConceptFlow mô tả request của bước (`X-External-AI: describe`, đã có) và thêm vào hàng đợi; worker dựng prompt, chạy, gửi câu trả lời về API. Bước 3 · Nghiên cứu (prompt dựng ở trang, đọc qua `POST /research/paste`) dùng loại mục thứ hai: prompt cố định + request đọc kết quả.
2. Nhà cung cấp là một bảng trong extension (`id`, tên, URL chat mới, host, selectors, chế độ hỗ trợ). Adapter dùng chung một lõi (gõ prompt, gửi, chờ xong, bấm Copy, đọc clipboard); mỗi nhà cung cấp chỉ khai báo selectors và cách nhận biết "đã xong".
3. Mục mang `provider` (null = theo mặc định lúc chạy). Trang ConceptFlow không chọn nhà cung cấp.
4. Giao thức trang ↔ extension lên `v: 2` (bỏ `run`/`result`/`progress`, thêm trạng thái theo mục). Trang gặp extension `v: 1` thì báo cập nhật extension thay vì chạy sai.

## Hệ quả

- Đóng hộp thoại không còn huỷ lượt chạy; mục vẫn ở hàng đợi và kết quả vẫn được lưu.
- Bỏ được `useAiBridge.run`, `useDeepSeekRun` và nửa trực tiếp của worker.
- Gemini và Grok chạy với chế độ Creator tự đặt trên trang; extension không bấm menu chọn chế độ (menu đổi thường và theo ngôn ngữ tài khoản).
- Contract `docs/contracts/ai-bridge-v1.md` thay bằng `ai-bridge-v2.md`.
