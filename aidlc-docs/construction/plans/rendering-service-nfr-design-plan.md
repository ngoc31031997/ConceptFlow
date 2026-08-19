# NFR Design Plan — Unit 5: Rendering Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-design-patterns.md`
- [ ] Tạo `logical-components.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: CRUD vs CQRS (BẮT BUỘC)
A) 💡 Suggested: CRUD đơn giản trên `outbox_events`/`processed_messages` (bảng kỹ thuật, ADR-0013). Không phải CQRS
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Resilience Pattern
A) 💡 Suggested: Không retry nội bộ. 1 lần Manim lỗi/timeout (sau `RENDER_TIMEOUT_SECONDS`) → `AnimationEngineError` → `rendering_failed` ngay, để Orchestrator quyết định retry (transient). Không có lỗi "permanent" đặc thù riêng ở Rendering ngoài validation input (zero-trust, đã raise ngay không cần retry)
   - ✅ Strengths: nhất quán Unit 2/3/4
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Idempotency Pattern
A) 💡 Suggested: 2 tầng, mirror TTS Service — message-level (Inbox, dedupe `message_id`) + artifact-level (file `.mp4` tồn tại, Functional Design Rule 5). Không cần lock/race-condition handling (Orchestrator gọi tuần tự, tương tự lý do đã chấp nhận ở TTS Service)
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Saga Pattern / Event-Driven Design / Inbox-Outbox Pattern
A) 💡 Suggested: Xác nhận lại — Saga: participant trực tiếp, bước "Render Scenes". Event-Driven: consume `render_scenes`, publish 4 loại event (`scene_render_started`, `scene_rendered`, `rendering_completed`, `rendering_failed`) — ĐIỂM KHÁC BIỆT so với Unit 2/3/4: nhiều Outbox row/command thay vì 1 (Low-Level Design Question 9). Inbox/Outbox: PostgreSQL-backed (`rendering-db`), giống Unit 2/3/4, chỉ khác số lượng row mỗi lần xử lý command
   - ✅ Strengths: nhất quán pattern, chỉ khác ở tần suất publish
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Security Pattern
A) 💡 Suggested: Zero-trust validation trong domain layer (`RenderSceneUseCase`, Functional Design Business Rule 1) — mạnh hơn các unit khác (validate toàn bộ input, không chỉ phần cần cho logic). Không auth/rate-limit riêng — chỉ Orchestrator gửi command nội bộ
   - ✅ Strengths: nhất quán, đã có validation mạnh
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
