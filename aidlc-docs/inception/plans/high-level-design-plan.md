# High-Level Design Plan

## Scope
Dựa trên `requirements.md` và `stories.md`, hệ thống gồm các mảng chức năng chính: Plugin/Content system, Script processing, Manim animation rendering, TTS, Video assembly, GUI, YouTube publishing, và Docker packaging. Đây là hệ thống **greenfield**, chạy **local trên một máy**, phục vụ **1 người dùng**, không có mục tiêu triển khai đa dịch vụ/cloud trong giai đoạn này.

## Execution Checklist
- [ ] Thu thập câu trả lời cho các câu hỏi kiến trúc bên dưới
- [ ] Phân tích câu trả lời, đặt câu hỏi follow-up nếu có mơ hồ/mâu thuẫn
- [ ] Tạo `system-context.md`
- [ ] Tạo `architecture-overview.md`
- [ ] Tạo `technology-direction.md`
- [ ] Tạo `integration-boundaries.md`
- [ ] Tạo `architectural-style.md`
- [ ] Tạo `high-level-design.md` tổng hợp
- [ ] Tạo ADR cho các quyết định kiến trúc quan trọng
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: System Context — Actor và hệ thống bên ngoài
Ngoài Creator (người dùng duy nhất) tương tác qua GUI, hệ thống có tương tác với hệ thống bên ngoài nào khác ngoài YouTube Data API không? (vd. dịch vụ TTS ngoài dù đã chọn offline, hệ thống lưu trữ file ngoài, v.v.)

A) Chỉ có YouTube Data API là hệ thống bên ngoài duy nhất — mọi thứ khác (TTS, render, storage) đều chạy nội bộ trong container
   - ✅ Strengths: đơn giản, ranh giới hệ thống rõ ràng, ít phụ thuộc mạng
   - ⚠️ Trade-offs: không có

B) Ngoài YouTube, còn cần tương tác với hệ thống lưu trữ ngoài (vd. cloud storage để backup project/video)

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Kiến trúc tổ chức hệ thống (Major Components / Macro Decomposition)
Hệ thống nên được tổ chức macro-level theo cách nào?

A) 💡 Suggested: Monolith module hóa (Modular Monolith) — một ứng dụng backend duy nhất (Python) chứa các module tách biệt rõ ràng (Plugin System, Script Parser, Renderer, TTS, Video Assembler, YouTube Publisher), cùng chạy trong 1 process/container, giao tiếp nội bộ qua Python function/interface calls; GUI là một layer riêng (web frontend) gọi vào backend qua local HTTP API
   - ✅ Strengths: đơn giản triển khai (đúng với yêu cầu Docker local, 1 người dùng), dễ debug, không cần quản lý nhiều service, phù hợp quy mô dự án cá nhân
   - ⚠️ Trade-offs: khó scale ngang nếu sau này cần xử lý nhiều video song song trên nhiều máy; nếu không module hóa cẩn thận, ranh giới plugin dễ bị xói mòn

B) Microservices tách biệt (vd. Render Service, TTS Service, Publisher Service chạy như các service độc lập, giao tiếp qua HTTP/message broker)
   - ✅ Strengths: mỗi service scale/deploy độc lập, cô lập lỗi tốt
   - ⚠️ Trade-offs: phức tạp vận hành không cần thiết cho 1 người dùng chạy local qua Docker; overhead lớn so với lợi ích ở quy mô này

C) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 3: Architectural Style (BẮT BUỘC)
Phong cách tổ chức code ở tầng hệ thống nên theo kiểu nào?

A) 💡 Suggested: Hexagonal / Ports & Adapters — logic nghiệp vụ lõi (plugin resolution, scene model, orchestration pipeline) tách biệt hoàn toàn khỏi chi tiết công nghệ (Manim rendering, TTS engine cụ thể, YouTube API, file system), giao tiếp qua "port" (interface); mỗi công nghệ cụ thể là một "adapter" cắm vào
   - ✅ Strengths: khớp tự nhiên với yêu cầu kiến trúc plugin/pluggable (NFR1) — content type plugin, TTS provider, đều có thể là adapter thay thế được; dễ test logic lõi độc lập với Manim/TTS thật
   - ⚠️ Trade-offs: nhiều interface/abstraction hơn so với code thẳng, cần kỷ luật giữ ranh giới port/adapter

