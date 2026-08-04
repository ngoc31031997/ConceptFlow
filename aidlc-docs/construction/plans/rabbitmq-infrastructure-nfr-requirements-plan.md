# NFR Requirements Plan — Unit 1: RabbitMQ Infrastructure

## Scope
RabbitMQ đã được chọn ở ADR-0007 (thay Kafka). Ở đây xác định yêu cầu phi chức năng cụ thể cho việc vận hành RabbitMQ trong docker-compose local.

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Phân tích, follow-up nếu có mơ hồ
- [ ] Tạo `nfr-requirements.md`
- [ ] Tạo `tech-stack-decisions.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Delivery Guarantee
Theo `integration-boundaries.md`, Orchestrator gửi command và nhận event qua RabbitMQ. Mức đảm bảo giao message nào là cần thiết?

A) 💡 Suggested: At-least-once delivery (message được ack bởi consumer sau khi xử lý xong; nếu consumer crash trước khi ack, message được redeliver) — kết hợp với idempotency ở consumer (đã thiết kế ở `services.md`) để xử lý an toàn trường hợp nhận trùng
   - ✅ Strengths: không mất message nếu service tạm thời crash — quan trọng vì Rendering có thể chạy nhiều phút; khớp với thiết kế idempotency đã có
   - ⚠️ Trade-offs: có thể nhận message trùng (đã compensate bằng idempotency ở tầng consumer)

B) At-most-once (không cần ack, chấp nhận có thể mất message)
   - ✅ Strengths: đơn giản nhất
   - ⚠️ Trade-offs: mất command nếu service crash giữa chừng — không phù hợp với yêu cầu Saga đáng tin cậy

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Message Persistence & Retention
Message có cần lưu trên đĩa (durable) để không mất khi RabbitMQ container restart không?

A) 💡 Suggested: Durable queue + persistent message — message được ghi xuống đĩa, sống sót qua broker restart; TTL message = 24h (đủ cho 1 phiên làm việc dài, tự động dọn message "mồ côi" nếu có lỗi không xử lý)
   - ✅ Strengths: không mất tiến trình Saga nếu vô tình restart Docker; TTL tránh queue phình to vô hạn nếu có lỗi
   - ⚠️ Trade-offs: throughput thấp hơn message không persistent (không đáng kể ở quy mô 1 người dùng)

B) Non-durable, in-memory only — mất hết khi container restart

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Dead-Letter Queue Policy
Khi 1 message xử lý lỗi nhiều lần liên tiếp, nên xử lý thế nào?

A) 💡 Suggested: Retry tối đa 3 lần (với exponential backoff), sau đó chuyển vào Dead-Letter Queue (DLQ) riêng; Orchestrator theo dõi DLQ và đánh dấu Saga step đó là `failed` (đã thiết kế state `failed_at_<step>`), Creator có thể trigger retry thủ công từ GUI
   - ✅ Strengths: tránh vòng lặp retry vô hạn; khớp với cơ chế compensating action đã thiết kế ở `services.md`
   - ⚠️ Trade-offs: cần cấu hình DLQ cho từng queue nghiệp vụ

B) Retry vô hạn cho đến khi thành công
   - ✅ Strengths: đơn giản
   - ⚠️ Trade-offs: có thể treo Saga vô thời hạn nếu lỗi là do bug, không phải lỗi tạm thời

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Monitoring
Có cần RabbitMQ Management UI (giao diện web theo dõi queue) trong môi trường local không?

A) Có — bật RabbitMQ Management plugin, expose port riêng (vd. 15672) để debug queue trong quá trình phát triển
   - ✅ Strengths: dễ debug, xem trực quan queue depth/message rate khi phát triển
   - ⚠️ Trade-offs: expose thêm 1 port, không cần thiết ở "production" cá nhân nhưng hữu ích lúc dev

B) Không — chỉ dùng RabbitMQ core, không cần UI

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Resource Limits (Scalability/Performance)
requirements.md xác nhận không yêu cầu tốc độ/scale cao. Xác nhận: không cần đặt resource limit (CPU/memory) cụ thể cho RabbitMQ container ở giai đoạn này, dùng default?

A) Đúng — dùng cấu hình mặc định của RabbitMQ, không giới hạn tài nguyên đặc biệt

B) Other (please describe after [Answer]: tag below)

[Answer]: A
