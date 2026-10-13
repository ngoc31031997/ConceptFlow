# ADR-0038: Render theo cảnh, mỗi cảnh một lệnh, có đoạn chồng ở chỗ nối liền mạch

## Status
Accepted

## Date
2026-10-13

## Stage
Application Design (CR-072)

## Context
Rendering xuất một `rendered.mp4` cho cả video; sửa một chỗ phải render lại cả video, và một lỗi làm hỏng cả bước. Creator muốn render và dựng lại từng cảnh (câu 11 B), máy quay liền mạch giữa các cảnh (câu 22 B), một cảnh lỗi không dừng cảnh khác và lỗi tạm thời tự chạy lại một lần (P-9 A). AD-8 hỏi ai điều phối.

## Options Considered
### Option A: orchestrator gửi một lệnh mỗi cảnh, giữ trạng thái từng cảnh
- What it is: `render_scene` cho mỗi cảnh; rendering xử lý song song theo số worker; orchestrator chạy lại, huỷ, tự thử lại theo cảnh.
- Strengths: thử lại và huỷ là việc của saga, đúng chỗ đang làm; tiến độ theo cảnh rõ.
- Trade-offs: saga render đổi nhiều.

### Option B: một lệnh cả video, rendering tự chia cảnh
- What it is: như hiện nay, rendering báo tiến độ từng cảnh.
- Strengths: saga ít đổi.
- Trade-offs: chạy lại một cảnh phải gửi lại lệnh cả video; orchestrator không thấy trạng thái từng cảnh.

## Decision
A (AD-8 A). Cảnh có chỗ nối liền mạch được render thêm một đoạn chồng ở đầu hoặc cuối; video-assembly cắt đoạn chồng khi ghép. Lệnh `render_scenes` cũ giữ tới khi saga mới chạy ổn, rồi bỏ (ghi vào backlog).

## Rationale
Chạy lại, huỷ và tự thử lại theo cảnh là các trạng thái của saga; đặt chúng ở orchestrator giữ đúng vai trò hiện có, và rendering vẫn không phải nhớ trạng thái giữa các lệnh.

## Consequences
- **Positive**: sửa một cảnh chỉ render lại cảnh đó và đoạn nối liền kề; render song song; lỗi khoanh vùng theo cảnh.
- **Negative / Accepted Trade-offs**: hợp đồng file `/shared/{id}/video/scenes/` mới; video-assembly phải ghép khớp đoạn chồng; cần kiểm QC chỗ nối (FR-26).