B) Layered / N-tier truyền thống (Presentation → Business/Service → Data Access)
   - ✅ Strengths: quen thuộc, dễ hiểu, ít abstraction hơn
   - ⚠️ Trade-offs: không tự nhiên hỗ trợ yêu cầu "pluggable content type" và "pluggable TTS" bằng Hexagonal — dễ dẫn đến việc business logic phụ thuộc trực tiếp vào Manim/TTS cụ thể

C) Domain-Driven Design với bounded context riêng cho từng domain giáo dục (Programming context, English context sau này, v.v.)
   - ✅ Strengths: rất rõ ràng khi có nhiều domain giáo dục hoạt động song song
   - ⚠️ Trade-offs: over-engineering ở giai đoạn hiện tại khi chỉ có 1 domain (lập trình) được implement — DDD bounded context có giá trị hơn khi đã có ≥2 domain thực tế

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Technology Direction — Ngôn ngữ & Framework
Manim là thư viện Python, nên backend chắc chắn dùng Python. Cần xác nhận hướng công nghệ cho các phần còn lại:

A) 💡 Suggested: Backend Python (FastAPI cho local HTTP API) + Frontend Web (React hoặc tương đương) chạy trong cùng container, giao tiếp qua REST/WebSocket cho tiến trình render real-time
   - ✅ Strengths: FastAPI nhẹ, phù hợp expose API cục bộ cho GUI; WebSocket hỗ trợ tốt việc hiển thị tiến trình render (Story C6); hệ sinh thái React mature cho GUI phức tạp (soạn script, preview video)
   - ⚠️ Trade-offs: cần đóng gói cả Node.js build tooling cho frontend trong Docker image, tăng độ phức tạp build

B) Desktop app dùng Python thuần (vd. PyQt/PySide hoặc Tkinter) — không cần web server, không cần frontend riêng
   - ✅ Strengths: đơn giản hóa Docker packaging (không cần Node.js/build frontend), toàn bộ là Python
   - ⚠️ Trade-offs: hiển thị GUI từ trong Docker container ra máy host phức tạp hơn nhiều (cần X11 forwarding/VNC trên macOS/Linux), trải nghiệm preview video kém hơn web

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Distributed Communication — GUI ↔ Backend
Vì có ít nhất 2 thành phần macro (GUI và Backend pipeline), cần xác định cách chúng giao tiếp:

A) 💡 Suggested: Đồng bộ (Synchronous) REST API cho các thao tác cấu hình (soạn script, chọn plugin, cấu hình publish) + Bất đồng bộ qua WebSocket/Server-Sent Events riêng cho luồng cập nhật tiến trình render (vì render chạy lâu, dạng long-running job)
   - ✅ Strengths: REST đơn giản cho CRUD-like operations; WebSocket phù hợp tự nhiên cho progress update real-time (Story C6) mà không cần GUI polling liên tục
   - ⚠️ Trade-offs: phải duy trì 2 kiểu giao tiếp thay vì 1

B) Chỉ REST API, GUI polling định kỳ để lấy trạng thái render
   - ✅ Strengths: đơn giản nhất, chỉ 1 kiểu giao tiếp
   - ⚠️ Trade-offs: polling tốn tài nguyên hơn, độ trễ cập nhật tiến trình cao hơn WebSocket

C) Other (please describe after [Answer]: tag below)

[Answer]: A nhưng Server-Sent Events

### Question 6: API Gateway
Hệ thống chỉ có một backend service duy nhất (theo đề xuất Modular Monolith ở Câu 2) được GUI gọi trực tiếp — không có nhiều service độc lập cần một gateway hợp nhất. Bạn xác nhận điều này đúng không?

A) Đúng — không cần API Gateway, GUI gọi thẳng backend service duy nhất qua REST/WebSocket nội bộ

B) Không đúng — tôi dự tính có nhiều service độc lập cần gateway (vui lòng mô tả trong Other)

C) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 7: Non-Functional Drivers (Scale/Availability)
Đã xác nhận ở requirements.md: không yêu cầu tốc độ render, chạy local, 1 người dùng. Có yếu tố phi chức năng nào khác cần cân nhắc ảnh hưởng đến kiến trúc tổng thể không?

A) Không — các NFR đã nêu trong requirements.md (NFR1-NFR7) là đầy đủ cho High-Level Design

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8: Deployment Topology (Conceptual)
Xác nhận mô hình triển khai khái niệm cho giai đoạn này:

A) Single-container Docker trên máy cá nhân, chạy toàn bộ backend + frontend trong 1 container (hoặc docker-compose với 2 container: backend + frontend, cùng trên 1 máy)
   - ✅ Strengths: khớp với NFR3 (Docker local), đơn giản để khởi chạy bằng 1 lệnh (Story F1)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
