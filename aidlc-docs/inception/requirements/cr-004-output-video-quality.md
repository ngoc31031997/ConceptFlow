# CR-004 — Chất lượng hình ảnh & profile encode chuẩn YouTube (P1)

## Date
2026-09-07

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
Video hiện xuất ra ở **720p30** với profile encode mặc định của ffmpeg. Đây là mức dưới chuẩn của mọi kênh giáo dục đang kiếm tiền, và làm mất đúng thế mạnh chính của Manim (animation mượt).

## Vấn đề (đã xác minh trong code)

1. **Hardcode 720p30** — `manim_renderer.py::_run_manim` dùng cờ `-qm` (medium quality = 1280×720 @ 30fps). Không có cách cấu hình.
2. **Thiếu tham số encode bắt buộc** — `ffmpeg_assembler.py::_run_pipeline` chỉ đặt `-c:v libx264 -preset medium -crf 23`. Thiếu:
   - `-pix_fmt yuv420p` (thiếu ⇒ một số thiết bị/nền tảng không phát được)
   - `-movflags +faststart` (thiếu ⇒ preview và upload chậm)
   - `-c:a aac -b:a 192k -ar 48000` (đang để ffmpeg tự chọn codec/bitrate audio)
   - `-profile:v high -bf 2 -g <2×fps>` (khuyến nghị của YouTube)
3. **Luôn re-encode khi bật phụ đề** — nhánh `if request.subtitle_cues:` bỏ hoàn toàn `-c:v copy`, gây (a) chậm, (b) mất chất lượng thế hệ 2 trước khi YouTube re-encode lần nữa.
4. **`crf 23 / preset medium` quá thấp cho bản upload** — YouTube sẽ transcode lại, nên file nguồn cần dư chất lượng.
5. **Phụ đề cố định PlayRes 1920×1080** — `subtitle_file.py` (`PLAY_RES_X/Y`) đúng cho 16:9 1080p nhưng sẽ sai tỉ lệ nếu đổi độ phân giải/khung hình (liên quan CR-007).

## Functional Requirements

### FR12 — Chất lượng đầu ra (mới)
- **FR12.1**: Độ phân giải/fps render PHẢI cấu hình được (`RENDER_QUALITY`), mặc định **1080p60** (`-qh`), hỗ trợ 720p30 (`-qm`) và 4K60 (`-qk`).
- **FR12.2**: File xuất ra PHẢI dùng profile encode chuẩn upload: `-c:v libx264 -preset slow -crf 18 -pix_fmt yuv420p -profile:v high -bf 2 -g <2×fps> -movflags +faststart`, audio `-c:a aac -b:a 192k -ar 48000`.
- **FR12.3**: Khi KHÔNG cần vẽ đè lên hình, video stream PHẢI được `-c:v copy` (không re-encode thừa).
- **FR12.4**: Phụ đề nên được burn-in **trong Manim** (hoặc kèm soft-sub `-c:s mov_text` để YouTube tự bật CC) thay vì luôn re-encode ở khâu assembly. Phương án chốt ở Low-Level Design.
- **FR12.5**: `subtitle_file.py` PHẢI lấy PlayRes từ độ phân giải thật của video, không hardcode.
- **FR12.6**: Creator PHẢI chọn được preset chất lượng ở GUI (Nháp nhanh 720p30 / Chuẩn 1080p60 / Cao 4K60) — bản nháp để duyệt nội dung, bản cao để upload.

## Ràng buộc
- **C1**: 1080p60 làm thời gian render tăng ~3–4× so với 720p30 ⇒ **phụ thuộc CR-003** (timeout/RAM) phải xong trước.
- **C2**: `preset slow -crf 18` làm khâu assembly chậm hơn nhiều ⇒ cũng phụ thuộc CR-003.
- **C3**: 4K60 với Manim rất nặng, để tuỳ chọn nhưng không khuyến nghị mặc định.
- **C4**: FR12.4 (burn phụ đề trong Manim) đụng vào script của Creator — cần cân nhắc kỹ vì Manim script là do người dùng viết. Phương án an toàn hơn: giữ burn ở ffmpeg nhưng dùng profile FR12.2.

## Phạm vi tác động
- **Rendering** (Unit 5), **Video Assembly** (Unit 6), **Orchestrator** (Unit 8, truyền preset), **Web GUI** (Unit 10, chọn preset), `docker-compose.yml`.

## Tiêu chí nghiệm thu
1. `ffprobe` bản xuất: 1920×1080, 60fps, `yuv420p`, `High` profile, audio `aac 48kHz`.
2. `-movflags +faststart` xác nhận bằng `ffprobe`/`mp4dump` (moov atom ở đầu file).
3. Video không bật phụ đề: khâu assembly KHÔNG re-encode video stream (kiểm chứng bằng thời gian chạy + `ffprobe` bitrate không đổi).
4. Upload thử lên YouTube ở chế độ private, YouTube nhận diện đúng 1080p60.

## Câu hỏi còn mở
1. Chốt FR12.4: burn phụ đề trong Manim, giữ ở ffmpeg, hay soft-sub?
2. `crf 18` hay `crf 16`? Phụ thuộc dung lượng file Creator chấp nhận được.
