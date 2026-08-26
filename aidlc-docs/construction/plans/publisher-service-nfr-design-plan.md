# NFR Design Plan — Unit 7: Publisher Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-design-patterns.md`
- [ ] Tạo `logical-components.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: CRUD vs CQRS (BẮT BUỘC)
A) 💡 Suggested: CRUD đơn giản trên `outbox_events`/`processed_messages` (ADR-0013) + `oauth_credentials` (1 row, upsert). Không phải CQRS — không có nhu cầu đọc/ghi lệch tải hay query phức tạp
   - ✅ Strengths: nhất quán Unit 2/3/4/5/6
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 2: Resilience Pattern
A) 💡 Suggested: Không retry nội bộ cho `publish_video`. Lỗi upload/timeout (sau `UPLOAD_TIMEOUT_SECONDS`) → `UploadError` → `publish_failed` ngay, để Orchestrator/GUI quyết định retry (transient, Functional Design Rule 8). Lỗi validation (zero-trust — `InvalidPublishRequestError`) và `MissingCredentialError` cũng raise ngay, không retry nội bộ. Với OAuth callback (REST, ngoài Saga): không có concept "retry nội bộ" — Creator tự bấm lại `/v1/auth/youtube/start` nếu callback lỗi (Functional Design Rule 6)
   - ✅ Strengths: nhất quán Unit 2/3/4/5/6
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 3: Idempotency Pattern
A) 💡 Suggested: CHỈ message-level (Inbox, dedupe `message_id`) — KHÔNG có artifact-level idempotency (khác Unit 3/5/6) vì mỗi `publish_video` thành công luôn tạo video MỚI trên YouTube (Low-Level Design Question 7, Functional Design Rule 7). OAuth credential (`oauth_credentials`) tự nhiên idempotent theo thiết kế — callback nhiều lần chỉ upsert cùng 1 row
   - ✅ Strengths: đúng bản chất domain (không có "artifact" nào để idempotency-by-file như video/audio clip)
   - ⚠️ Trade-offs: retry `publish_video` sau lỗi giữa chừng có thể tạo video trùng trên YouTube (đã ghi nhận, chấp nhận được ở MVP — Low-Level Design Question 7)

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 4: Saga Pattern / Event-Driven Design / Inbox-Outbox Pattern
A) 💡 Suggested: Xác nhận lại — Saga: participant trực tiếp, bước cuối "Publish Video" (sau "Assemble Video", không có bước nào sau nó trong Saga Render+Publish). Event-Driven: consume `publish_video`, publish đúng 1 trong 2 event (`video_published`/`publish_failed`) — 1 Outbox row/command (mirror Unit 6). Inbox/Outbox: PostgreSQL-backed (`publisher-db`), schema Outbox/Inbox giống hệt Unit 2/3/4/5/6 (ADR-0013), CỘNG bảng `oauth_credentials` riêng (không thuộc Inbox/Outbox pattern — business data, không phải kỹ thuật)
   - ✅ Strengths: nhất quán pattern, đơn giản hơn Unit 5 (1 row/command)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 5: Security Pattern
A) 💡 Suggested: Zero-trust validation (Functional Design Rule 1) cho `publish_video` payload trước khi gọi YouTube API. OAuth flow: `state` parameter chuẩn chống CSRF (Low-Level Design's interface-contracts.md). Credential lưu plaintext (ADR-0016). Không auth/rate-limit riêng cho AMQP hay REST OAuth endpoint (threat model single-user local, nhất quán NFR Requirements Question 5)
   - ✅ Strengths: nhất quán, đã xác lập ở NFR Requirements
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a
