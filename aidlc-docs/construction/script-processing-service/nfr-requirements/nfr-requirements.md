# NFR Requirements — Unit 4: Script Processing Service

## Performance
Parsing Markdown chạy trực tiếp trong consumer's async handler — không cần threadpool (CPU-bound nhẹ, hoàn tất trong mili-giây, khác biệt căn bản với TTS Service's Piper synthesis tốn giây).

## Availability
Chấp nhận unavailability tạm thời — không multi-instance/failover (nhất quán toàn hệ thống). Message ở lại queue `script_processing.commands` (RabbitMQ durability) cho tới khi service khởi động lại.

## Security
Validate input trong domain layer (`MarkdownScriptParser`). Không auth/rate-limit riêng — chỉ Orchestrator gửi command qua RabbitMQ nội bộ.

## Messaging & Event Participation
Consumer `parse_script` (queue `script_processing.commands`), producer `script_parsed`/`parse_failed` (qua Outbox → `orchestrator.events`).

## Distributed Transaction Participation (Saga)
**Vai trò**: Participant trực tiếp cho bước "Parse Script" — bước đầu tiên của Saga Render Pipeline (`services.md`). **Compensating action**: Không cần rollback — stateless, không side-effect bền vững; Orchestrator retry command nếu cần.

## Caching Requirements
N/A — không có dữ liệu nào cần cache (parser không phụ thuộc tài nguyên nạp lúc khởi động, khác với Piper voice model của TTS Service).

## Tech Stack Consistency
Python 3.12 (ADR-0009). Không cần FastAPI (không có REST endpoint) — chỉ `aio-pika` (AMQP) + `asyncpg` (Postgres, ADR-0013), nhất quán với Unit 2/Unit 3 (sau retrofit).
