# High-Level Design — Follow-up Questions

## Contradiction 1: Microservices (Q2) vs. Single-container Docker local (Q8) vs. yêu cầu 1 người dùng/local-first (requirements.md)
Bạn chọn (B) Microservices ở Câu 2 — các service độc lập (Render Service, TTS Service, Publisher Service...) giao tiếp qua HTTP/message broker. Nhưng:
- Câu 8 bạn xác nhận (A) — triển khai dạng single-container Docker trên máy cá nhân
- `requirements.md` (NFR3, Q8 gốc) đã xác định: chỉ 1 người dùng, chạy local, "cloud/server tính sau", không yêu cầu render song song/batch (NFR2)

Microservices thường đi kèm chi phí vận hành (service discovery, network giữa các service, resilience/retry, có thể cần message broker riêng) — chi phí này thường chỉ hợp lý khi có nhu cầu scale độc lập hoặc nhiều team/nhiều máy. Với 1 người dùng chạy trên 1 máy qua Docker, chi phí này có vẻ vượt quá lợi ích thực tế ở giai đoạn này.

### Clarification Question 1
Bạn có thể làm rõ lý do chọn Microservices, hoặc xác nhận lại lựa chọn phù hợp hơn?

A) Tôi thực ra muốn kiến trúc **Modular Monolith** (module hóa rõ ràng bên trong 1 backend service, dễ tách thành microservices sau này nếu cần) — chọn nhầm ở Câu 2, xác nhận lại theo hướng này

B) Tôi vẫn muốn Microservices thực sự ngay từ bây giờ, mỗi service (Render, TTS, Publisher...) chạy như container riêng biệt trong cùng docker-compose trên 1 máy — chấp nhận thêm độ phức tạp vận hành (network giữa container, message broker nếu cần) để có ranh giới service rõ ràng và dễ tách máy sau này

C) Other (please describe after [Answer]: tag below)

[Answer]: B

## Ambiguity 2: API Gateway cần thiết — nhưng chưa rõ chi tiết
Ở Câu 6, bạn chọn (B) — không đồng ý với đề xuất "không cần gateway", cho biết dự tính có nhiều service độc lập cần gateway, nhưng chưa mô tả cụ thể.

### Clarification Question 2
Nếu hệ thống đi theo hướng Microservices thật (lựa chọn B ở Clarification Question 1), bạn hình dung API Gateway đóng vai trò gì cụ thể?

A) Một gateway đơn giản (vd. reverse proxy nhẹ như Nginx/Traefik, hoặc 1 FastAPI gateway service) đứng trước tất cả service, GUI chỉ gọi vào gateway — gateway định tuyến (route) request đến đúng service (Render/TTS/Publisher/...)
   - ✅ Strengths: GUI chỉ cần biết 1 điểm vào duy nhất, dễ thêm auth/rate-limit tập trung sau này
   - ⚠️ Trade-offs: thêm 1 thành phần cần vận hành, thêm 1 điểm có thể lỗi (single point of failure) dù chạy local

B) Không cần gateway riêng — nếu chọn (A) Modular Monolith ở Clarification Question 1 thì câu hỏi này không áp dụng (N/A)

C) Other (please describe after [Answer]: tag below)

[Answer]: A
