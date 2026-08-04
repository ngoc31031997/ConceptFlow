# Application Design Plan

## Scope
Dựa trên High-Level Design (8 component: Web GUI, API Gateway, Content Plugin Service, Script Processing Service, Rendering Service, TTS Service, Video Assembly Service, Publisher Service), Application Design sẽ định nghĩa cụ thể interface/method-level (không đi sâu business logic — để dành Functional Design) và service orchestration layer tại Gateway.

## Execution Checklist
- [ ] Thu thập câu trả lời cho câu hỏi bên dưới
- [ ] Phân tích câu trả lời, đặt câu hỏi follow-up nếu có mơ hồ/mâu thuẫn
- [ ] Tạo `components.md`
- [ ] Tạo `component-methods.md`
- [ ] Tạo `services.md`
- [ ] Tạo `component-dependency.md`
- [ ] Tạo `application-design.md` tổng hợp
- [ ] Tạo ADR cho quyết định component boundary quan trọng (nếu có)
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Cơ chế chia sẻ artifact trung gian giữa các service
`integration-boundaries.md` xác định Rendering Service tạo animation clip, TTS Service tạo audio clip, Video Assembly Service cần cả hai để ghép. Các service này chạy trong container riêng biệt — cần cơ chế chia sẻ file cụ thể.

A) 💡 Suggested: Shared Docker volume — tất cả service mount chung một volume (vd. `/data/projects/{project_id}/`), mỗi service ghi output vào đường dẫn theo quy ước, service sau đọc trực tiếp từ đó
   - ✅ Strengths: đơn giản nhất để implement, không cần thêm cơ chế transfer file qua network, phù hợp vì tất cả container chạy trên cùng 1 máy
   - ⚠️ Trade-offs: các service phải thống nhất chặt chẽ về quy ước đường dẫn/tên file; nếu sau này tách sang nhiều máy sẽ phải đổi cơ chế

B) Truyền file qua HTTP (multipart upload/download) giữa các service — mỗi service expose endpoint để service khác tải file lên/xuống
   - ✅ Strengths: không phụ thuộc shared filesystem, dễ tách sang nhiều máy sau này
   - ⚠️ Trade-offs: phức tạp hơn, thêm overhead network cho file lớn (video clip), không cần thiết khi tất cả đang chạy chung 1 máy

C) Other (please describe after [Answer]: tag below)

[Answer]: A nhưng sau này có thể mở rộng dùng các service file khác như filenet minio hay s3 tạm thời cứ để shared docker volume

### Question 2: State Machine của "Video Project" tại Gateway
`high-level-design.md` (Open Items) xác định Gateway cần quản lý trạng thái tổng thể của video project. Cần xác nhận các trạng thái cụ thể.

A) 💡 Suggested: `draft → script_parsed → plugin_configured → rendering → tts_generating → assembling → ready_to_publish → publishing → published` (và `failed_at_<step>` cho mỗi bước có thể lỗi)
   - ✅ Strengths: phản ánh chính xác từng bước trong pipeline (khớp Epic A-F trong stories.md), dễ hiển thị tiến trình chi tiết trên GUI (Story C6)
   - ⚠️ Trade-offs: nhiều trạng thái hơn, cần đồng bộ chặt giữa Gateway và các service khi mỗi bước hoàn tất

B) Đơn giản hóa: chỉ `draft → processing → completed → published` và `failed` chung, không phân biệt lỗi ở bước nào
   - ✅ Strengths: đơn giản, ít trạng thái cần quản lý
   - ⚠️ Trade-offs: khó hiển thị chi tiết bước nào đang chạy/lỗi cho GUI (ảnh hưởng trực tiếp Story C6 — yêu cầu hiển thị "scene nào đang xử lý")

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Định dạng Plugin Interface (Content Plugin Service)
`architectural-style.md` xác định Content Plugin Service dùng Hexagonal với port `ContentPluginPort`. Cần xác định plugin được nạp theo cơ chế nào.

A) 💡 Suggested: Plugin nội bộ dạng Python class implement interface chung, đăng ký qua config/registry trong code (vd. `PLUGIN_REGISTRY = {"programming": ProgrammingPlugin()}`) — không load plugin động từ file ngoài ở giai đoạn này
   - ✅ Strengths: đơn giản, đủ dùng khi chỉ có 1 plugin (lập trình) được implement trong MVP, vẫn tuân thủ NFR1 vì thêm domain mới = thêm 1 class + đăng ký, không sửa core
   - ⚠️ Trade-offs: chưa hỗ trợ nạp plugin động từ bên ngoài (vd. file .py riêng do người dùng viết) mà không sửa code — nhưng đây là NGOÀI phạm vi MVP theo requirements.md

B) Plugin động — hệ thống quét thư mục `plugins/` và tự động nạp bất kỳ module Python nào implement đúng interface, không cần sửa code core khi thêm plugin
   - ✅ Strengths: mở rộng thực sự không cần sửa code, kể cả không phải nhánh chính
   - ⚠️ Trade-offs: phức tạp hơn nhiều (dynamic import, validation, sandbox an toàn khi tải code ngoài) — vượt quá nhu cầu hiện tại (chỉ chủ dự án tự thêm plugin, không phải bên thứ ba)

C) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 4: Component Method Interface Style
Method signature trong `component-methods.md` nên mô tả ở mức độ nào?

A) Mức API contract (request/response schema của từng REST endpoint + SSE event schema) — vì các component giao tiếp qua network (Microservices), không phải function call trực tiếp
   - ✅ Strengths: khớp đúng bản chất giao tiếp Microservices đã chọn (ADR-0001); là input trực tiếp cho Low-Level Design/Code Generation sau này
   - ⚠️ Trade-offs: không có, đây là cách tiếp cận chuẩn cho microservices

B) Mức pseudo function signature nội bộ Python (vd. `def render_scene(scene: Scene) -> AnimationClip`), bỏ qua chi tiết REST/HTTP
   - ✅ Strengths: gần gũi hơn với code thực tế bên trong mỗi service
   - ⚠️ Trade-offs: không phản ánh đúng ranh giới network thực sự giữa các service, dễ gây hiểu lầm là gọi hàm trực tiếp được

C) Other (please describe after [Answer]: tag below)

[Answer]: A
