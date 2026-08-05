# NFR Design Plan — Unit 3: TTS Service

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-design-patterns.md`
- [ ] Tạo `logical-components.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: CRUD vs CQRS (BẮT BUỘC)
Unit 3 có data model nghiệp vụ nào cần lưu trữ có cấu trúc CRUD/CQRS không?

A) 💡 Suggested: **N/A** — không có database, không có data model nghiệp vụ cần persist ngoài file audio (blob, không phải structured data cần query). Unit hoàn toàn stateless
   - ✅ Strengths: đúng bản chất unit, tránh over-engineer
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Resilience Pattern
NFR Requirements đã xác định timeout 60s cho synthesis. Có cần retry nội bộ trong TTS Service khi Piper lỗi tạm thời, hay để Rendering Service tự retry toàn bộ request?

A) 💡 Suggested: KHÔNG retry nội bộ trong TTS Service — 1 lần gọi Piper thất bại (crash/timeout) → trả lỗi ngay (`TTSEngineError`/502), để Rendering Service (qua Saga compensating action) quyết định có retry hay không. Tránh 2 tầng retry chồng lên nhau (retry nội bộ + retry của Saga) gây khó đoán tổng thời gian chờ
   - ✅ Strengths: đơn giản, tránh double-retry, nhất quán với nguyên tắc "TTS Service không tự quyết định chiến lược retry của Saga" (NFR Requirements Q7)
   - ⚠️ Trade-offs: nếu Piper có lỗi thoáng qua rất ngắn (transient), phải chờ round-trip đầy đủ tới Rendering Service mới retry được — chấp nhận được vì lỗi Piper hiếm khi tự phục hồi ngay trong mili-giây

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Caching Strategy — Voice Model In-Memory
NFR Requirements đã xác định load Piper voice model 1 lần lúc khởi động. Cần xác nhận chi tiết placement/key design.

A) 💡 Suggested: **In-process cache** (không phải Redis/distributed) — biến module-level trong `piper_adapter.py`, key là `language` (`"vi"`/`"en"`), value là Piper voice model instance đã load. Không có TTL/invalidation (model không đổi trong vòng đời process; nếu cần đổi model, restart service). Không cần cache-aside/write-through pattern phức tạp vì chỉ load 1 lần tại startup, không load lazy theo request
   - ✅ Strengths: đơn giản nhất, đúng nhu cầu thực tế (chỉ 2 model cố định, không đổi khi chạy)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Idempotency Detail
Functional Design đã xác định idempotency qua kiểm tra file tồn tại tại đường dẫn shared volume. Có cần thêm cơ chế lock/race-condition handling khi 2 request cùng `project_id`+`scene_index` đến gần như đồng thời không?

A) 💡 Suggested: KHÔNG cần lock ở MVP — theo thiết kế Saga (`services.md`), Rendering Service gọi tuần tự từng scene (không có 2 request trùng `project_id`+`scene_index` đồng thời trong luồng bình thường). Nếu race condition hiếm gặp xảy ra (vd. do lỗi ở tầng gọi), rủi ro tối đa là ghi đè file audio giống hệt nội dung (cùng input → cùng output, vô hại) — chấp nhận được, không cần cơ chế lock phức tạp cho MVP
   - ✅ Strengths: đơn giản, đúng mức độ rủi ro thực tế thấp
   - ⚠️ Trade-offs: nếu sau này có nhiều Rendering Service instance gọi song song thật, cần xem xét lại — chưa cần ở MVP (Rendering Service cũng chỉ 1 instance theo thiết kế hệ thống)

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Security Pattern
Xác nhận lại pattern bảo mật cụ thể.

A) 💡 Suggested: Input validation qua Pydantic schema (FastAPI) cho `project_id`, `scene_index`, `text`, `language`. Không auth/rate-limit riêng (chỉ Rendering Service gọi nội bộ, cùng Docker network, Security Baseline extension tắt)
   - ✅ Strengths: nhất quán với Unit 2, đúng mức cần thiết
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: Event-Driven Design / Saga Pattern / Inbox-Outbox
Đã xác nhận ở NFR Requirements: Unit 3 không tham gia RabbitMQ, không publish/consume event, vai trò Saga là participant gián tiếp không cần compensating action.

A) 💡 Suggested: **N/A** cho Event-Driven Design và Inbox/Outbox Pattern (không publish event, không có database transaction cần đồng bộ với event). Saga Pattern: xác nhận lại participant gián tiếp, không compensating action (đã quyết định ở NFR Requirements) — không cần thiết kế thêm ở NFR Design
   - ✅ Strengths: nhất quán, không cần thêm thiết kế
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
