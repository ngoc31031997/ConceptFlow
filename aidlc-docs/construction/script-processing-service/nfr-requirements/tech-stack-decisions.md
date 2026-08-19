# Tech Stack Decisions — Unit 4: Script Processing Service

## Language/Runtime: Python 3.12
- **Rationale**: Khớp `technology-direction.md`/ADR-0009. Không có ràng buộc kỹ thuật đặc biệt (không như TTS Service với Piper) — chọn theo hướng mặc định hệ thống.

## Messaging Client: aio-pika
- Nhất quán với Unit 1/2/3 — client AMQP async chuẩn cho toàn bộ service Python trong hệ thống.

## Database Client: asyncpg
- Nhất quán với Unit 2/Unit 3 (sau retrofit, ADR-0013) — client Postgres async, dùng cho Inbox/Outbox.

## Không dùng FastAPI/Pydantic
Unit này không có REST endpoint (chỉ AMQP, theo ADR-0012 — không gọi Content Plugin Service trực tiếp), nên không cần FastAPI. Validation input nằm trong domain layer (`MarkdownScriptParser`), không phải qua Pydantic schema.

## Testing
- `pytest` + `pytest-asyncio` (`asyncio_mode = "auto"`) — nhất quán Unit 2/Unit 3.
- `ruff` cho lint/format.

Không cần ADR riêng — hệ quả trực tiếp của ADR-0009/ADR-0013, không có trade-off cạnh tranh đáng kể ở mức chi tiết này.
