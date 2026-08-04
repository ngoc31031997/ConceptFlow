# Unit of Work Plan

## Approach
Application Design đã xác định 10 microservice độc lập. Theo định nghĩa "unit of work" (mỗi microservice = 1 unit độc lập triển khai), đề xuất ánh xạ **1 service = 1 unit**, trừ khi có lý do gộp một số unit nhỏ lại để giảm overhead quản lý ở giai đoạn 1 người dùng/dự án cá nhân.

## Execution Checklist
- [ ] Thu thập câu trả lời cho câu hỏi bên dưới
- [ ] Phân tích câu trả lời, đặt câu hỏi follow-up nếu có mơ hồ/mâu thuẫn
- [ ] Tạo `unit-of-work.md`
- [ ] Tạo `unit-of-work-dependency.md`
- [ ] Tạo `unit-of-work-story-map.md`
- [ ] Xác nhận mọi story trong `stories.md` đều được gán vào 1 unit
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Story Grouping — Mức độ chia nhỏ unit
Đề xuất ánh xạ 1 microservice = 1 unit of work (10 unit: Web GUI, API Gateway, Orchestrator Service, RabbitMQ setup, Content Plugin Service, Script Processing Service, Rendering Service, TTS Service, Video Assembly Service, Publisher Service). Bạn có đồng ý cách chia này không?

A) Đồng ý — giữ nguyên 1 service = 1 unit (10 unit), khớp trực tiếp với ranh giới Microservices đã thiết kế
   - ✅ Strengths: đơn giản, nhất quán với kiến trúc đã duyệt, mỗi unit có trách nhiệm rõ ràng để thiết kế/code độc lập
   - ⚠️ Trade-offs: 10 unit là khá nhiều cho 1 người phát triển — có thể tốn thời gian lặp lại quy trình Low-Level Design/Functional Design/Code Generation cho từng unit

B) Gộp một số unit nhỏ/hạ tầng lại để giảm số lượng unit cần đi qua toàn bộ quy trình Construction riêng biệt (vd. gộp "RabbitMQ setup" vào Infrastructure Design chung thay vì là 1 unit riêng; gộp Web GUI + API Gateway thành 1 unit "Frontend Layer" vì đều là lớp giao tiếp với người dùng)
   - ✅ Strengths: giảm số vòng lặp Construction, phù hợp hơn cho 1 người phát triển với dự án cá nhân
   - ⚠️ Trade-offs: unit gộp có thể có trách nhiệm hỗn hợp, khó theo dõi tiến độ riêng từng phần

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Thứ tự phát triển (Development Sequence)
Các unit có dependency lẫn nhau (RabbitMQ là hạ tầng nền cho Orchestrator + 5 service nghiệp vụ; Rendering phụ thuộc TTS). Bạn muốn phát triển theo thứ tự nào?

A) 💡 Suggested: Theo dependency từ dưới lên — hạ tầng trước (RabbitMQ, Shared Volume) → service nghiệp vụ không phụ thuộc unit khác (Content Plugin, TTS) → service phụ thuộc (Script Processing, Rendering, Video Assembly, Publisher) → Orchestrator Service (cần tất cả service nghiệp vụ sẵn sàng để test luồng Saga) → API Gateway → Web GUI (cần backend hoàn chỉnh để test end-to-end)
   - ✅ Strengths: mỗi unit có thể test độc lập ngay khi hoàn thành (unit test + có thể gọi thử qua RabbitMQ/REST) trước khi unit phụ thuộc nó được code
   - ⚠️ Trade-offs: chưa thấy được trải nghiệm GUI đến tận cuối quy trình

B) Theo trải nghiệm người dùng — GUI trước (dùng mock data) → Gateway → Orchestrator → các service nghiệp vụ
   - ✅ Strengths: sớm có cái nhìn trực quan về sản phẩm cuối
   - ⚠️ Trade-offs: cần xây mock/stub cho toàn bộ backend trước khi có backend thật, tốn công sức phụ

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Code Organization (Greenfield, multi-unit)
Với 10 (hoặc số lượng theo Câu 1) unit độc lập, cấu trúc thư mục dự án nên tổ chức theo cách nào?

A) 💡 Suggested: Monorepo — 1 git repository chứa tất cả unit, mỗi unit là 1 thư mục con cấp cao (vd. `services/orchestrator/`, `services/rendering/`, `frontend/`, `docker-compose.yml` ở root)
   - ✅ Strengths: dễ quản lý cho 1 người phát triển, 1 lệnh `docker-compose up` build toàn bộ, dễ đồng bộ thay đổi cross-service (vd. sửa message schema ảnh hưởng 2 service cùng lúc)
   - ⚠️ Trade-offs: repo lớn dần theo thời gian, không tách quyền truy cập riêng từng service (không cần thiết cho 1 người dùng)

B) Polyrepo — mỗi unit là 1 git repository riêng biệt
   - ✅ Strengths: cô lập hoàn toàn, phù hợp khi có nhiều team độc lập
   - ⚠️ Trade-offs: phức tạp hóa việc quản lý cho 1 người phát triển dự án cá nhân, khó đồng bộ thay đổi cross-service, cần thêm cơ chế quản lý version giữa các repo

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Business Domain Alignment
Xác nhận: mỗi unit hiện tại được nhóm theo ranh giới microservice kỹ thuật (đã quyết định ở Application Design), không theo bounded context nghiệp vụ riêng (vì DDD bị đánh giá là over-engineering ở ADR-0002). Bạn xác nhận cách tiếp cận này là đúng, không cần nhóm lại theo business domain?

A) Đúng — giữ nguyên nhóm theo ranh giới microservice kỹ thuật đã có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
