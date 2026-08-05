# Tech Stack Decisions — Unit 2: Content Plugin Service

## Language/Runtime: Python 3.12
- **Rationale**: Khớp `technology-direction.md` (ADR-0003); không có lý do kỹ thuật hay sư phạm để lệch hướng cho unit này (xem ADR-0009 — polyglot có chọn lọc chỉ áp dụng cho Orchestrator/Gateway).

## Framework: FastAPI
- **Ecosystem**: Mature, hỗ trợ async tốt, tích hợp Pydantic cho validation (đáp ứng NFR Security ở trên), OpenAPI docs tự động.
- **Performance**: Đủ nhanh cho workload in-memory lookup của unit này (không phải yếu tố quyết định).
- **Team familiarity**: Không có kinh nghiệm trước đó được ghi nhận; FastAPI có tài liệu rõ ràng, learning curve thấp cho Python developer.
- **Maintenance**: Cộng đồng lớn, release cadence ổn định.
- **Licensing**: MIT, không có chi phí.

## Messaging Client: aio-pika
- Khớp quyết định tại `tech-stack-decisions.md` (Unit 1) — client AMQP async chuẩn cho toàn bộ service Python trong hệ thống.

## Testing
- `pytest` cho unit test (domain/application layer, dùng `FakeContentPluginRegistry` theo `dependency-injection.md`)
- `pytest-asyncio` cho test AMQP consumer/producer (async)

Không cần ADR riêng — các lựa chọn framework/testing library này là hệ quả trực tiếp của ADR-0003/ADR-0009, không có trade-off cạnh tranh đáng kể khác ở mức chi tiết này.
