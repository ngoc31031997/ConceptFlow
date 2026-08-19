# NFR Requirements Plan — Unit 5: Rendering Service

## Unit Context
Message-driven, CPU/memory-nặng (Manim rendering), Postgres Inbox/Outbox (ADR-0013), dynamic template plugin loading (ADR-0015).

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-requirements.md`
- [ ] Tạo `tech-stack-decisions.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Tech Stack Consistency (BẮT BUỘC)
A) 💡 Suggested: Python 3.12 (ADR-0009) + thư viện `manim` (Community Edition) — ràng buộc kỹ thuật cứng (Manim chỉ có Python binding chính thức). Không cần FastAPI (không có REST, nhất quán TTS/Script Processing sau retrofit)
   - ✅ Strengths: nhất quán, đúng ràng buộc kỹ thuật
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Performance — Đã xác định ở Low-Level Design
A) 💡 Suggested: Xác nhận lại: threadpool (`ThreadPoolExecutor`) cho Manim render, timeout `RENDER_TIMEOUT_SECONDS` (mặc định 300s, đọc từ env var — Question 6 của LLD). `AnimationTemplateRegistry` build 1 lần lúc khởi động (discovery), giữ trong memory suốt vòng đời process — tránh discover lại mỗi request
   - ✅ Strengths: nhất quán LLD, giảm overhead discovery lặp lại
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Resource Constraints — Memory/CPU
Manim rendering có thể tốn nhiều CPU/RAM (đặc biệt animation phức tạp, độ phân giải cao). Có cần giới hạn tài nguyên nào ở tầng NFR không?

A) 💡 Suggested: KHÔNG giới hạn cứng ở tầng ứng dụng (không set memory limit trong code) — để Docker Compose's `deploy.resources` (nếu cần) xử lý ở tầng Infrastructure Design sau. Ở mức NFR chỉ ghi nhận: xử lý TUẦN TỰ từng scene trong batch (không render song song nhiều scene cùng lúc trong 1 process) để tránh cạn kiệt tài nguyên máy dev cá nhân
   - ✅ Strengths: đơn giản, phù hợp máy dev cá nhân (không cần tuning phức tạp ở MVP)
   - ⚉️ Trade-offs: render tuần tự chậm hơn nếu có nhiều scene — chấp nhận được, ưu tiên ổn định hơn tốc độ ở giai đoạn MVP

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Availability
A) 💡 Suggested: Chấp nhận unavailability tạm thời — không multi-instance (nhất quán toàn hệ thống). Message ở lại queue `rendering.commands` cho tới khi service khởi động lại
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Security
A) 💡 Suggested: Zero-trust validation đã có ở Functional Design (Business Rule 1) đóng vai trò validation chính. Không auth/rate-limit riêng — chỉ Orchestrator gửi command qua RabbitMQ nội bộ
   - ✅ Strengths: nhất quán, đã có validation mạnh hơn các unit khác (zero-trust)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: Messaging & Event Participation / Saga Participation
A) 💡 Suggested: Consumer `render_scenes` (queue `rendering.commands`), producer `scene_render_started`/`scene_rendered`/`rendering_completed`/`rendering_failed` (qua Outbox). Saga role: Participant trực tiếp, bước "Render Scenes" (sau "Synthesize Speech", ADR-0014). Compensating action: không cần rollback — idempotent theo `project_id`+`scene_index` (Functional Design Rule 5), Orchestrator retry command nếu cần
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 7: Caching Requirements
A) 💡 Suggested: `AnimationTemplateRegistry` build 1 lần lúc khởi động (đã nêu ở Question 2) — đây là cache duy nhất. Không cần cache animation clip nào khác ngoài cơ chế idempotency-by-file đã có
   - ✅ Strengths: đơn giản, đủ dùng
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
