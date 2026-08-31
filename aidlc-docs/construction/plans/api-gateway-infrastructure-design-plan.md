# Infrastructure Design Plan — Unit 9: API Gateway

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `infrastructure-design.md`
- [x] Tạo `deployment-architecture.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Deployment Environment
A) 💡 Suggested: Docker container, base `node:20-alpine`. Cùng docker network `backend`
   - ✅ Strengths: image nhỏ gọn, nhất quán network
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Storage Infrastructure
A) 💡 Suggested: KHÔNG cần database/volume riêng — Gateway hoàn toàn stateless (NFR Design đã xác nhận)
   - ✅ Strengths: đơn giản, không hạ tầng thừa
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Networking — Port Exposure (đặc thù unit này — Gateway là entry point DUY NHẤT ra host)
Khác mọi service khác (chỉ nội bộ `backend` network), Gateway PHẢI expose port ra host vì đây là entry point cho GUI (browser của Creator, chạy trên host, không phải container khác).

A) 💡 Suggested: Map port `8080:8080` ra host (container lắng nghe `8080`) — GUI (Unit 10, dù chạy dev server riêng hay được Gateway serve tĩnh) gọi API qua `http://localhost:8080`. Đây là port DUY NHẤT trong toàn hệ thống map ra host ngoài RabbitMQ Management UI (`15672`, dev/debug only)
   - ✅ Strengths: đúng vai trò Gateway = single entry point, nhất quán `integration-boundaries.md`'s thiết kế (GUI ↔ Gateway là ranh giới duy nhất Creator's browser chạm tới)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Health Check
A) 💡 Suggested: `GET /health` trả `200 {status: "ok"}` (đã thiết kế ở LLD's routing table) — không kiểm tra downstream (Gateway "khỏe" không phụ thuộc downstream có sẵn sàng hay không, tránh cascading health-check failure)
   - ✅ Strengths: đơn giản, tránh false negative khi 1 downstream tạm down nhưng Gateway vẫn hoạt động bình thường cho các route khác
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Messaging Infrastructure
A) 💡 Suggested: Kết nối `rabbitmq:5672` nội bộ, declare 1 exclusive queue bind vào `progress.fanout` lúc start (LLD/NFR Design đã xác nhận) — không cần thêm exchange/queue mới ở tầng hạ tầng Unit 1
   - ✅ Strengths: tận dụng hạ tầng có sẵn
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Database Read/Write Splitting / Sharding
A) 💡 Suggested: Không áp dụng — Gateway không có database
   - ✅ Strengths: đúng bản chất
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7: Load Balancer
A) 💡 Suggested: Không áp dụng — 1 instance duy nhất (Fixed, nhất quán các unit khác)
   - ✅ Strengths: đơn giản
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8: API Gateway Infrastructure (chính Unit này LÀ Gateway — xác nhận không cần thêm 1 lớp gateway khác)
A) 💡 Suggested: Unit 9 CHÍNH LÀ implementation cụ thể của "API Gateway" đã quyết định ở `integration-boundaries.md` (Inception) — không cần thêm 1 sản phẩm gateway thương mại nào khác (Kong/Traefik/AWS API Gateway) phía trước nó. Đủ cho quy mô 1 Creator/local
   - ✅ Strengths: đúng bản chất, tránh over-engineer thêm 1 lớp gateway nữa
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 9: Monitoring Infrastructure
A) 💡 Suggested: Không có monitoring stack riêng — `docker-compose logs` + `pino` structured JSON logging (NFR Requirements), nhất quán các unit khác
   - ✅ Strengths: đơn giản, đúng quy mô
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
