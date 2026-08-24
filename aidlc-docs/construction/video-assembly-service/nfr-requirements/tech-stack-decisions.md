# Tech Stack Decisions — Unit 6: Video Assembly Service

## Language/Runtime: Python 3.12
- **Rationale**: Nhất quán ADR-0009 (Python cho hầu hết service). Không có ràng buộc kỹ thuật cứng nào buộc phải dùng ngôn ngữ khác — ffmpeg là external binary, gọi được từ bất kỳ ngôn ngữ nào, nhưng giữ Python để đồng nhất tooling/CI với Unit 2/3/4/5.

## Video/Audio Assembly: ffmpeg + ffprobe (external binaries, subprocess)
- **Ecosystem**: Công cụ chuẩn ngành cho ghép video/audio, tương thích tốt với output Manim (`technology-direction.md`). `ffprobe` (cùng bộ cài đặt) dùng cho media-format pre-check (Functional Design Question 3).
- **Invocation**: `subprocess` gọi CLI trực tiếp, không dùng wrapper library (`ffmpeg-python`) — kiểm soát đầy đủ command args, dễ debug (Low-Level Design Question 3).
- **Performance**: Chủ yếu dùng `-c copy` (không re-encode video) nên nhẹ hơn CPU-bound task như Manim render; vẫn chạy trong `ThreadPoolExecutor` để không block event loop khi gọi `subprocess.run()`.
- **System dependency**: Cần cài `ffmpeg` (bao gồm `ffprobe`) trong Docker image — quyết định cụ thể về base image ở Infrastructure Design.

## Messaging Client: aio-pika
Nhất quán Unit 1/2/3/4/5.

## Database Client: asyncpg
Nhất quán Unit 2/3/4/5 (ADR-0013, Postgres Inbox/Outbox — xác nhận rõ ràng theo yêu cầu người dùng ở NFR Requirements Question 1).

## Testing
- `pytest` + `pytest-asyncio` (`asyncio_mode = "auto"`) — nhất quán.
- `ruff` cho lint/format.
- Test cho ffmpeg/ffprobe thực tế (integration, không phải unit test thuần) sẽ cần cân nhắc riêng ở Code Generation — unit test business logic dùng `FakeVideoAssembler`, không gọi subprocess thật.

Không cần ADR riêng cho các lựa chọn framework/testing — hệ quả trực tiếp của ADR-0009/ADR-0013. ffmpeg/ffprobe không phải trade-off cạnh tranh (đã xác định từ Inception — "ffmpeg cho Video Assembly", `technology-direction.md`), không cần ADR riêng.
