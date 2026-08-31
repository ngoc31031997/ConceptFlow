# NFR Design Patterns — Unit 8: Orchestrator Service

## Data Access Pattern: CRUD (not CQRS)
`projects`/`saga_steps` dùng cùng 1 model cho cả đọc (`GET /v1/projects/{id}`) và ghi (`HandleStepEventUseCase`, `StartRenderSagaUseCase`, ...) — không tách read/write model. `outbox_events`/`processed_messages` là bảng kỹ thuật hỗ trợ Inbox/Outbox, không phải read model nghiệp vụ.
**Rationale**: quy mô 1 Creator/máy dev cá nhân, không có tải đọc/ghi lệch hoặc nhu cầu query phức tạp biện minh cho CQRS.

## Resilience Pattern: No Automatic Retry — Explicit Retry-by-Step
- Khi 1 bước lỗi (`*_failed` event hoặc DLQ delivery), Orchestrator KHÔNG tự động retry — chuyển ngay `Project.Status → failed_at_<step>` (Functional Design Rule 8).
- Creator chủ động gọi `POST /v1/projects/{id}/retry` sau khi xem lỗi qua SSE — retry có chủ đích, không phải backoff-retry mù quáng.
- **Rationale**: nhiều lỗi ở hệ thống này không transient (vd. lỗi Manim template, script không hợp lệ) — retry tự động sẽ lặp lại lỗi tốn tài nguyên vô ích; Creator cần thấy lỗi cụ thể trước khi quyết định.
- Container-level recovery (crash, không phải business failure): Docker Compose `restart: unless-stopped` (NFR Requirements Question 4).

## Idempotency Pattern: 2 Tầng
1. **Message-level** (Inbox): dedupe `message_id` cho MỌI event nhận vào (12 loại) — chống xử lý trùng khi RabbitMQ redeliver.
2. **Step-level** (`SagaStep.status` guard): event chỉ xử lý nếu bước tương ứng đang `in_progress` (Functional Design Rule 4) — chống double-processing khi 1 service lỡ publish event 2 lần cho cùng 1 bước.

Không có artifact-level idempotency ở Orchestrator — đó là trách nhiệm từng service nghiệp vụ downstream (vd. Rendering Service skip scene đã render).

## Security Pattern
- Không auth/rate-limit riêng cho REST (Gateway ↔ Orchestrator) hay AMQP nội bộ — threat model single-user, local Docker network (NFR Requirements Question 5).
- Validate input cơ bản (zero-trust tối thiểu) ở REST layer: `project_id` tồn tại, enum hợp lệ (`visibility`, `voice_language`), required field trước khi tạo/update `Project`.
- KHÔNG validate lại dữ liệu payload từ AMQP event nhận vào — các service nghiệp vụ upstream đã validate trước khi publish, tin tưởng lẫn nhau trong hệ thống nội bộ 1 trust boundary.

## Saga Pattern: Orchestration-based (ADR-0007)
Orchestrator Service là **central coordinator** duy nhất — sở hữu toàn bộ định nghĩa thứ tự bước và state machine. Không phải choreography (các service nghiệp vụ không biết bước tiếp theo, chỉ publish event kết quả). Xem `messaging-design.md` cho chi tiết compensating action.

## Inbox/Outbox Pattern (semantic khác biệt so với các unit khác — xem ADR-0019)
- **Outbox**: dùng để đảm bảo COMMAND gửi đi đúng 1 lần (không phải event như Unit 2–7) — vì Orchestrator là bên khởi phát action.
- **Inbox**: dedupe EVENT nhận vào theo `message_id`.
- Relay: `OutboxRelay` poll `outbox_events` mỗi 500ms (NFR Requirements Question 3), publish qua `amqp.Publisher`.

## Logical Components
Xem `logical-components.md`.

## Không áp dụng
- **Caching**: không có nhu cầu (NFR Requirements).
- **Circuit Breaker**: không cần — không có external dependency đồng bộ nào ngoài Postgres/RabbitMQ (cả 2 đều local, cùng Docker network).
