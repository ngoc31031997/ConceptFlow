# ADR-0032: Câu thoại là đơn vị thời gian của shot Remotion

## Status
Accepted

## Date
2026-10-03

## Stage
Application Design (CR-067)

## Context
Video tham khảo đổi hình theo từng câu thoại (2–4 s một lần) và hình bám đúng lúc từ được đọc. Hệ thống chỉ biết thời lượng của cả shot (~10 s, 2–3 câu): TTS trả một `duration_seconds` cho mỗi dòng thoại, và mỗi shot là một dòng, nên code của shot chỉ hẹn giờ được theo tỉ lệ `duration`. Đo trên hai video gần nhất: hình đứng yên ≥ 2 s chiếm 18–19% thời lượng, mẫu dài 6%.

Đường đi sẵn có của lời thoại: script xuất `narrations` → rendering đọc ở lượt chạy thử → orchestrator gửi mỗi dòng cho TTS như một `scene_index` → TTS trả file và độ dài từng dòng → rendering dựng `segments` và trả `wait_offsets` mỗi dòng một mốc → orchestrator dựng `narration_segments` và `subtitle_cues` từ chính các mốc đó.

Creator chọn: TTS đọc từng câu thành một file (CR-067, Q2-a).

## Các phương án
- **A. Mỗi câu thoại là một dòng `narrations`; script khai `shotLineCounts` (chọn).** TTS, orchestrator, video-assembly và mọi message RabbitMQ không đổi: số "scene" của orchestrator giờ là số câu. Rendering gom câu thành đoạn của shot và truyền `lines` (frame bắt đầu từng câu) vào shot.
- **B. TTS trả mốc từng câu trong một file.** Một file tiếng mỗi shot, nhưng đổi hợp đồng TTS ↔ orchestrator ↔ rendering, và mốc từ (word boundary) chỉ Edge/Azure có.

## Decision
A.
- Storyboard: shot có `lines: [{say, show}]` (1–4 câu). llm-service ghép `narration` = các `say` nối bằng dấu cách, nên mọi chỗ đọc `narration` (kiểm ngân sách lời thoại, kiểm minh hoạ của authoring-service, Manim) không đổi.
- Merger: `narrations` phẳng theo câu, `export const shotLineCounts`, `ShotProps = {duration, lines}`.
- Rendering (`remotion_renderer._segments_from`): câu của một shot nối liền, cách 0,1 s; giữa hai shot vẫn 0,3 s; `wait_offsets` một mốc cho mỗi câu; `shotLineCounts` không cộng đúng số dòng là lỗi rõ ràng. Script không có `shotLineCounts` render y như trước.
- Kit `segments.tsx`: `Segment.lines`, `evenLines`, `lineSpan`.
- Phụ đề tự thành một cue cho mỗi câu. Short dọc mặc định đốt phụ đề ở trên khung, dưới vùng nút của Shorts (thay cho "phụ đề tắt" của ADR-0031).

## Consequences
- Số lượt TTS tăng khoảng ba lần; Azure/Google tính theo ký tự nên chi phí gần như không đổi.
- Thay đổi chỉ áp cho luồng AI của Remotion. Luồng prompt thủ công và Manim giữ một dòng mỗi shot.
- Vân tay đoạn shot đổi khi shot có `lines` (JSON shot khác) và khi prompt `remotion_engineer_ai` đổi; code đã lưu của project cũ vẫn render như cũ vì không có `shotLineCounts`.
