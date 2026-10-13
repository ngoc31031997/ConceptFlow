# ADR-0037: Phong cách theo thế giới của chủ đề, thay nền sáng bão hoà mặc định của CR-067

## Status
Accepted

## Date
2026-10-13

## Stage
Application Design (CR-072)

## Context
FR7 của CR-067 đặt nền sáng bão hoà làm mặc định cho cảnh giải thích ý, nền tối phải có lý do. Creator muốn phong cách quyết định theo chủ đề (vũ trụ thì nền tối, lòng quả sung thì cảm giác bên trong quả sung) (câu 19, C3), và muốn cảnh giải thích ý cũng nằm trong thế giới đó (C4).

## Options Considered
### Option A: giữ nền sáng bão hoà mặc định
- What it is: như CR-067.
- Strengths: không đổi luật.
- Trade-offs: không theo được chủ đề.

### Option B: thế giới theo chủ đề, giữ ngôn ngữ hình chung, thêm lớp chất liệu theo chủ đề
- What it is: mỗi video có thế giới (bối cảnh, bảng màu, ánh sáng, chất liệu) theo chủ đề; cách dựng khối và cách vẽ nhân vật theo ngôn ngữ hình đang bật; cảnh giải thích ý là vật trong thế giới.
- Strengths: mỗi chủ đề có cảm giác riêng mà vẫn là một kênh; phim liền mạch, không "nhảy ra slide".
- Trade-offs: Đạo diễn AI phải nghĩ cách đặt ý vào cảnh; khó hơn nền màu đặc.

### Option C: tự do hoàn toàn theo từng video
- What it is: mỗi video một kiểu vẽ riêng.
- Strengths: sáng tạo nhất.
- Trade-offs: không có chữ ký kênh; thư viện hình khó dùng lại.

## Decision
B (C3 B, C4 A). FR7 "nền sáng bão hoà mặc định" của CR-067 bị thay.

## Rationale
Creator muốn theo chủ đề nhưng vẫn là một kênh chuyên nghiệp. Ngôn ngữ hình chung (ADR-0036) giữ chữ ký kênh; thế giới và hành trình màu (FR-6) đem lại khác biệt giữa các video và giữa các phần trong một video.

## Consequences
- **Positive**: video có thế giới riêng; cảnh giải thích ý liền mạch với phần kể chuyện, hợp với máy quay liền mạch (FR-8).
- **Negative / Accepted Trade-offs**: prompt Đạo diễn và Kỹ sư khó hơn; cần kiểm tương phản theo từng thế giới thay cho bộ nền cố định.
