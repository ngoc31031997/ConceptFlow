# Requirements: Manim-based Educational Video Generation Tool

## Intent Analysis Summary

- **User Request**: "Tôi muốn xây dựng 1 tool dùng Manim để tạo các video dạy học về lập trình tương tự kênh 3Blue1Brown."
- **Request Type**: New Project (Greenfield)
- **Scope Estimate**: System-wide — nhiều component tương tác (content plugin system, animation engine trên nền Manim, TTS, video assembly, GUI, YouTube publishing, containerized runtime)
- **Complexity Estimate**: Complex — nhiều thành phần kỹ thuật khác nhau (rendering, audio synthesis, GUI, external API integration), cộng thêm ràng buộc kiến trúc mở rộng đa lĩnh vực ngay từ đầu

## 1. Mục tiêu sản phẩm

Xây dựng một **pipeline sản xuất video giáo dục hoàn chỉnh** (production pipeline), lấy Manim làm animation engine lõi, cho phép một người dùng cá nhân tạo video dạy học có chất lượng trực quan tương tự phong cách 3Blue1Brown — từ input dạng script/markdown đến video hoàn chỉnh có giọng đọc, và tự động đăng lên YouTube.

Miền nội dung ban đầu là **lập trình** (thuật toán, cấu trúc dữ liệu, khái niệm lập trình tổng quát), nhưng công cụ phải được thiết kế theo **kiến trúc plugin/pluggable content-type** ngay từ giai đoạn đầu, để có thể mở rộng sang các lĩnh vực giáo dục khác (ví dụ: dạy Tiếng Anh) trong tương lai mà không cần viết lại kiến trúc lõi.

## 2. Phạm vi (Scope)

### Trong phạm vi (In Scope)
- Kiến trúc plugin cho "content type" / "scene generator", với một plugin cụ thể được implement đầy đủ cho miền lập trình
- Animation engine dựa trên Manim, hỗ trợ minh họa thuật toán & cấu trúc dữ liệu (sorting, graph traversal, cây, v.v.) và khái niệm lập trình tổng quát (OOP, recursion, con trỏ, memory model, concurrency, v.v.)
- Sinh giọng đọc tự động bằng TTS mã nguồn mở/offline, hỗ trợ song ngữ Việt/Anh
- Giao diện Web/Desktop GUI cho phép người dùng tạo video mà không cần chạy lệnh dòng lệnh
- Lắp ráp video hoàn chỉnh (animation + audio + có thể có nhạc nền) thành file .mp4
- Tích hợp YouTube API để tự động đăng video sau khi render xong
- Đóng gói chạy bằng Docker trên máy cá nhân (local-first)

### Ngoài phạm vi (Out of Scope — cho giai đoạn này)
- Triển khai lên server/cloud để render batch hoặc chia sẻ với team (để giai đoạn sau)
- Chia sẻ công cụ cho nhóm/cộng đồng bên ngoài (chỉ phục vụ 1 người dùng cá nhân hiện tại)
- Implement plugin cho các miền giáo dục khác ngoài lập trình (chỉ thiết kế kiến trúc cho phép, không implement)
- Tối ưu hiệu năng render song song/hàng loạt trên GPU/cloud (render offline, không quan trọng tốc độ)
- Tích hợp TTS cloud trả phí (chọn TTS offline/mã nguồn mở)

## 3. Functional Requirements

### FR1 — Content Plugin Architecture
- FR1.1: Hệ thống PHẢI định nghĩa một interface/abstract base cho "content type" (loại chủ đề giáo dục), tách biệt phần logic sinh nội dung (script → cấu trúc scene) khỏi phần render animation dùng chung.
- FR1.2: Hệ thống PHẢI implement một plugin cụ thể cho miền **lập trình**, bao gồm ít nhất:
  - Sinh animation minh họa thuật toán & cấu trúc dữ liệu
  - Sinh animation minh họa khái niệm lập trình tổng quát (OOP, recursion, con trỏ, memory model, concurrency)
- FR1.3: Kiến trúc plugin PHẢI cho phép thêm plugin mới (domain giáo dục khác) mà không cần sửa đổi mã nguồn của core pipeline (animation engine, TTS, video assembly, publishing).

### FR2 — Script/Input Processing
- FR2.1: Hệ thống PHẢI nhận input dạng script/markdown mô tả nội dung video (lời thoại + cấu trúc nội dung cần minh họa).
- FR2.2: Hệ thống PHẢI phân tích (parse) script này thành cấu trúc scene có thể dùng để sinh animation Manim và giọng đọc tương ứng.

### FR3 — Animation Generation (Manim Core)
- FR3.1: Hệ thống PHẢI cung cấp các scene/component Manim tái sử dụng cho các pattern minh họa lập trình phổ biến: hiển thị code có syntax highlight, animation minh họa từng bước chạy thuật toán, sơ đồ trực quan hóa cấu trúc dữ liệu.
- FR3.2: Hệ thống PHẢI render animation thành video clip từ cấu trúc scene được sinh ra ở FR2.2.

### FR4 — Text-to-Speech (Voice-over)
- FR4.1: Hệ thống PHẢI tự động sinh giọng đọc từ nội dung lời thoại trong script, sử dụng công cụ TTS mã nguồn mở/offline (ví dụ Coqui TTS, Piper, hoặc tương đương).
- FR4.2: Hệ thống PHẢI hỗ trợ sinh giọng đọc cho cả tiếng Việt và tiếng Anh, tùy theo cấu hình của từng video.
- FR4.3: Hệ thống PHẢI đồng bộ hóa thời lượng giọng đọc với animation tương ứng (animation timing khớp với audio).

