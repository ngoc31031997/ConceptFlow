# NFR Requirements — Unit 9: API Gateway

## Scalability
- Không giới hạn cứng số SSE connection đồng thời — Node.js event loop xử lý dễ dàng ở quy mô 1 Creator/vài project song song.

## Performance
- Timeout cố định 30s cho mọi request proxy (`AbortController`), đủ cho REST request khởi tạo Saga (không chờ toàn bộ pipeline hoàn tất).

## Availability & Reliability
- AMQP consumer tự động reconnect (backoff cố định 5s, không giới hạn số lần thử) khi mất kết nối RabbitMQ.
- SSE client vẫn giữ kết nối mở trong lúc Gateway mất kết nối AMQP, chỉ tạm không nhận progress update — không phải lỗi nghiêm trọng, GUI có thể fallback `GET /v1/projects/{id}`.
- Không retry tự động cho request proxy lỗi (nhất quán triết lý Unit 8 — để Creator/GUI quyết định thử lại).

## Security
- Không auth/rate-limit cho GUI ↔ Gateway (threat model single-user local).
- CORS: để ngỏ, xác nhận cụ thể khi Unit 10 (Web GUI) quyết định kiến trúc serving (GUI riêng port hay serve qua Gateway).

## Maintainability
- Structured JSON logging qua `pino` — mỗi log line: `timestamp`, `level`, `message`, `requestId?`, `projectId?`.

## Messaging & Event Participation
- Gateway CHỈ consume `progress.fanout`, không publish gì tới RabbitMQ, không tham gia Saga.
- Delivery guarantee: at-most-once cho progress (chấp nhận được — progress chỉ mang tính UX, source of truth thật là `Project.Status` qua `GET /v1/projects/{id}`).

## Distributed Transaction Participation
- Không áp dụng — Gateway không phải Saga orchestrator/participant.

## Caching
- Không áp dụng — Gateway proxy nguyên trạng, không cache response (dữ liệu downstream thay đổi liên tục trong lúc Saga chạy).

## Usability
- SSE là cơ chế chính cho tiến trình real-time (Story C6); REST cho mọi thao tác cấu hình khác.
