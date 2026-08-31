# Tech Stack Decisions — Unit 9: API Gateway

## Language & Runtime: Node.js (LTS, 20.x)
- Đã chốt ở ADR-0009 (system-wide selective polyglot) — mô hình reverse-proxy/event-loop khác biệt rõ với Python async, pattern phổ biến thực tế cho API Gateway.

## Framework: Express 4.x
- **Ecosystem/library maturity**: framework phổ biến nhất cho reverse-proxy pattern, middleware ecosystem lớn, tài liệu/cộng đồng rộng nhất trong hệ Node.js.
- **Performance**: đủ nhanh cho quy mô 1 Creator — Fastify có throughput cao hơn ở benchmark tải lớn nhưng không có ý nghĩa thực tế ở đây.
- **Team familiarity**: N/A mới với dự án, nhưng Express là lựa chọn phổ biến nhất, tài liệu/ví dụ tham khảo nhiều nhất cho pattern gateway.
- **Long-term maintenance**: cộng đồng lớn, ổn định lâu năm.
- **Licensing/cost**: MIT, mã nguồn mở, không chi phí.

## Logging: pino
- Structured JSON logging hiệu năng cao, chuẩn cộng đồng Node.js.

## AMQP Client: amqplib
- Client RabbitMQ chuẩn cộng đồng Node.js (nhất quán ADR-0009's follow-up note).

## Consistency with System-Wide Direction
`technology-direction.md`/ADR-0009 chỉ định Node.js cho API Gateway cụ thể — NFR Requirements xác nhận (không đảo ngược) và chọn framework/library cụ thể trong hệ Node.js. Xem ADR-0020.

## Related ADR
`aidlc-docs/decisions/ADR-0020-api-gateway-node-express-stack.md`.
