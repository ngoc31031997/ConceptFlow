# NFR Requirements Plan — Unit 4: Script Processing Service

## Unit Context
Stateless AMQP consumer/producer, Postgres Inbox/Outbox đã xác định từ Low-Level Design (ADR-0013). Không có endpoint REST nào.

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-requirements.md`
- [ ] Tạo `tech-stack-decisions.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Tech Stack Consistency (BẮT BUỘC)
A) 💡 Suggested: Python 3.12 (ADR-0009), KHÔNG cần FastAPI (không có REST endpoint nào, giống Unit 3 sau retrofit) — chỉ `aio-pika` (AMQP) + `asyncpg` (Postgres), nhất quán với Unit 2/Unit 3
   - ✅ Strengths: nhất quán, không cài đặt thư viện thừa
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Performance
Parsing Markdown là CPU-bound nhẹ (regex/string processing, không I/O ngoài trừ DB/AMQP). Có cần threadpool như TTS Service không?

A) 💡 Suggested: KHÔNG cần threadpool — parsing script (vài KB text) hoàn tất trong mili-giây, không đáng kể so với I/O của DB/AMQP; khác biệt căn bản với TTS (Piper synthesis tốn giây, không phải mili-giây). Chạy trực tiếp trong consumer's async handler
   - ✅ Strengths: đơn giản, không over-engineer cho workload nhẹ
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Availability
A) 💡 Suggested: Chấp nhận unavailability tạm thời — không multi-instance (nhất quán toàn hệ thống). Message ở lại queue `script_processing.commands` (RabbitMQ durability) cho tới khi service khởi động lại
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Security
A) 💡 Suggested: Validate input trong domain layer (`MarkdownScriptParser`, không phải framework validation vì không có REST). Không auth/rate-limit riêng — chỉ Orchestrator gửi command qua RabbitMQ nội bộ
   - ✅ Strengths: nhất quán Unit 2/Unit 3
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Messaging & Event Participation / Saga Participation
A) 💡 Suggested: Consumer `parse_script` (queue `script_processing.commands`), producer `script_parsed`/`parse_failed` (qua Outbox → `orchestrator.events`). Saga role: Participant trực tiếp cho bước "Parse Script" (bước đầu tiên của Saga Render Pipeline). Compensating action: không cần rollback (stateless, không side-effect bền vững) — Orchestrator retry command nếu cần
   - ✅ Strengths: nhất quán với Unit 2/Unit 3 (sau retrofit)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Caching Requirements
A) 💡 Suggested: N/A — không có gì để cache (parser không phụ thuộc dữ liệu ngoài nạp lúc khởi động, khác với Piper voice model của TTS Service)
   - ✅ Strengths: đúng bản chất unit
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
