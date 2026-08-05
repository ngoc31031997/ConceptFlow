# NFR Requirements — Unit 2: Content Plugin Service

## Performance
Không có SLA cứng (theo NFR2 hệ thống). Xử lý là in-memory lookup/validation thuần túy (không I/O ngoài trừ AMQP/REST của chính request) — response time thực tế ở mức mili-giây, không cần tối ưu đặc biệt.

## Availability
Chấp nhận unavailability tạm thời — không cần multi-instance/failover. Nếu service down, message `classify_scenes` được RabbitMQ requeue (at-least-once, retry 3 lần, theo `nfr-requirements.md` Unit 1) và tự xử lý khi service khởi động lại.

## Security
Validate input qua Pydantic schema (FastAPI mặc định) để tránh lỗi runtime từ dữ liệu sai định dạng. Không cần auth/rate-limit riêng vì chỉ Gateway/Orchestrator (nội bộ) gọi tới, không expose public (Security Baseline extension đã tắt).

## Messaging & Event Participation
Consumer của `classify_scenes` (queue `content_plugin.commands`), producer của `scenes_classified`/`classification_failed` (→ `orchestrator.events`). Kế thừa delivery guarantee at-least-once từ Unit 1; idempotency xử lý ở tầng consumer (Rule 5, Functional Design).

## Distributed Transaction Participation (Saga)
**Vai trò**: Participant (không phải coordinator) cho bước "Classify Scenes". **Compensating action**: Không cần — bước này là pure computation, không side-effect bền vững cần rollback. Retry = gọi lại `classify_scenes`.

## Tech Stack Consistency
Xác nhận Python 3.12 + FastAPI (khớp `technology-direction.md`/ADR-0003, không lệch hướng cho unit này). Xem ADR-0009 cho quyết định polyglot có chọn lọc áp dụng cho các unit khác trong hệ thống (Orchestrator: Go, API Gateway: Node.js) — quyết định này phát sinh trong quá trình xác nhận tech stack của chính unit này, nhưng không thay đổi lựa chọn cho Unit 2.
