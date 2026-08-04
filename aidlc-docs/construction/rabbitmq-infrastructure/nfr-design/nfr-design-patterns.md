# NFR Design Patterns — Unit 1: RabbitMQ Infrastructure

## CRUD vs CQRS
**N/A** — Unit này không sở hữu data model nghiệp vụ nào (không phải service đọc/ghi dữ liệu domain), chỉ là hạ tầng message broker trung chuyển. CRUD/CQRS không áp dụng.

## Resilience Pattern
- **Retry**: Exponential backoff (1s → 5s → 15s), tối đa 3 lần, cấu hình qua RabbitMQ policy trên từng queue command.
- **Dead-Letter Queue (DLQ)**: Mỗi queue command có 1 DLQ tương ứng (`<queue-name>.dlq`) nhận message sau khi hết số lần retry. Orchestrator Service (Unit 8) là consumer của các DLQ này để cập nhật trạng thái Saga = `failed_at_<step>`.
- **Message TTL**: 24h trên tất cả queue command (theo NFR Requirements).

## Saga Role
**Transport only (N/A vai trò coordinator)** — RabbitMQ không chứa logic Saga. Orchestrator Service (Unit 8) là Saga coordinator duy nhất; RabbitMQ chỉ định tuyến command/event giữa Orchestrator và các service nghiệp vụ. Quyết định này xác nhận qua NFR Design Question 3 (giải thích thêm về khái niệm Saga coordinator được cung cấp trong quá trình planning).

## Security Pattern
- Credential mặc định (`guest/guest`) bị đổi qua biến môi trường `RABBITMQ_DEFAULT_USER`/`RABBITMQ_DEFAULT_PASS` trong docker-compose.
- Port AMQP (5672) và Management UI (15672) chỉ expose trong docker-compose internal network; Management UI có thể map ra host port cho mục đích dev/debug (không public ra internet vì chạy local).
