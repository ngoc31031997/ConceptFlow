# NFR Design Plan — Unit 6: Video Assembly Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-design-patterns.md`
- [ ] Tạo `logical-components.md`
- [ ] Tạo `messaging-design.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: CRUD vs CQRS (BẮT BUỘC)
A) 💡 Suggested: CRUD đơn giản trên `outbox_events`/`processed_messages` (bảng kỹ thuật, ADR-0013). Không phải CQRS — không có nhu cầu đọc/ghi lệch tải hay query phức tạp
   - ✅ Strengths: nhất quán Unit 2/3/4/5
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Resilience Pattern
A) 💡 Suggested: Không retry nội bộ. 1 lần ffmpeg lỗi/timeout (sau `ASSEMBLY_TIMEOUT_SECONDS`) → `AssemblyEngineError` → `assembly_failed` ngay, để Orchestrator quyết định retry (transient). Lỗi validation (zero-trust — `MissingArtifactError`/`InvalidSceneIndexError`/`InconsistentMediaFormatError`) cũng raise ngay, không retry nội bộ, vì input sai sẽ tiếp tục sai nếu retry ngay lập tức (cần Orchestrator/upstream sửa trước)
   - ✅ Strengths: nhất quán Unit 2/3/4/5
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Idempotency Pattern
A) 💡 Suggested: 2 tầng, mirror Unit 3/5 — message-level (Inbox, dedupe `message_id`) + artifact-level (file `final.mp4` tồn tại, Functional Design Rule 8). Không cần lock/race-condition handling (Orchestrator gọi tuần tự theo saga, tương tự lý do đã chấp nhận ở Unit 3/5)
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Saga Pattern / Event-Driven Design / Inbox-Outbox Pattern
A) 💡 Suggested: Xác nhận lại — Saga: participant trực tiếp, bước "Assemble Video" (sau "Render Scenes"). Event-Driven: consume `assemble_video`, publish 1 trong 2 event (`video_assembled` HOẶC `assembly_failed`) — ĐIỂM KHÁC BIỆT so với Unit 5: đúng 1 Outbox row/command (giống Unit 2/3/4, không phải nhiều row như Unit 5, vì không có progress per-scene, Low-Level Design Question 10). Inbox/Outbox: PostgreSQL-backed (`video-assembly-db`), schema giống hệt Unit 2/3/4/5 (ADR-0013)
   - ✅ Strengths: nhất quán pattern, đơn giản hơn Unit 5 (1 row/command)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 5: Security Pattern
A) 💡 Suggested: Zero-trust validation trong domain/application layer (Functional Design Rule 1, 2, 3) — validate toàn bộ input (file tồn tại, scene_index hợp lệ, media format đồng nhất) trước khi chạy ffmpeg. Không auth/rate-limit riêng — chỉ Orchestrator gửi command nội bộ qua RabbitMQ
   - ✅ Strengths: nhất quán, validation mạnh hơn cả Unit 5 (thêm media-format check)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
