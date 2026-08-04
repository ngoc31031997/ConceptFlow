# Infrastructure Design Plan — Unit 1: RabbitMQ Infrastructure

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `infrastructure-design.md`
- [ ] Tạo `deployment-architecture.md`
- [ ] Trình bày để phê duyệt

## Not Applicable (đã xác nhận từ NFR Requirements/Design, không hỏi lại)
- **Database Read/Write Splitting / Sharding**: N/A — RabbitMQ không phải database quan hệ, không có read/write splitting theo nghĩa này
- **Load Balancer**: N/A — chỉ 1 instance RabbitMQ container, không cần cân bằng tải
- **API Gateway**: Đã quyết định ở HLD (ADR-0004) — không thuộc phạm vi unit này (là Unit 9 riêng)

---

## Clarifying Questions

### Question 1: Deployment Environment
Xác nhận môi trường triển khai cho RabbitMQ container (đã xác định ở HLD/NFR Requirements)?

A) Docker container trong docker-compose, chạy trên máy cá nhân (macOS, môi trường hiện tại của bạn), image `rabbitmq:3.13-management`

B) Other (please describe after [Answer]: tag below)

[Answer]: 

### Question 2: Storage/Persistence
Data của RabbitMQ (queue, message durable) cần lưu ở đâu để không mất khi container bị xóa/tạo lại (không chỉ restart)?

A) 💡 Suggested: Docker named volume (`rabbitmq_data`) mount vào `/var/lib/rabbitmq` trong container — dữ liệu tồn tại độc lập với vòng đời container, chỉ mất khi volume bị xóa tường minh
   - ✅ Strengths: chuẩn Docker, tách biệt data khỏi container lifecycle, dễ backup (`docker volume` commands)
   - ⚠️ Trade-offs: không có, đây là cách làm chuẩn cho stateful service trong Docker

B) Bind mount tới thư mục cụ thể trên host (vd. `./data/rabbitmq`)
   - ✅ Strengths: dễ xem/backup trực tiếp qua file explorer
   - ⚠️ Trade-offs: phụ thuộc quyền file hệ điều hành host, kém portable hơn named volume

C) Other (please describe after [Answer]: tag below)

[Answer]: 

### Question 3: Networking
Port nào cần expose ra host, port nào chỉ nội bộ docker-compose network?

A) 💡 Suggested: AMQP port (5672) CHỈ nội bộ docker network (không map ra host — chỉ các service khác trong cùng docker-compose truy cập qua tên service `rabbitmq:5672`); Management UI port (15672) map ra host (`localhost:15672`) để bạn truy cập từ trình duyệt lúc dev
   - ✅ Strengths: giảm bề mặt tấn công (AMQP không expose ra host), vẫn tiện debug qua Management UI
   - ⚠️ Trade-offs: không thể kết nối AMQP trực tiếp từ máy host (chỉ từ trong container khác) — chấp nhận được vì mọi service đều chạy trong docker-compose

B) Expose cả 2 port ra host

C) Other (please describe after [Answer]: tag below)

[Answer]: 

### Question 4: Scaling Configuration
Xác nhận: không cần auto-scaling cho RabbitMQ (1 instance cố định), đúng theo NFR Requirements (không yêu cầu scale)?

A) Đúng — 1 instance cố định, không auto-scaling

B) Other (please describe after [Answer]: tag below)

[Answer]: 

### Question 5: Health Check
docker-compose có cần health check để các service khác biết RabbitMQ đã sẵn sàng trước khi kết nối không?

A) 💡 Suggested: Có — dùng `healthcheck` trong docker-compose gọi `rabbitmq-diagnostics ping`, các service khác dùng `depends_on: condition: service_healthy` để đợi RabbitMQ sẵn sàng trước khi start
   - ✅ Strengths: tránh lỗi "connection refused" khi service khác start trước RabbitMQ sẵn sàng
   - ⚠️ Trade-offs: không có, đây là thực hành chuẩn cho docker-compose multi-service

B) Không cần, để service tự retry kết nối

C) Other (please describe after [Answer]: tag below)

[Answer]: 
