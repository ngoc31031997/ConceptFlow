# CR-003 — Năng lực render video dài 5–10 phút (P0, blocker)

## Date
2026-09-07

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
Mọi giới hạn tài nguyên/thời gian hiện tại được đặt cho video demo vài chục giây. Với video 5–10 phút, pipeline **luôn thất bại** trước khi ra được file.

## Vấn đề (đã xác minh trong code)

> **ĐÃ HIỆU CHỈNH THEO SỐ ĐO (Pha 0, 2026-09-07).** Bản đầu của CR này viết trước khi benchmark và **ước lượng sai 3 con số**. Số liệu thật và phần đính chính đầy đủ nằm ở `aidlc-docs/construction/build-and-test/long-form-baseline.md`. Bảng dưới đây đã cập nhật.

| Chỗ | Giá trị | Số đo thật (fixture 215s @1080p60) | Đánh giá |
|---|---|---|---|
| `manim_renderer.py` `_limit_child_resources` đặt `RLIMIT_CPU = timeout` | = wall-clock | **CPU/wall = 2.21×** | 🔴 **Bug đang hoạt động, nghiêm trọng nhất.** Tiến trình bị `SIGXCPU` ở wall-clock ≈ `timeout / 2.21`, tức giới hạn render thực tế **chỉ bằng 45%** con số ghi trong config (300s → chết ở ~136s). Đây là thứ đánh sập trước cả timeout. |
| `docker-compose.yml` `RENDER_TIMEOUT_SECONDS: "300"` | 300s | Render 98.9s cho video 215s ⇒ **~0.46× thời lượng**; video 10 phút ≈ **276s** | 🟠 Thiếu biên. Ước lượng ban đầu ("20–60+ phút") **sai ~10 lần**. Vẫn cần nới vì script nặng (3D/MathTex/updater) chậm hơn nhiều. |
| `manim_renderer.py` `RENDER_MEMORY_LIMIT_BYTES = 2 GiB` (`RLIMIT_AS`) | 2 GiB hardcode | Đỉnh **764 MB** | 🟡 Đủ cho workload này, nhưng hardcode và không có biên. **Đề xuất 8 GiB ban đầu là bất khả thi** — Docker VM chỉ có 7 GiB tổng. Giá trị đúng: **4 GiB, qua env**. |
| `docker-compose.yml` `ASSEMBLY_TIMEOUT_SECONDS: "180"` | 180s | Burn phụ đề 24.8s cho video 215s ⇒ **~0.115× thời lượng**; video 10 phút ≈ **70s** | 🟡 Ước lượng ban đầu ("5–15 phút") **sai ~6 lần**. 180s thực ra đủ dùng, chỉ thiếu biên an toàn. |
| RabbitMQ `consumer_timeout` (không đặt trong `rabbitmq.conf`) | mặc định 30 phút | Render thật ~4.6 phút cho video 10 phút | 🟡 **Hạ mức ưu tiên**: chưa bị chạm ở workload thường, không còn là "bug sắp nổ" như bản CR đầu mô tả. Vẫn nên đặt tường minh để có biên. |
| `manim_renderer.py` cờ `--disable_caching` | luôn bật | — | 🟡 Sửa 1 câu narration ⇒ render lại toàn bộ. Chi phí thấp hơn dự đoán (vài phút, không phải 40 phút) nhưng vẫn đáng cải thiện. |
| Không có checkpoint | — | — | 🟡 Ít cấp bách hơn dự đoán, vì render không còn là tác vụ 40 phút. |
| Progress | theo step | Render 4–5 phút không tín hiệu | 🟠 Vẫn là vấn đề UX thật. |
| `docker-compose.yml` | không có `deploy.resources` | — | 🟠 Render nặng làm nghẽn cả stack. |

## Functional Requirements

