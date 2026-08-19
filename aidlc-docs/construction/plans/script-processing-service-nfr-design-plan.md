# NFR Design Plan — Unit 4: Script Processing Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-design-patterns.md`
- [ ] Tạo `logical-components.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: CRUD vs CQRS (BẮT BUỘC)
A) 💡 Suggested: CRUD đơn giản trên `outbox_events`/`processed_messages` (bảng kỹ thuật, không phải business data — nhất quán Unit 2/Unit 3 sau retrofit, ADR-0013). Không phải CQRS
   - ✅ Strengths: nhất quán, đúng quy mô
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Resilience Pattern
A) 💡 Suggested: Không retry nội bộ — 1 lần parse lỗi (cú pháp) → publish `parse_failed` ngay, Creator sửa script và Orchestrator gửi lại `parse_script` mới (không phải retry tự động, vì lỗi cú pháp không tự phục hồi — khác với `synthesis_failed` của TTS Service vốn có thể transient)
   - ✅ Strengths: đúng bản chất lỗi (permanent, cần con người sửa), không cần cơ chế retry
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Idempotency Pattern
A) 💡 Suggested: Chỉ 1 tầng — message-level qua Inbox (`processed_messages`, ADR-0013). KHÔNG có artifact-level như TTS Service (không có file nào để kiểm tra tồn tại — parse là pure computation, không side-effect bền vững ngoài việc publish event)
   - ✅ Strengths: đơn giản, đúng bản chất unit (stateless, không tạo artifact)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Saga Pattern / Event-Driven Design / Inbox-Outbox Pattern
A) 💡 Suggested: Xác nhận lại — Saga: participant trực tiếp, bước đầu tiên "Parse Script", không compensating action. Event-Driven: consume `parse_script`, publish `script_parsed`/`parse_failed` (integration event, envelope chuẩn Unit 1). Inbox/Outbox: PostgreSQL-backed (`script-processing-db`), giống hệt Unit 2/Unit 3 — Outbox ghi trong transaction với Inbox mark, `OutboxRelay` publish bất đồng bộ
   - ✅ Strengths: nhất quán toàn hệ thống, không cần thiết kế mới
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Security Pattern
A) 💡 Suggested: Validate input trong `MarkdownScriptParser` (domain layer). Không auth/rate-limit riêng — chỉ Orchestrator gửi command nội bộ
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
