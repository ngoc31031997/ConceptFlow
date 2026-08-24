# NFR Requirements Plan — Unit 6: Video Assembly Service

## Unit Context
Message-driven, IO/CPU-nặng vừa phải (ffmpeg mux/concat/overlay, chủ yếu `-c copy` nên nhẹ hơn Manim render), Postgres Inbox/Outbox (ADR-0013).

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-requirements.md`
- [ ] Tạo `tech-stack-decisions.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Tech Stack Consistency (BẮT BUỘC)
A) 💡 Suggested: Python 3.12 (ADR-0009) + ffmpeg (external binary, gọi qua `subprocess`, LLD Question 3) + `ffprobe` (cùng bộ với ffmpeg, cho media-format pre-check, Functional Design Question 3). Không cần FastAPI (không có REST, nhất quán TTS/Script Processing/Rendering sau retrofit)
   - ✅ Strengths: nhất quán hệ thống, ffmpeg/ffprobe là công cụ chuẩn đã quyết định ở `technology-direction.md`
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A nhưng cũng cần inbox outbox pattern

### Question 2: Performance — Đã xác định ở Low-Level Design
A) 💡 Suggested: Xác nhận lại: `ThreadPoolExecutor` cho toàn bộ chuỗi ffprobe+ffmpeg (LLD Question 6), timeout `ASSEMBLY_TIMEOUT_SECONDS` (mặc định 180s, đọc từ env var). Vì hầu hết bước dùng `-c copy` (không re-encode video), thời gian xử lý chủ yếu phụ thuộc IO đọc/ghi shared volume, không CPU-bound nặng như Manim render — timeout 180s là dư dả cho video giáo dục thông thường (vài phút)
   - ✅ Strengths: nhất quán LLD, thời gian assembly nhanh hơn đáng kể so với rendering
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Resource Constraints — Disk I/O
Video Assembly Service đọc nhiều file input (animation + audio clip mỗi scene) và ghi file trung gian (`_tmp/`) + output cuối cùng lên shared volume. Có cần giới hạn/quản lý gì ở tầng NFR không?

A) 💡 Suggested: KHÔNG giới hạn cứng ở tầng ứng dụng (không set disk quota trong code) — để Docker Compose xử lý ở tầng Infrastructure Design sau nếu cần. Ở mức NFR chỉ ghi nhận: xử lý TUẦN TỰ (không assembly song song nhiều project cùng lúc trong 1 process — do RabbitMQ consumer prefetch=1, nhất quán Unit 2/3/4/5), và LUÔN dọn `_tmp/` sau khi thành công (đã có ở LLD Question 7) để tránh tích luỹ file rác chiếm dung lượng
   - ✅ Strengths: đơn giản, phù hợp máy dev cá nhân, tránh rò rỉ dung lượng shared volume theo thời gian
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Availability
A) 💡 Suggested: Chấp nhận unavailability tạm thời — không multi-instance (nhất quán toàn hệ thống). Message ở lại queue `video_assembly.commands` cho tới khi service khởi động lại
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Security
A) 💡 Suggested: Zero-trust validation đã có ở Functional Design (Business Rule 1) đóng vai trò validation chính. Không auth/rate-limit riêng — chỉ Orchestrator gửi command qua RabbitMQ nội bộ
   - ✅ Strengths: nhất quán, đã có validation mạnh (zero-trust + media-format consistency check)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: Messaging & Event Participation / Saga Participation
A) 💡 Suggested: Consumer `assemble_video` (queue `video_assembly.commands`), producer `video_assembled`/`assembly_failed` (qua Outbox, 1 event/command). Saga role: Participant trực tiếp, bước "Assemble Video" (sau "Render Scenes"). Compensating action: không cần rollback — idempotent theo `project_id` (file `final.mp4` đã tồn tại → trả kết quả ngay, Functional Design Rule 8), Orchestrator retry command nếu cần, animation/audio clip input giữ nguyên
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 7: Caching Requirements
A) 💡 Suggested: Không cần cache nào ở tầng ứng dụng — không có registry/discovery như Unit 2/5 (chỉ 1 chiến lược assembly cố định). Idempotency-by-file (`final.mp4` đã tồn tại) đóng vai trò tương đương "cache kết quả" cho retry
   - ✅ Strengths: đơn giản, đúng bản chất (không có gì để cache ngoài chính output)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