### FR11 — Render dài hạn (mới)
- **FR11.1**: Timeout render PHẢI cấu hình được, mặc định **1800s** (6.5× biên trên 276s đo được). Timeout assembly mặc định **900s** (12× biên trên 70s đo được).
- **FR11.2**: Giới hạn bộ nhớ render PHẢI cấu hình được (`RENDER_MEMORY_LIMIT_GB`, mặc định **4** — không phải 8, vì Docker VM chỉ có 7 GiB tổng).
- **FR11.3**: `RLIMIT_CPU` PHẢI **bị bỏ**. Số đo cho thấy nó luôn cắt sớm (CPU/wall = 2.21×); timeout wall-clock của `subprocess.run` đã là cơ chế chặn đúng và đủ.
- **FR11.4**: Trong lúc render, Rendering Service PHẢI phát progress định kỳ (ít nhất mỗi 15s) với % ước lượng, đọc từ stdout/stderr của Manim.
- **FR11.5**: Manim caching PHẢI bật được (`--disable_caching` thành tuỳ chọn), có cache dir bền trên volume, khoá theo hash script — để lần render lại sau khi sửa nhỏ nhanh hơn đáng kể.
- **FR11.6**: Mỗi container nặng (rendering, video-assembly) PHẢI có giới hạn CPU/RAM khai báo trong compose.
- **FR11.7 (nên có)**: Render theo **từng scene rồi concat** thay vì một lượt duy nhất, để (a) checkpoint được, (b) retry chỉ scene lỗi, (c) progress chính xác.

## Ràng buộc
- **C1**: FR11.7 là thay đổi kiến trúc lớn nhất của CR này và **phụ thuộc CR-002** (offset timeline phải cộng dồn qua các scene). Có thể tách thành pha 2.
- **C2 (đã hạ mức sau khi đo)**: RabbitMQ `consumer_timeout` mặc định 30 phút và không được override trong `infra/rabbitmq/rabbitmq.conf`. Số đo cho thấy render video 10 phút chỉ mất ~4.6 phút ⇒ **chưa chạm ngưỡng** ở workload thường. Vẫn đặt tường minh 3600s để có biên cho script nặng, nhưng đây **không phải** bug cấp bách như bản CR đầu mô tả.
- **C3**: Ổ đĩa — 10 phút 1080p60 từ Manim (frame trung gian + video) có thể chiếm hàng GB trên `shared_artifacts`; cần dọn dẹp artifact tạm.

## Phạm vi tác động
- **Rendering** (Unit 5), **Video Assembly** (Unit 6), **RabbitMQ Infra** (Unit 1), **Orchestrator** (Unit 8, progress), **Web GUI** (Unit 10, hiển thị %), `docker-compose.yml`.

## Giá trị chốt (Pha 0)
| Biến | Cũ | Mới |
|---|---|---|
| `RENDER_TIMEOUT_SECONDS` | 300 | **1800** |
| `RENDER_MEMORY_LIMIT_GB` | 2 (hardcode) | **4** (env) |
| `RLIMIT_CPU` | = timeout | **bỏ** |
| `ASSEMBLY_TIMEOUT_SECONDS` | 180 | **900** |
| RabbitMQ `consumer_timeout` | mặc định | **3600s** tường minh |

## Tiêu chí nghiệm thu
1. Render thành công 1 video Manim ~8 phút @1080p60 không timeout, không OOM.
2. RabbitMQ không drop/redeliver job trong suốt quá trình render dài.
3. Web GUI hiển thị tiến trình nhúc nhích ít nhất mỗi 15s.
4. Render lại sau khi sửa 1 dòng narration nhanh hơn rõ rệt nhờ cache.

## Câu hỏi còn mở
1. FR11.7 (render per-scene): **đề xuất HOÃN.** Căn cứ Pha 0 — render không còn là tác vụ 40 phút mà chỉ ~4.6 phút cho video 10 phút, nên lợi ích của checkpoint/retry-từng-scene nhỏ hơn nhiều so với chi phí thay đổi kiến trúc. Quyết ở bước 1C.5.
2. ~~Benchmark trên máy thật~~ → **ĐÃ LÀM** (Pha 0, xem `long-form-baseline.md`).
