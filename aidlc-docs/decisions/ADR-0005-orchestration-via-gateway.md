# ADR-0005: Orchestration Pattern via API Gateway (No Message Broker)

## Status
Superseded by ADR-0007

## Date
2026-08-04

## Stage
High-Level Design

## Context
Với kiến trúc Microservices (ADR-0001), luồng tạo video đi qua nhiều service theo thứ tự (Script Processing → Content Plugin → Rendering + TTS → Video Assembly → Publisher). Cần quyết định cách phối hợp (choreography vs. orchestration) và cách xử lý tính nhất quán giữa các bước.

## Options Considered
### Option A: Orchestration qua API Gateway (không dùng message broker)
- What it is: API Gateway gọi tuần tự các service theo đúng thứ tự nghiệp vụ dựa trên kết quả bước trước; lỗi giữa chừng được đánh dấu ở trạng thái dự án, cho phép retry từng bước.
- Strengths: Đơn giản, dễ hiểu, dễ debug (luồng tuyến tính rõ ràng theo code của Gateway); không cần thêm hạ tầng message broker.
- Trade-offs: Gateway trở thành điểm phối hợp trung tâm — nếu sau này cần nhiều luồng nghiệp vụ phức tạp hơn, có thể cần tách orchestrator riêng.

### Option B: Choreography qua message broker/event bus
- What it is: Mỗi service tự lắng nghe sự kiện từ service trước và tự kích hoạt hành động tiếp theo, không có điều phối trung tâm.
- Strengths: Service độc lập hơn, không có single point of coordination; phù hợp khi có nhiều consumer độc lập cho cùng sự kiện.
- Trade-offs: Cần thêm message broker (Kafka/RabbitMQ/...) — hạ tầng không cần thiết cho pipeline tuyến tính, 1 người dùng; khó theo dõi luồng tổng thể (distributed tracing phức tạp hơn) so với quy mô hiện tại.

## Decision
Chọn **Option A: Orchestration qua API Gateway**, với eventual consistency đơn giản và compensating action thủ công tối thiểu (giữ kết quả các bước đã hoàn thành, cho phép retry bước lỗi) thay vì Saga pattern đầy đủ.

## Rationale
Luồng nghiệp vụ là tuyến tính cho 1 video tại 1 thời điểm, không có nhiều consumer độc lập hay xử lý song song đa video (theo NFR2: không yêu cầu render batch). Choreography và message broker chỉ có giá trị khi có independent consumers hoặc cần xử lý sự kiện phân tán phức tạp — không đúng với quy mô 1 người dùng, chạy local hiện tại. Saga pattern đầy đủ (với compensating transaction cho từng bước) là over-engineering khi không có ghi dữ liệu đồng thời vào cùng entity từ nhiều service.

## Consequences
- **Positive**: Luồng nghiệp vụ dễ theo dõi và debug (nằm trong logic của Gateway); không cần thêm hạ tầng message broker.
- **Negative / Accepted Trade-offs**: Gateway chịu trách nhiệm phối hợp nhiều hơn (không chỉ routing thuần túy); nếu hệ thống mở rộng sang xử lý nhiều video song song sau này, có thể cần tách orchestrator riêng khỏi Gateway.
- **Follow-ups**: Application Design cần định nghĩa cụ thể state machine của "video project" (draft/processing/failed-at-step/published) mà Gateway quản lý.

## Superseded
Người dùng yêu cầu, sau khi Application Design đã hoàn tất, chuyển sang mô hình **Saga Orchestration với một Orchestrator Service riêng biệt, giao tiếp qua Message Queue**, để có cơ chế compensating transaction rõ ràng và tách trách nhiệm điều phối khỏi Gateway (routing thuần túy). Xem ADR-0007.

## Related
- Design artifact: `aidlc-docs/inception/high-level-design/integration-boundaries.md` (đã cập nhật theo ADR-0007)
- Related ADRs: ADR-0001, ADR-0004, ADR-0007 (supersedes this ADR)
