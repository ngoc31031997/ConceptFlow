# NFR Requirements — Unit 6: Video Assembly Service

## Performance
Toàn bộ chuỗi ffprobe pre-check + ffmpeg (mux → concat → overlay) chạy trong `ThreadPoolExecutor`, timeout `ASSEMBLY_TIMEOUT_SECONDS` (mặc định 180s, đọc từ env var — Low-Level Design Question 6). Vì hầu hết bước dùng `-c copy` (không re-encode video), thời gian xử lý chủ yếu phụ thuộc IO đọc/ghi shared volume, nhẹ hơn đáng kể so với Manim render.

## Resource Constraints — Disk I/O
Không giới hạn cứng disk quota ở tầng ứng dụng — để Docker Compose xử lý sau ở Infrastructure Design nếu cần. Xử lý TUẦN TỰ (RabbitMQ consumer prefetch=1, không assembly song song nhiều project cùng lúc trong 1 process, nhất quán Unit 2/3/4/5). File trung gian (`_tmp/`) LUÔN được dọn sau khi assembly thành công (Low-Level Design Question 7) để tránh tích luỹ dung lượng.

## Availability
Chấp nhận unavailability tạm thời — không multi-instance/failover (nhất quán toàn hệ thống). Message ở lại queue `video_assembly.commands` cho tới khi service khởi động lại.

## Security
Zero-trust validation (Functional Design Rule 1) + media-format consistency check (Rule 3) đóng vai trò validation chính. Không auth/rate-limit riêng — chỉ Orchestrator gửi command qua RabbitMQ nội bộ.

## Messaging & Event Participation
Consumer `assemble_video` (queue `video_assembly.commands`), producer `video_assembled`/`assembly_failed` (qua Outbox → `orchestrator.events`, 1 event/command — Low-Level Design Question 10). Delivery guarantee: at-least-once, dedupe qua Inbox theo `message_id` (ADR-0013, nhất quán Unit 2/3/4/5).

## Distributed Transaction Participation (Saga)
**Vai trò**: Participant trực tiếp, bước "Assemble Video" — sau bước "Render Scenes" trong Saga Render Pipeline. **Compensating action**: Không cần rollback — idempotent theo `project_id` (file `final.mp4` đã tồn tại → trả kết quả ngay, Functional Design Rule 8); Orchestrator retry command `assemble_video` nếu cần; animation/audio clip input KHÔNG bị xoá dù assembly lỗi (Functional Design Rule 9).

## Caching Requirements
Không cần cache nào ở tầng ứng dụng — không có registry/discovery (khác Unit 2/5, chỉ 1 chiến lược assembly cố định). Idempotency-by-file (`final.mp4` đã tồn tại) đóng vai trò tương đương "cache kết quả" cho retry.

## Tech Stack Consistency
Python 3.12 (ADR-0009) + ffmpeg/ffprobe (external binary, gọi qua `subprocess`) + PostgreSQL Inbox/Outbox (ADR-0013, asyncpg) + RabbitMQ (aio-pika) — không cần FastAPI (không có REST, nhất quán TTS/Script Processing/Rendering sau retrofit). Xem `tech-stack-decisions.md` cho chi tiết đầy đủ.
