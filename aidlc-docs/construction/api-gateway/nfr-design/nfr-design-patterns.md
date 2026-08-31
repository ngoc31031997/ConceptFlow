# NFR Design Patterns — Unit 9: API Gateway

## Data Access Pattern: Không áp dụng (No CRUD/CQRS)
Gateway không có data store riêng — không đọc/ghi database. Concept CRUD/CQRS không có ý nghĩa cho 1 pure proxy/routing layer.

## Resilience Pattern
- **Proxy request**: timeout cố định 30s (`AbortController`), không auto-retry, trả `502 {error: "upstream_unavailable", service}` khi downstream không kết nối được (LLD Flow 4).
- **AMQP consumer**: auto-reconnect backoff cố định 5s, không giới hạn số lần thử (kết nối nội bộ Docker network, không có lý do "give up" vĩnh viễn).
- Không circuit breaker — không cần thiết ở quy mô 1 Creator, kết nối downstream đều nội bộ Docker network ổn định.

## Idempotency Pattern: Không áp dụng ở tầng Gateway
Gateway forward request nguyên trạng, không tự quyết định "trùng lặp" vì không có ngữ cảnh nghiệp vụ. Idempotency của thao tác nghiệp vụ (vd. tạo `saga_id` mới) là trách nhiệm của service downstream (Orchestrator, Unit 8).

## Security Pattern
- Không auth/rate-limit nội bộ (threat model single-user local).
- Validate tối thiểu: `project_id` path param hợp lệ (basic sanitize chống path traversal dù nội bộ) — không validate sâu body/payload (trách nhiệm downstream, tránh double-validation).
- CORS: để ngỏ cho Unit 10 xác nhận cụ thể.

## Event-Driven Design / Saga / Inbox-Outbox: Không áp dụng đầy đủ
Gateway CHỈ consume `progress.fanout` (AMQP-to-SSE bridge), không publish message nghiệp vụ nào, không tham gia Saga (không phải orchestrator, không phải participant). KHÔNG cần Inbox/Outbox — không có state cần đồng bộ với message gửi/nhận. Chi tiết consumer xem `messaging-design.md`.

## Logical Components
Xem `logical-components.md`.

## Không áp dụng
- **Caching**: không cần — proxy nguyên trạng, dữ liệu downstream thay đổi liên tục.
- **Circuit Breaker**: không cần — kết nối downstream nội bộ Docker network.
- **Rate Limiter**: không cần — threat model single-user.
