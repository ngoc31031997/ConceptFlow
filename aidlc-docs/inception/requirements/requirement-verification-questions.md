# Requirements Clarification Questions

Vui lòng trả lời tất cả câu hỏi bằng cách điền chữ cái lựa chọn sau tag `[Answer]:`. Nếu không có lựa chọn nào phù hợp, chọn "Other" và mô tả câu trả lời của bạn.

## Question 1: Mục tiêu cốt lõi của công cụ
Công cụ này chủ yếu dùng để làm gì?

A) Trình tạo video tự động end-to-end — người dùng nhập kịch bản/script (text hoặc code) và công cụ tự sinh toàn bộ video Manim (animation + giọng đọc + nhạc nền) mà không cần viết code Manim thủ công
   - ✅ Strengths: nhanh, phù hợp người không rành Manim, tối đa hóa năng suất
   - ⚠️ Trade-offs: khó tùy biến chi tiết animation, chất lượng phụ thuộc vào chất lượng "generator" (có thể cần LLM hỗ trợ sinh code Manim)

B) Bộ khung / thư viện tiện ích (framework wrapper trên Manim) — cung cấp các scene/component tái sử dụng được (ví dụ: hiển thị code có syntax highlight, animation minh họa thuật toán, sơ đồ cấu trúc dữ liệu) để người dùng tự viết script Python lắp ráp video nhanh hơn
   - ✅ Strengths: linh hoạt, kiểm soát chi tiết animation, dễ mở rộng
   - ⚠️ Trade-offs: người dùng vẫn cần biết lập trình Python/Manim cơ bản

C) Pipeline sản xuất video hoàn chỉnh (production pipeline) — kết hợp cả (B) làm lõi animation, cộng thêm quản lý dự án video (storyboard, voice-over, render batch, xuất bản lên YouTube) như một hệ thống studio nội bộ
   - ✅ Strengths: bao quát toàn bộ quy trình từ ý tưởng đến video hoàn chỉnh, gần nhất với việc "xây kênh giống 3Blue1Brown"
   - ⚠️ Trade-offs: phạm vi lớn hơn nhiều, cần nhiều thời gian/công sức để xây dựng và vận hành

D) Other (please describe after [Answer]: tag below)

[Answer]: C 

## Question 2: Loại nội dung lập trình cần minh họa
Video sẽ tập trung minh họa loại nội dung lập trình nào là chính? (có thể chọn nhiều nếu cần, mô tả rõ trong Other)

A) Thuật toán & cấu trúc dữ liệu (sorting, graph traversal, cây, v.v. — animation trực quan hóa quá trình chạy)

B) Khái niệm lập trình tổng quát (OOP, recursion, con trỏ/pointer, memory model, concurrency, v.v.)

C) Trình bày & giải thích đoạn code cụ thể (code walkthrough với highlight, chú thích, animation từng bước biến đổi code)

D) Toán học ứng dụng cho lập trình (math visualization kiểu 3Blue1Brown gốc: vector, ma trận, xác suất, v.v. áp dụng vào CS)

E) Other (please describe after [Answer]: tag below)

[Answer]: loại nội dung lập trình chỉ là một implement cụ thể, mục đích vẫn là các chủ đề giáo dục ở các lĩnh vực trong cuộc sống. trước mắt là lập trình có thể tập trung vào A và B nhưng sau này phải mở rộng nếu đổi sang chủ đề học Tiếng Anh thì sao

## Question 3: Đối tượng người dùng của công cụ
Ai sẽ là người sử dụng công cụ này để tạo video?

A) Chỉ mình bạn (dùng cá nhân để tạo nội dung cho kênh riêng) — không cần giao diện phức tạp, ưu tiên tốc độ phát triển

B) Một nhóm nhỏ (team sản xuất nội dung) — cần một số quy trình chia sẻ (template, thư viện chung, review)

C) Công cụ mã nguồn mở / công khai cho cộng đồng người tạo nội dung giáo dục khác dùng — cần tài liệu, API ổn định, đóng gói cẩn thận

D) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question 4: Giao diện sử dụng (Interface)
Bạn muốn tương tác với công cụ như thế nào?

A) Command-line tool (CLI) — chạy script Python, tham số dòng lệnh, phù hợp workflow lập trình viên

