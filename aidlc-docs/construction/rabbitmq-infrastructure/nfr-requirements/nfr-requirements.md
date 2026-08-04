# NFR Requirements — Unit 1: RabbitMQ Infrastructure

## Reliability
- **Delivery Guarantee**: At-least-once delivery — consumer ack sau khi xử lý xong; message redeliver nếu consumer crash trước ack. Kết hợp với idempotency ở tầng consumer (đã thiết kế tại `services.md`, mục Idempotency) để xử lý an toàn message trùng.
- **Persistence**: Durable queue + persistent message (`delivery_mode=2`) — sống sót qua RabbitMQ container restart.
- **Message TTL**: 24 giờ — tự động dọn message "mồ côi" nếu có lỗi xử lý không được resolve.
- **Dead-Letter Policy**: Retry tối đa 3 lần với exponential backoff (1s, 5s, 15s), sau đó message chuyển vào Dead-Letter Queue (DLQ) riêng cho từng queue nghiệp vụ (vd. `rendering.commands.dlq`). Orchestrator Service theo dõi DLQ và đánh dấu Saga step tương ứng là `failed_at_<step>`; Creator có thể trigger retry thủ công từ GUI (theo cơ chế đã thiết kế ở `services.md`).

## Observability
- **Monitoring**: Bật RabbitMQ Management Plugin, expose port 15672 (chỉ trong môi trường local/dev) để theo dõi queue depth, message rate, DLQ trong quá trình phát triển.

## Performance & Scalability
- Không có yêu cầu throughput/scale đặc biệt (theo `requirements.md` NFR2) — dùng cấu hình mặc định của RabbitMQ, không đặt resource limit (CPU/memory) riêng ở giai đoạn này.

## Messaging & Event Participation
- Unit này là hạ tầng thuần túy (message broker) — không tự publish/consume message nghiệp vụ; đóng vai trò trung gian định tuyến giữa Orchestrator Service và 5 service nghiệp vụ (theo `integration-boundaries.md`).

## Distributed Transaction Participation
- Không trực tiếp tham gia logic Saga (đó là trách nhiệm của Orchestrator Service, Unit 8) — chỉ cung cấp hạ tầng durable messaging cho Saga hoạt động.

## Security
- Security Baseline extension đã tắt (theo `requirements.md`). Ở mức tối thiểu: dùng credential mặc định khác `guest/guest` (đổi qua biến môi trường trong docker-compose), không expose port AMQP (5672) ra ngoài host network (chỉ trong docker-compose internal network) — đây là thực hành an toàn cơ bản, không phải blocking constraint theo extension.
