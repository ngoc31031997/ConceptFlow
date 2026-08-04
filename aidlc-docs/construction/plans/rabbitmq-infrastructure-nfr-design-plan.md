# NFR Design Plan — Unit 1: RabbitMQ Infrastructure

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-design-patterns.md`
- [ ] Tạo `logical-components.md`
- [ ] Tạo `messaging-design.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: CRUD vs CQRS (BẮT BUỘC hỏi)
Unit này không sở hữu dữ liệu nghiệp vụ (chỉ là message broker trung chuyển) — không có model đọc/ghi nào để áp dụng CRUD hay CQRS. Bạn xác nhận CQRS không áp dụng cho unit này?

A) Xác nhận — N/A, unit thuần túy hạ tầng messaging, không có data model nghiệp vụ

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Exchange/Queue Topology
Theo `component-methods.md`, có 5 queue command (`script_processing.commands`, `content_plugin.commands`, `rendering.commands`, `video_assembly.commands`, `publisher.commands`) và 1 queue event chung (`orchestrator.events`). Loại exchange nào phù hợp?

A) 💡 Suggested: Direct Exchange cho command (routing key = tên service, 1-1 tới đúng queue), Direct Exchange riêng cho event (routing key = loại event, tất cả về `orchestrator.events` vì chỉ Orchestrator là consumer duy nhất của event)
   - ✅ Strengths: đơn giản, đúng nhu cầu hiện tại (1 producer → 1 consumer xác định cho mỗi command; nhiều producer → 1 consumer cho event)
   - ⚠️ Trade-offs: nếu sau này cần nhiều consumer cùng lắng nghe 1 loại event (fan-out), cần đổi sang Fanout/Topic Exchange

B) Topic Exchange cho cả command và event (routing key dạng `service.action`, hỗ trợ pattern matching linh hoạt hơn)
   - ✅ Strengths: linh hoạt hơn cho mở rộng sau này (vd. thêm domain giáo dục mới có thể cần routing phức tạp hơn)
   - ⚠️ Trade-offs: phức tạp hơn mức cần thiết cho nhu cầu hiện tại (routing 1-1 đơn giản)

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Saga Role của unit này
Xác nhận: Unit 1 (RabbitMQ) đóng vai trò "transport" cho Saga, KHÔNG phải Saga coordinator (đó là Orchestrator Service, Unit 8) — đúng không?

A) Đúng — RabbitMQ chỉ là hạ tầng vận chuyển message, không chứa logic Saga nào

B) Other (please describe after [Answer]: tag below)

[Answer]: A

