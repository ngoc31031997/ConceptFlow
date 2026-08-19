# Tech Stack Decisions — Unit 5: Rendering Service

## Language/Runtime: Python 3.12
- **Rationale**: Ràng buộc kỹ thuật cứng — Manim chỉ có Python binding chính thức. Khớp ADR-0009.

## Animation Engine: Manim Community Edition
- **Ecosystem**: Thư viện animation toán học/lập trình mã nguồn mở, phù hợp trực tiếp với mục tiêu sản phẩm (video giáo dục kiểu 3Blue1Brown). Hỗ trợ sẵn `Code` mobject (Pygments-based syntax highlight) cho Story B3.
- **Performance**: CPU-bound nặng, chạy trong threadpool (không block event loop, xem `nfr-requirements.md`).
- **System dependency**: Cần ffmpeg (ghép frame thành video) + có thể cần LaTeX (cho công thức toán, không bắt buộc ở MVP nếu chỉ minh họa code/thuật toán không cần công thức) — quyết định cụ thể về LaTeX ở Infrastructure Design.

## Messaging Client: aio-pika
Nhất quán Unit 1/2/3/4.

## Database Client: asyncpg
Nhất quán Unit 2/3/4 (ADR-0013).

## Testing
- `pytest` + `pytest-asyncio` (`asyncio_mode = "auto"`) — nhất quán.
- `ruff` cho lint/format.
- Test cho Manim rendering thực tế (integration, không phải unit test thuần) sẽ cần cân nhắc riêng ở Code Generation — unit test business logic dùng `FakeAnimationRenderer`/`FakeTemplate`, không chạy Manim thật.

Không cần ADR riêng cho các lựa chọn framework/testing — hệ quả trực tiếp của ADR-0009/ADR-0013. Animation engine (Manim) không phải trade-off cạnh tranh (đã xác định từ Inception — "dùng Manim làm animation engine lõi", `technology-direction.md`), không cần ADR riêng.