### FR5 — Video Assembly
- FR5.1: Hệ thống PHẢI ghép animation (video clip từ FR3.2) với audio (giọng đọc từ FR4.1) thành một file video hoàn chỉnh (.mp4).
- FR5.2: Hệ thống NÊN cho phép thêm nhạc nền (background music) tùy chọn.

### FR6 — GUI (Web/Desktop)
- FR6.1: Hệ thống PHẢI cung cấp giao diện đồ họa (Web hoặc Desktop) cho phép người dùng:
  - Nhập/soạn script nội dung video
  - Chọn content type plugin (mặc định: lập trình)
  - Chọn ngôn ngữ giọng đọc (Việt/Anh)
  - Khởi chạy quá trình render video
  - Xem trạng thái/tiến trình render
  - Xem/preview video kết quả sau khi hoàn tất
- FR6.2: GUI PHẢI hoạt động mà không yêu cầu người dùng chạy lệnh dòng lệnh trực tiếp để tạo video.

### FR7 — YouTube Publishing
- FR7.1: Hệ thống PHẢI tích hợp YouTube Data API để tự động tải video đã render lên kênh YouTube của người dùng.
- FR7.2: Hệ thống PHẢI cho phép người dùng cấu hình thông tin video khi đăng (tiêu đề, mô tả, tag, chế độ hiển thị: public/unlisted/private) trước khi upload.
- FR7.3: Hệ thống PHẢI xác thực với YouTube qua OAuth 2.0, lưu trữ credential an toàn trên máy local.

### FR8 — Containerized Runtime
- FR8.1: Toàn bộ hệ thống (GUI, pipeline render, TTS, dependency Manim/LaTeX/ffmpeg) PHẢI chạy được thông qua Docker trên máy cá nhân, không yêu cầu cài đặt thủ công từng dependency trên máy host.

## 4. Non-Functional Requirements

### NFR1 — Extensibility (Kiến trúc)
- Kiến trúc plugin (FR1) là ràng buộc kiến trúc bắt buộc, không phải tính năng tùy chọn — mọi thiết kế module lõi (animation engine, TTS, video assembly) PHẢI không phụ thuộc trực tiếp vào chi tiết miền lập trình.

### NFR2 — Performance
- Không có yêu cầu về tốc độ render — chấp nhận render offline, chạy lâu (kể cả qua đêm), ưu tiên chất lượng video hơn tốc độ.

### NFR3 — Deployment & Portability
- Hệ thống chạy local trên máy cá nhân qua Docker; chưa cần hỗ trợ triển khai cloud/server trong giai đoạn này (có thể bổ sung sau).

### NFR4 — Usability
- GUI phải đủ đơn giản để một người dùng không rành kỹ thuật sâu (nhưng có hiểu biết cơ bản) có thể tạo video từ đầu đến cuối mà không cần đọc code.

### NFR5 — Language Support
- Hệ thống PHẢI hỗ trợ song ngữ Việt/Anh cho giọng đọc; kiến trúc TTS nên cho phép mở rộng thêm ngôn ngữ khác sau này (hệ quả gián tiếp từ NFR1).

### NFR6 — Security
- Extension "Security Baseline" đã được người dùng chọn **KHÔNG kích hoạt** (phù hợp dự án cá nhân/prototype). Tuy nhiên, việc lưu trữ OAuth credential cho YouTube (FR7.3) vẫn PHẢI tuân theo thực hành tối thiểu an toàn (không hard-code secret, không commit credential vào source control) như một nguyên tắc code chung, không phải blocking constraint theo extension.

### NFR7 — Testing Approach
- Extension "Property-Based Testing" đã được người dùng chọn **KHÔNG kích hoạt**. Testing sẽ dùng phương pháp thông thường (unit test theo ví dụ cụ thể) thay vì property-based testing.

## 5. Extension Configuration

| Extension | Enabled | Decided At |
|---|---|---|
| Security Baseline | No | Requirements Analysis |
| Property-Based Testing | No | Requirements Analysis |

## 6. Key Architectural Considerations (chuyển tiếp sang High-Level Design)

- **Plugin boundary**: Cần xác định rõ interface giữa "content plugin" (miền lập trình) và "core pipeline" (Manim rendering, TTS, video assembly, YouTube publishing) — đây là quyết định kiến trúc quan trọng nhất, ảnh hưởng đến toàn bộ hệ thống.
- **TTS abstraction**: Dù chọn TTS offline cụ thể (vd. Coqui/Piper) cho MVP, nên cân nhắc abstraction tối thiểu để không khóa cứng vào một thư viện TTS duy nhất, hỗ trợ NFR5.
- **GUI ↔ Pipeline communication**: Cần quyết định GUI là web app (local server + browser) hay desktop app (Electron/PyQt/Tkinter), vì điều này ảnh hưởng đến cách đóng gói Docker và trải nghiệm người dùng.
- **Docker packaging**: Manim có nhiều dependency hệ thống (LaTeX, ffmpeg, Cairo/Pango) — cần thiết kế Dockerfile bao gồm đầy đủ các dependency này cùng GUI runtime.
- **YouTube OAuth flow trong Docker**: Cần thiết kế cách xử lý luồng xác thực OAuth (thường cần trình duyệt) khi ứng dụng chính chạy trong container.

## 7. Summary of Key Requirements

- Pipeline sản xuất video hoàn chỉnh: script → animation (Manim) + giọng đọc (TTS offline) → video → tự động đăng YouTube
- Kiến trúc plugin bắt buộc ngay từ MVP, chỉ implement plugin lập trình, nhưng chừa chỗ cho các domain giáo dục khác
- GUI Web/Desktop bắt buộc có trong MVP
- Chạy local qua Docker, không cần cloud/server ở giai đoạn này
- Không kích hoạt Security Baseline hay Property-Based Testing extensions
