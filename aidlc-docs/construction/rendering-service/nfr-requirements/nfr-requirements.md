# NFR Requirements — Unit 5: Rendering Service

## Performance
Manim render chạy trong `ThreadPoolExecutor`, timeout `RENDER_TIMEOUT_SECONDS` (mặc định 300s, đọc từ env var — Low-Level Design Question 6). `AnimationTemplateRegistry` build 1 lần lúc khởi động (dynamic discovery, ADR-0015), giữ trong memory suốt vòng đời process.

## Resource Constraints
Không giới hạn cứng CPU/RAM ở tầng ứng dụng — để Docker Compose's `deploy.resources` xử lý sau ở Infrastructure Design nếu cần. Xử lý TUẦN TỰ từng scene trong batch (không render song song nhiều scene cùng lúc trong 1 process) để tránh cạn kiệt tài nguyên máy dev cá nhân.

## Availability
Chấp nhận unavailability tạm thời — không multi-instance/failover (nhất quán toàn hệ thống). Message ở lại queue `rendering.commands` cho tới khi service khởi động lại.

## Security
Zero-trust validation (Functional Design Business Rule 1) đóng vai trò validation chính. Không auth/rate-limit riêng — chỉ Orchestrator gửi command qua RabbitMQ nội bộ.

## Messaging & Event Participation
Consumer `render_scenes` (queue `rendering.commands`), producer `scene_render_started`/`scene_rendered`/`rendering_completed`/`rendering_failed` (qua Outbox → `orchestrator.events`).

## Distributed Transaction Participation (Saga)
**Vai trò**: Participant trực tiếp, bước "Render Scenes" — sau bước "Synthesize Speech" (ADR-0014) trong Saga Render Pipeline. **Compensating action**: Không cần rollback — idempotent theo `project_id`+`scene_index` (Functional Design Rule 5); Orchestrator retry command `render_scenes` nếu cần, các scene đã render thành công được giữ nguyên.

## Caching Requirements
`AnimationTemplateRegistry` build 1 lần lúc khởi động — cache duy nhất. Không cache animation clip nào khác ngoài cơ chế idempotency-by-file đã có (Business Rule 5).

## Tech Stack Consistency
Python 3.12 (ADR-0009) + thư viện `manim` (Community Edition) — ràng buộc kỹ thuật cứng (Manim chỉ có Python binding chính thức). Không cần FastAPI (không có REST, nhất quán TTS/Script Processing sau retrofit).
