# CR-007 — Xuất bản dọc 9:16 cho YouTube Shorts / TikTok (P1 cho mục tiêu đa nền tảng)

## Date
2026-09-07

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
Mục tiêu của Creator là đăng cả **YouTube và TikTok**. Pipeline hiện tại **không hỗ trợ TikTok ở bất kỳ khâu nào**.

## Vấn đề (đã xác minh trong code)

1. **Chỉ có khung 16:9** — Manim mặc định 16:9; `subtitle_file.py` cố định `PLAY_RES_X=1920 / PLAY_RES_Y=1080`.
2. **Chỉ có publisher YouTube** — `services/publisher/adapters/youtube/youtube_publisher.py` là adapter duy nhất; `domain/ports.py::VideoPublisherPort` chỉ được implement cho YouTube.
3. **Sai định dạng nội dung** — video 5–10 phút không hợp TikTok (tối ưu cho < 3 phút). Đăng nguyên bản video dài lên TikTok gần như chắc chắn không hiệu quả.

## Chiến lược đề xuất
Không cố sản xuất riêng video dọc dài. Thay vào đó: **sinh clip phái sinh (derivative clips)** từ video 16:9 đã render — 2–3 clip dọc 30–60s làm Shorts/TikTok để kéo traffic về video dài. Đây là mô hình phân phối chuẩn của kênh giáo dục.

## Functional Requirements

### FR19 — Clip dọc phái sinh (mới)
- **FR19.1**: Sau khi có video hoàn chỉnh, hệ thống PHẢI sinh được N clip dọc 1080×1920 từ các đoạn do Creator chọn.
- **FR19.2**: Creator chọn đoạn bằng marker trong script (vd. `# CLIP: "tên clip"` … `# ENDCLIP`) hoặc bằng khoảng thời gian nhập ở GUI.
- **FR19.3**: Chuyển 16:9 → 9:16 PHẢI dùng bố cục **blur background + video gốc căn giữa** (không crop mất nội dung — Manim thường có công thức/chữ ở rìa khung).
- **FR19.4**: Phụ đề trên clip dọc PHẢI to hơn hẳn, đặt giữa khung (chuẩn Shorts/TikTok), tự lấy PlayRes từ khung dọc (liên quan CR-004 FR12.5).
- **FR19.5**: Clip dọc PHẢI có **2 preset độ dài** (đã chốt với Creator 2026-09-07), Creator chọn khi sinh clip:
  - **Preset `short`**: ≤ 60s — đủ điều kiện YouTube Shorts.
  - **Preset `long`**: 60–180s — đủ điều kiện TikTok Creator Rewards (yêu cầu video > 1 phút).
- **FR19.6 (mới)**: Hệ thống PHẢI **validate độ dài theo preset** trước khi xuất, và báo lỗi rõ ràng nếu đoạn Creator chọn không nằm trong khoảng của preset (vd. chọn `short` nhưng đoạn dài 75s) — không im lặng cắt cụt.
- **FR19.7 (mới)**: Một đoạn PHẢI xuất được ra **cả hai preset** trong cùng một lần chạy nếu Creator muốn (cùng nội dung, hai file), tránh phải chọn đoạn hai lần.

### FR20 — Đăng đa nền tảng (mới)
- **FR20.1**: Clip dọc PHẢI tải về được từ GUI (đăng TikTok thủ công là chấp nhận được cho MVP).
- **FR20.2 (tuỳ chọn)**: Adapter TikTok Content Posting API qua `VideoPublisherPort` — cần đăng ký TikTok Developer app và duyệt.

## Ràng buộc
- **C1**: TikTok Content Posting API yêu cầu app được TikTok duyệt; với kênh cá nhân, quy trình duyệt là rào cản thật ⇒ FR20.2 để pha sau, MVP dùng FR20.1 (tải về + đăng tay).
- **C2 — ĐÃ CHỐT (2026-09-07)**: chính sách TikTok Creator Rewards yêu cầu video **> 1 phút**, mâu thuẫn với giới hạn ≤60s của Shorts ⇒ **làm cả hai preset** (FR19.5), không ưu tiên một nền tảng. Chi phí thêm chủ yếu là thời gian encode, không phải độ phức tạp kiến trúc — cùng một filtergraph, khác khoảng cắt.
- **C2b**: Ngưỡng độ dài của cả hai nền tảng do bên thứ ba đặt ra và **thay đổi theo thời gian** (Shorts từng là 15s rồi 60s; ngưỡng TikTok cũng đã đổi). Do đó giới hạn preset PHẢI là **config, không hardcode**.
- **C3**: Phụ thuộc **CR-002** (offset) để cắt đúng đoạn theo mốc narration.
- **C4**: Cắt clip là re-encode ⇒ phụ thuộc **CR-003** (timeout) và **CR-004** (profile encode).

## Phạm vi tác động
- **Video Assembly** (Unit 6): bước sinh clip dọc (filter `scale`+`boxblur`+`overlay`).
- **Script Processing** (Unit 4): parse marker `# CLIP:`.
- **Orchestrator** (Unit 8): bước saga mới `generate_clips` (hoặc mở rộng `assemble_video`).
- **Web GUI** (Unit 10): chọn đoạn, preview, tải clip.
- **Publisher** (Unit 7): chỉ khi làm FR20.2.

## Tiêu chí nghiệm thu
1. Từ 1 video 8 phút sinh ra clip 1080×1920 ở **cả hai preset**: một clip ≤60s và một clip 60–180s.
2. Upload thử clip preset `short` (private) — YouTube nhận diện là Shorts.
3. Clip preset `long` dài hơn 60s, phù hợp yêu cầu TikTok Creator Rewards.
4. Phụ đề đọc rõ trên màn hình điện thoại; nội dung Manim ở rìa khung không bị cắt mất.
5. Chọn đoạn sai độ dài so với preset ⇒ báo lỗi rõ ràng, không cắt cụt im lặng.

## Câu hỏi còn mở
1. Có làm adapter TikTok API (FR20.2) không, hay đăng tay là đủ? (Đề xuất: đăng tay cho MVP — quy trình duyệt app của TikTok là rào cản thật với kênh cá nhân, xem §C1.)
2. Ngưỡng cụ thể của preset `long` (60–180s hay hẹp hơn)? Quyết ở Low-Level Design.
