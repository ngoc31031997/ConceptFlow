# ADR-0043: Nghiên cứu có nguồn bằng prompt chép ra công cụ ngoài; backend kiểm đoạn trích và chặn địa chỉ nội bộ

## Status
Accepted

## Date
2026-10-17

## Stage
Application Design (CR-072, cập nhật Tiền kỳ · Nội dung)

## Context
Yêu cầu đã duyệt (câu 20 A) giữ trường "Kiểm chứng" và để "nghiên cứu có nguồn thật" ngoài CR-072. Ngày 2026-10-17 Creator đưa bước Nghiên cứu và kiểm chứng vào luồng (FR-28): dữ kiện phải có nguồn, phân biệt sự thật, giả thuyết, điểm chưa biết, ngộ nhận, và kịch bản tham chiếu dữ kiện theo mã. Hệ thống chưa có công cụ tra web; model trong Hive không tự mở trang web. Câu hỏi là lấy nguồn từ đâu và làm sao biết nguồn có thật.

## Options Considered
### Option A: Prompt chép ra công cụ nghiên cứu ngoài, dán JSON lại, backend tải từng link kiểm đoạn trích
- What it is: hệ thống dựng prompt (brief, câu hỏi, luật nguồn, mẫu JSON `conceptflow.research.v1`); Creator chạy ở ChatGPT, Perplexity hay Gemini Deep Research rồi dán câu trả lời; authoring-service đọc khối JSON, gộp theo mã, tải trang của từng link và tìm đoạn trích trong trang.
- Strengths: dùng được công cụ tra web tốt nhất Creator đang có mà không cần khoá API; đoạn trích được máy kiểm nên nguồn bịa lộ ra ngay; không tốn token của Hive cho việc tra.
- Trade-offs: Creator phải chép và dán; trang trả phí hoặc chặn bot thành "không tải được"; backend phải tải URL do người ngoài đưa vào (rủi ro SSRF).

### Option B: Gọi API tìm kiếm ngay (Tavily, Wikipedia, Semantic Scholar)
- What it is: backend tự tra theo câu hỏi.
- Strengths: không phải chép, dán.
- Trade-offs: thêm khoá API, chi phí và một phụ thuộc ngoài; chất lượng tổng hợp kém công cụ Deep Research; vẫn phải kiểm đoạn trích.

### Option C: AI của Hive tự liệt kê dữ kiện, không có nguồn
- What it is: như trường "Kiểm chứng" hiện nay.
- Strengths: không có việc mới.
- Trade-offs: không trả lời được câu "nguồn ở đâu"; đó là đúng điều Creator muốn sửa. Vẫn giữ làm đường đi khi Creator bỏ qua bước (nhãn "chưa có nguồn").

## Decision
A, với nguồn tra cứu đặt sau một cổng (`ResearchSource`) để B cắm vào sau mà không đổi bảng dữ kiện. C là cách bước chạy khi Creator bỏ qua Nghiên cứu.

Việc tải trang (`SourceFetcher`, authoring-service) theo các luật sau; chi tiết số liệu chốt ở Functional Design của U3b:
- Chỉ `http` và `https`, cổng 80 và 443.
- Phân giải tên miền trước khi kết nối và từ chối địa chỉ loopback, mạng riêng (RFC 1918, ULA IPv6), link-local, địa chỉ metadata đám mây, `0.0.0.0`, và mọi tên dịch vụ trong mạng Docker; kết nối tới đúng địa chỉ đã kiểm, không phân giải lại.
- Mỗi lần chuyển hướng kiểm lại như một link mới; giới hạn số lần chuyển hướng.
- Giới hạn thời gian, dung lượng tải về và số trang tải cùng lúc; chỉ đọc nội dung chữ (HTML, văn bản); kiểu khác thành "không tải được" kèm lý do.
- Không gửi cookie hay thông tin đăng nhập; không chạy script của trang.
- So đoạn trích sau khi chuẩn hoá chữ (bỏ thẻ HTML, gộp khoảng trắng, thống nhất dấu nháy và gạch); không khớp thì nêu câu gần giống nhất nếu có.

## Rationale
Creator chọn cách chép prompt vì công cụ Deep Research bên ngoài tra tốt hơn và không cần khoá mới. Kiểm đoạn trích bằng máy là phần làm cho nguồn đáng tin: một nguồn bịa không có đoạn trích đó trong trang. Cổng `ResearchSource` giữ đường mở cho API tìm kiếm mà không đổi dữ liệu. Tải URL người ngoài đưa vào là đường SSRF điển hình, nên luật chặn địa chỉ nội bộ là một phần của quyết định, không để tuỳ chi tiết code.

## Consequences
- **Positive**: dữ kiện có nguồn kiểm được; kịch bản, ghi chú bàn giao và bước Rà soát dùng chung một bảng dữ kiện có mã.
- **Negative / Accepted Trade-offs**: authoring-service lần đầu gọi ra Internet; trang trả phí không kiểm được; Creator làm thêm thao tác chép, dán.
- **Follow-ups**: cắm API tìm kiếm qua `ResearchSource` (backlog); kiểm chứng lại sau khi chốt (backlog, Creator: chưa cần).

## Related
- Design artifact: `aidlc-docs/inception/application-design/cr-072/components.md`, `component-methods.md`, `services.md`.
- Yêu cầu: `aidlc-docs/inception/requirements/cr-072-2d-film-pipeline.md` FR-28, tiêu chí chấp nhận 11–13.
