# ADR-0007: Saga Orchestration via Dedicated Orchestrator Service + Message Queue

## Status
Accepted

## Date
2026-08-04

## Stage
Application Design

## Context
Sau khi Application Design ban đầu hoàn tất (Gateway đóng vai trò orchestrator, gọi REST đồng bộ — ADR-0005), người dùng yêu cầu dùng Message Queue cho vai trò điều phối. Làm rõ qua câu hỏi follow-up: người dùng muốn **một service điều phối riêng biệt** (không phải Event-driven Choreography thuần túy), theo **Saga pattern** — tức Saga orchestration-based, không phải choreography-based.

## Options Considered
### Option A (giữ nguyên ADR-0005): Orchestration qua API Gateway, REST đồng bộ, không message broker
- What it is: Gateway vừa là entry point vừa là orchestrator, gọi REST tuần tự tới từng service.
- Strengths: Đơn giản nhất, không cần hạ tầng thêm.
- Trade-offs: Gateway gánh cả 2 trách nhiệm (routing + orchestration); không có cơ chế compensating transaction rõ ràng, retry xử lý thủ công ở tầng ứng dụng; không tách rời được nếu sau này cần queue/backpressure khi render lâu.

### Option B: Event-driven Choreography qua message broker (Kafka/RabbitMQ)
- What it is: Mỗi service tự lắng nghe sự kiện và tự kích hoạt bước tiếp theo, không có điều phối trung tâm.
- Strengths: Service độc lập nhất, không có single point of coordination.
- Trade-offs: Người dùng từ chối — khó theo dõi luồng tổng thể, không phù hợp với mong muốn có 1 điểm kiểm soát rõ ràng cho luồng nghiệp vụ tuyến tính.

### Option C (Chọn): Saga Orchestration-based với Orchestrator Service riêng biệt, giao tiếp qua Message Queue
- What it is: Tách một **Orchestrator Service** độc lập khỏi API Gateway. Gateway chỉ còn vai trò routing/entry point cho GUI. Orchestrator Service là nơi duy nhất biết thứ tự nghiệp vụ (Saga steps) và điều phối các service khác **thông qua Message Queue** (gửi command message, nhận event message xác nhận hoàn tất/lỗi từng bước), thay vì gọi REST đồng bộ trực tiếp. Mỗi bước Saga có định nghĩa compensating action rõ ràng để rollback khi bước sau thất bại.
- Strengths: Tách bạch rõ trách nhiệm (Gateway = routing, Orchestrator = business process); Message Queue cho phép xử lý bất đồng bộ, chịu lỗi tốt hơn (message không mất khi 1 service tạm thời down/chậm — đặc biệt hữu ích vì Rendering là bước chạy lâu); cơ chế compensating transaction của Saga được định nghĩa tường minh cho từng bước thay vì xử lý lỗi thủ công.
- Trade-offs: Thêm 2 thành phần hạ tầng mới (Orchestrator Service + Message Queue broker) vào docker-compose; độ phức tạp vận hành tăng so với ADR-0005 (cần quản lý thêm broker, thêm cơ chế idempotency cho message); cần thiết kế compensating action cụ thể cho từng bước (vd. xóa animation clip đã render nếu bước ghép video thất bại và Creator không muốn giữ lại).

## Decision
Chọn **Option C**: Saga Orchestration-based với **Orchestrator Service** riêng biệt, giao tiếp với các service nghiệp vụ (Content Plugin, Script Processing, Rendering, TTS, Video Assembly, Publisher) qua **Message Queue** (RabbitMQ). API Gateway giữ vai trò routing/entry point cho GUI (REST + SSE), chuyển tiếp yêu cầu khởi tạo pipeline sang Orchestrator Service.

**Message Queue cụ thể**: RabbitMQ — phù hợp mô hình command/task queue với routing theo từng service, hỗ trợ tốt cơ chế acknowledge/retry cần thiết cho Saga (so với Kafka vốn tối ưu hơn cho event streaming/log lớn, không cần thiết ở quy mô 1 người dùng hiện tại).

## Rationale
Người dùng xác nhận rõ muốn Saga orchestration-based (không phải choreography) qua một service điều phối riêng — điều này tách bạch rõ ràng giữa "routing" (Gateway) và "business process coordination" (Orchestrator), đồng thời Message Queue giúp Rendering (bước chạy lâu, có thể mất nhiều phút) không chặn (block) các thành phần khác và có cơ chế retry/durability tốt hơn REST đồng bộ.

## Consequences
- **Positive**: Trách nhiệm rõ ràng (Gateway vs Orchestrator); compensating transaction tường minh cho từng bước Saga; chịu lỗi tốt hơn nhờ message durability của RabbitMQ (không mất command nếu 1 service tạm thời unavailable); dễ mở rộng thêm bước Saga mới (vd. thêm domain giáo dục khác) mà không sửa Gateway.
- **Negative / Accepted Trade-offs**: Thêm RabbitMQ + Orchestrator Service vào docker-compose (tăng số container từ 8 lên 10); cần thiết kế idempotency cho consumer (xử lý trường hợp message được deliver nhiều lần); cần định nghĩa compensating action cụ thể cho từng bước — việc này sẽ chi tiết hóa ở Low-Level Design của Orchestrator Service.
- **Follow-ups**: Cập nhật `architecture-overview.md`, `integration-boundaries.md` (HLD), và `components.md`, `services.md`, `component-methods.md`, `component-dependency.md` (Application Design) để phản ánh Orchestrator Service + Message Queue. Low-Level Design của Orchestrator Service cần định nghĩa cụ thể Saga step definitions và compensating actions.

## Related
- Design artifact: `aidlc-docs/inception/high-level-design/integration-boundaries.md`, `aidlc-docs/inception/application-design/services.md`
- Related ADRs: Supersedes ADR-0005; related to ADR-0001 (Microservices), ADR-0004 (API Gateway)
