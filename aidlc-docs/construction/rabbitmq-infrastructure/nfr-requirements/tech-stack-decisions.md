# Tech Stack Decisions — Unit 1: RabbitMQ Infrastructure

## Message Broker: RabbitMQ
- **Version**: RabbitMQ 3.13 (latest stable tại thời điểm thiết kế), image `rabbitmq:3.13-management` (bao gồm Management Plugin theo NFR Requirements Q4)
- **Rationale**: Đã quyết định ở ADR-0007 (Application Design) — command/task queue nature, dead-letter/retry native, phù hợp quy mô 1 người dùng/1 máy. Xem ADR-0007 để biết phân tích đầy đủ so với Kafka.
- **Consistency với system-wide direction**: Khớp với `technology-direction.md` (HLD) — không có polyglot deviation nào ở unit này.

## Client Library (cho các service khác kết nối RabbitMQ)
- **Python**: `aio-pika` (async AMQP client cho Python, tương thích FastAPI/asyncio) — dùng chung cho tất cả service Python (Orchestrator, Content Plugin, Script Processing, Rendering, Video Assembly, Publisher).
  - **Ecosystem**: mature, hỗ trợ tốt asyncio, phù hợp FastAPI (đã chọn ở ADR-0003)
  - **Alternative cân nhắc**: `pika` (sync client) — không chọn vì các service dùng FastAPI (async), `aio-pika` khớp tự nhiên hơn, tránh block event loop
  - **Team familiarity**: Không có ràng buộc kinh nghiệm trước đó (dự án mới); `aio-pika` có tài liệu rõ ràng, cộng đồng ổn định

Không cần ADR riêng cho quyết định này vì đây là hệ quả trực tiếp, không có trade-off đáng kể khác ngoài lựa chọn RabbitMQ đã ghi ở ADR-0007 (không phải quyết định độc lập có nhiều phương án cạnh tranh thực sự).