B) 💡 Suggested: Thư viện Python (import as library) — chỉ cung cấp các class/hàm tiện ích, người dùng viết script Manim của riêng mình import từ thư viện này
   - ✅ Strengths: đơn giản nhất để xây, tích hợp tự nhiên vào workflow Manim hiện có
   - ⚠️ Trade-offs: không có "trải nghiệm sản phẩm" độc lập

C) Giao diện web/desktop (GUI) — cho phép chọn template, xem preview, chỉnh thông số mà không cần viết code

D) Other (please describe after [Answer]: tag below)

[Answer]: C

## Question 5: Xử lý giọng đọc / âm thanh (Voice-over & Audio)
Video có cần lồng tiếng thuyết minh không, và nếu có thì theo cách nào?

A) Không cần — chỉ video animation câm, người dùng tự lồng tiếng bên ngoài công cụ

B) Có, dùng Text-to-Speech (TTS) tự động (ví dụ tích hợp thư viện TTS để đồng bộ animation với giọng đọc sinh ra từ script)

C) Có, hỗ trợ đồng bộ hóa với file audio ghi âm sẵn (người dùng tự thu âm, công cụ giúp căn animation khớp với audio timing)

D) Other (please describe after [Answer]: tag below)

[Answer]: B

## Question 6: Ngôn ngữ nội dung video
Video sẽ được tạo bằng ngôn ngữ nào?

A) Tiếng Việt

B) Tiếng Anh

C) Cả hai, có thể chuyển đổi tùy video (đa ngôn ngữ)

D) Other (please describe after [Answer]: tag below)

[Answer]: C

## Question 7: Yêu cầu hiệu năng render
Manim render video có thể tốn nhiều thời gian/tài nguyên. Yêu cầu về hiệu năng render là gì?

A) Không quan trọng — render offline, chạy qua đêm cũng được, ưu tiên chất lượng

B) Cần render nhanh để lặp lại (iterate) nhanh trong quá trình soạn video (preview độ phân giải thấp, render nháp nhanh)

C) Cần khả năng render song song/hàng loạt (batch nhiều video hoặc nhiều scene cùng lúc, ví dụ trên cloud/GPU)

D) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question 8: Môi trường triển khai
Công cụ sẽ chạy ở đâu?

A) Chỉ chạy local trên máy cá nhân (macOS/Linux/Windows)

B) Cần chạy được trên server/cloud để render batch hoặc chia sẻ với team

C) Cả hai — local để phát triển/preview, cloud để render sản xuất

D) Other (please describe after [Answer]: tag below)

[Answer]: Docker local máy cá nhân trước, sau này tính sau.

## Question 9: Phạm vi giai đoạn đầu (MVP scope)
Với giai đoạn đầu tiên (MVP), bạn muốn ưu tiên gì nhất?

A) Một bộ thư viện scene/component Manim tái sử dụng cho các pattern phổ biến khi dạy lập trình (code highlight, thuật toán, cấu trúc dữ liệu) + vài ví dụ video mẫu

B) Một pipeline end-to-end tối thiểu: từ script/markdown → video hoàn chỉnh có giọng đọc, dù chỉ hỗ trợ 1-2 loại nội dung

C) Một CLI/tool scaffold dự án video (project generator) giúp khởi tạo cấu trúc file, cấu hình render, không quan tâm animation cụ thể

D) Other (please describe after [Answer]: tag below)

[Answer]: B

## Question: Security Extensions
Should security extension rules be enforced for this project?

A) Yes — enforce all SECURITY rules as blocking constraints (recommended for production-grade applications)

B) No — skip all SECURITY rules (suitable for PoCs, prototypes, and experimental projects)

X) Other (please describe after [Answer]: tag below)

[Answer]: B

## Question: Property-Based Testing Extension
Should property-based testing (PBT) rules be enforced for this project?

A) Yes — enforce all PBT rules as blocking constraints (recommended for projects with business logic, data transformations, serialization, or stateful components)

B) Partial — enforce PBT rules only for pure functions and serialization round-trips (suitable for projects with limited algorithmic complexity)

C) No — skip all PBT rules (suitable for simple CRUD applications, UI-only projects, or thin integration layers with no significant business logic)

X) Other (please describe after [Answer]: tag below)

[Answer]: C
