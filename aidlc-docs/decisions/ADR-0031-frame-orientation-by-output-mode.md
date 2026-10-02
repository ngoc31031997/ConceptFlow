# ADR-0031: Khung hình theo chế độ đầu ra; short dựng dọc riêng từ đầu

## Status
Accepted

## Date
2026-10-01

## Stage
Application Design (CR-060)

## Context
Short hiện được cắt từ video 16:9 đã ghép rồi đặt giữa một nền mờ phóng to (`video-assembly/adapters/clips/vertical_clip.py`). Kết quả là một khung ngang nhỏ giữa màn hình dọc, khác hẳn short tham chiếu: dựng thẳng khung 9:16, kịch bản riêng 30–60 giây, chữ từ khoá động thay phụ đề. Khung 1920×1080 được viết cứng ở prompt Đạo diễn/Kỹ sư, llm-service (storyboard, merger) và rendering (luật bố cục, layout probe, kit).

Creator chọn: short dựng dọc riêng với kịch bản riêng (CR-060, câu 3a).

## Các phương án
- **A. Hướng khung là thuộc tính của format.** Thêm cột `orientation` vào `video_formats` (orchestrator) và hai bản domain format. Orchestrator phải đọc format để biết bỏ intro/outro.
- **B. Hướng khung theo chế độ đầu ra (chọn).** `video_output_mode = "short"` nghĩa là dựng 1080×1920 từ đầu; mọi chế độ khác là 1920×1080. Format chỉ được lọc theo độ dài (`max_seconds ≤ 90` là format short).
- **C. Dựng lại các shot của video dài theo khung dọc.** Giữ một kịch bản cho cả hai, nhưng một shot bố cục cho 16:9 không chuyển sang 9:16 mà vẫn đẹp, và short tham chiếu có kịch bản riêng.

## Decision
B.
- Một khái niệm `Frame` (kích thước + vùng an toàn) có hai giá trị, giữ ở bốn nơi có test đối chiếu chéo với `conceptflow-mini/primitives.tsx` (`FRAMES`): authoring-service `domain/frame.go`, llm-service `app/frame.py`, rendering `domain/frame.py`. Vùng an toàn dọc chừa 200 px trên, 360 px dưới, 140 px phải cho giao diện Shorts.
- authoring-service chọn khung từ chế độ đầu ra (`domain.FrameFor`), điền biến prompt `{{frame}}`, `{{frame_width}}`, `{{frame_height}}`, `{{safe_area}}`, `{{frame_rules}}`, và gửi `frame` cho llm-service (`/v1/storyboard/finalize`, `/v2/code/*`). llm-service viết `<Composition>` đúng kích thước; rendering đọc kích thước từ chính `<Composition>` khi kiểm bố cục và khi render.
- Short dùng format `vertical_short_60s`, engine Remotion (Manim không có khung dọc), phụ đề tắt. Khi Creator chọn short ở bước 2, orchestrator tự điền ba thứ đó; khi bấm "Tiếp tục" nó từ chối tổ hợp không dựng được (short với format dài hay Manim, video dài với format short).
- Short không ghép intro/outro 16:9 của kênh và không chạy `generate_clips`.
- "Làm bản short dọc" từ màn Kết quả của video dài tạo project nháp mới với `short_of`, nối hai project qua `companion_project_id` cả hai chiều.

## Consequences
- Khung dọc thêm kích thước khung vào vân tay đoạn khung ở bước Code; khung ngang giữ nguyên vân tay, nên code đã lưu của các dự án đang làm không bị coi là cũ.
- Chế độ `both` (video dài kèm clip cắt) không còn trong bộ chọn; đường cắt clip (bước 12, `generate_clips`, `ClipsPanel`, `self.clip`, prompt `short_script`) còn trong code nhưng không lựa chọn mới nào dẫn tới, và được gỡ ở CR-061. Đã gỡ ở CR-061: `video_output_mode` chỉ còn `long` | `short`, luồng còn 13 bước (Kết quả = 12, Đăng video = 13).
- Thẻ thương hiệu cuối short và intro/outro dọc của kênh chưa có; cần tài sản kênh dọc.
