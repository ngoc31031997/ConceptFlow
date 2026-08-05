# NFR Design Patterns — Unit 2: Content Plugin Service

## CRUD vs CQRS
**N/A** — Không có data model nghiệp vụ cần lưu trữ; unit hoàn toàn stateless (registry in-memory nạp lúc khởi động).

## Idempotency Pattern
In-memory `set[message_id]`, TTL 24h (khớp message TTL của Unit 1), tự dọn định kỳ (background task đơn giản, không cần scheduler ngoài). Chấp nhận mất dedupe state khi restart vì bước "Classify Scenes" không có side-effect bền vững — hệ quả tối đa của việc mất dedupe là publish event trùng, được Orchestrator Service tự xử lý idempotent ở tầng của nó.

## Resilience Pattern
Dùng cơ chế reconnect tự động có sẵn của `aio-pika` (built-in connection recovery) cho kết nối RabbitMQ — không tự viết circuit breaker riêng vì chỉ có 1 dependency hạ tầng duy nhất.

## Saga Pattern
Participant (không phải coordinator) cho bước "Classify Scenes". Không có compensating action (theo NFR Requirements — pure computation, không side-effect bền vững).

## Security Pattern
Input validation qua Pydantic schema (FastAPI). Không auth/rate-limit riêng (chỉ gọi nội bộ từ Gateway/Orchestrator).
