# CR-003 — Năng lực render video dài 5–10 phút (P0, blocker)

## Date
2026-09-07

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
Mọi giới hạn tài nguyên/thời gian hiện tại được đặt cho video demo vài chục giây. Với video 5–10 phút, pipeline **luôn thất bại** trước khi ra được file.

## Vấn đề (đã xác minh trong code)

| Chỗ | Giá trị | Vì sao hỏng với video 10 phút |
|---|---|---|
| `docker-compose.yml` `RENDER_TIMEOUT_SECONDS: "300"` | 300s | Manim render 10 phút @1080p mất 20–60+ phút ⇒ luôn `AnimationEngineError: timed out` |
| `manim_renderer.py` `RENDER_MEMORY_LIMIT_BYTES = 2 GiB` (`RLIMIT_AS`) | 2 GiB | Manim + cairo + ffmpeg cho scene dài vượt 2 GiB address-space ⇒ crash `MemoryError` khó chẩn đoán |
| `manim_renderer.py` `_limit_child_resources` đặt `RLIMIT_CPU = timeout` | = wall-clock | CPU-time ≠ wall-clock. Manim đa luồng đốt CPU-time nhanh hơn wall-clock ⇒ bị `SIGXCPU` **sớm hơn cả timeout** |
| `docker-compose.yml` `ASSEMBLY_TIMEOUT_SECONDS: "180"` | 180s | Khi bật phụ đề, `ffmpeg_assembler.py` re-encode toàn bộ (`libx264 -preset medium -crf 23`); 1080p 10 phút mất 5–15 phút ⇒ luôn timeout |
| `manim_renderer.py` cờ `--disable_caching` | luôn bật | Sửa 1 câu narration ⇒ render lại toàn bộ 40 phút |
| Không có checkpoint | — | Fail ở phút 39/40 ⇒ mất trắng |
| Progress | theo step | 40 phút không có tín hiệu nào, Creator tưởng treo |
| `docker-compose.yml` | không có `deploy.resources` | Render nặng làm nghẽn cả stack (Postgres/RabbitMQ) |

## Functional Requirements

### FR11 — Render dài hạn (mới)
- **FR11.1**: Timeout render PHẢI cấu hình được và mặc định đủ cho video 10 phút (đề xuất 5400s). Timeout assembly mặc định 1800s.
- **FR11.2**: Giới hạn bộ nhớ render PHẢI cấu hình được (`RENDER_MEMORY_LIMIT_GB`, mặc định 8).
- **FR11.3**: `RLIMIT_CPU` PHẢI được tính tách khỏi wall-clock (đề xuất bỏ, hoặc = `timeout × số core`) để không giết tiến trình render hợp lệ.
- **FR11.4**: Trong lúc render, Rendering Service PHẢI phát progress định kỳ (ít nhất mỗi 15s) với % ước lượng, đọc từ stdout/stderr của Manim.
- **FR11.5**: Manim caching PHẢI bật được (`--disable_caching` thành tuỳ chọn), có cache dir bền trên volume, khoá theo hash script — để lần render lại sau khi sửa nhỏ nhanh hơn đáng kể.
- **FR11.6**: Mỗi container nặng (rendering, video-assembly) PHẢI có giới hạn CPU/RAM khai báo trong compose.
- **FR11.7 (nên có)**: Render theo **từng scene rồi concat** thay vì một lượt duy nhất, để (a) checkpoint được, (b) retry chỉ scene lỗi, (c) progress chính xác.

## Ràng buộc
- **C1**: FR11.7 là thay đổi kiến trúc lớn nhất của CR này và **phụ thuộc CR-002** (offset timeline phải cộng dồn qua các scene). Có thể tách thành pha 2.
- **C2**: RabbitMQ consumer phải chịu được job chạy > 1 giờ — kiểm tra `consumer_timeout` của RabbitMQ (mặc định 30 phút, sẽ **giết consumer** giữa lúc render!). Đây là bug tiềm ẩn thứ hai, phải sửa trong `infra/rabbitmq/rabbitmq.conf`.
- **C3**: Ổ đĩa — 10 phút 1080p60 từ Manim (frame trung gian + video) có thể chiếm hàng GB trên `shared_artifacts`; cần dọn dẹp artifact tạm.

## Phạm vi tác động
- **Rendering** (Unit 5), **Video Assembly** (Unit 6), **RabbitMQ Infra** (Unit 1), **Orchestrator** (Unit 8, progress), **Web GUI** (Unit 10, hiển thị %), `docker-compose.yml`.

## Tiêu chí nghiệm thu
1. Render thành công 1 video Manim ~8 phút @1080p60 không timeout, không OOM.
2. RabbitMQ không drop/redeliver job trong suốt quá trình render dài.
3. Web GUI hiển thị tiến trình nhúc nhích ít nhất mỗi 15s.
4. Render lại sau khi sửa 1 dòng narration nhanh hơn rõ rệt nhờ cache.

## Câu hỏi còn mở
1. FR11.7 (render per-scene) làm ngay trong CR này hay tách CR riêng sau khi CR-002 xong?
2. Giá trị timeout/RAM cụ thể phụ thuộc máy của Creator — cần benchmark 1 lần trên máy thật.
