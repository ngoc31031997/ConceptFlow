# Business Rules — Unit 4: Script Processing Service

## Rule 1: At Least One Scene
Script phải có ít nhất 1 heading `## Scene N` hợp lệ. Nếu không tìm thấy heading nào trong toàn bộ script → raise `ScriptSyntaxError(line_number=None, reason="no scenes found")`.
- **Rationale**: Story A2 — script rỗng/vô nghĩa không parse được thành gì.

## Rule 2: Scene Numbering Must Be Strictly Sequential Starting at 1
Số N trong heading `## Scene N` phải tăng dần liên tục bắt đầu từ 1 (1, 2, 3, ...) — không được nhảy số (vd. 1, 3) hay lặp lại (vd. 1, 1, 2). Vi phạm → raise `ScriptSyntaxError(line_number, reason="scene numbering must be sequential (expected Scene {expected}, got Scene {actual})")` tại dòng heading vi phạm.
- **Rationale**: Người dùng xác nhận rõ ràng — số nhảy cóc gây khó hiểu khi Creator đọc lại script; validate giúp phát hiện lỗi copy-paste/xóa scene sai sót sớm.
- `scene_index` (0-based, output) = N - 1 khi hợp lệ (đồng nhất với vị trí xuất hiện, vì đã validate N liên tục).

## Rule 3: `narration_text` Must Not Be Empty
Mỗi scene phải có `narration_text` không rỗng (sau khi strip whitespace). Vi phạm → `ScriptSyntaxError(line_number, reason="narration_text must not be empty")` tại heading của scene đó.
- **Rationale**: Khớp domain rule của Content Plugin Service (`narration_text` bắt buộc, Unit 2's `business-rules.md`) — nhất quán toàn hệ thống.

## Rule 4: `illustration_hint` Is Optional
Dòng blockquote `> ...` không bắt buộc phải có trong mỗi scene. Nếu thiếu, `illustration_hint = None` — KHÔNG raise lỗi.
- **Rationale**: Không phải scene nào cũng cần gợi ý minh họa riêng (Functional Design Question 2).

## Rule 5: At Most One Code Fence Per Scene
Nếu 1 scene có ≥ 2 code fence (```` ``` ````), raise `ScriptSyntaxError(line_number, reason="a scene may contain at most one code block")` tại code fence thứ 2.
- **Rationale**: `Scene.code_snippet` chỉ chứa được 1 chuỗi (không phải list) — người dùng xác nhận chọn raise lỗi rõ ràng thay vì âm thầm bỏ qua hoặc gộp code lại (Question 3).

## Rule 6: Content Before the First Heading Is Ignored
Nội dung TRƯỚC heading `## Scene 1` đầu tiên (vd. tiêu đề, ghi chú tự do) bị bỏ qua hoàn toàn — không map vào bất kỳ scene nào, không phải lỗi cú pháp (trừ khi không có heading nào — xem Rule 1).
- **Rationale**: Cho phép Creator viết ghi chú tự do ở đầu file mà không lo lỗi cú pháp (Question 4).

## Rule 7: Fail-Fast on First Syntax Error
Parser dừng ngay tại lỗi cú pháp đầu tiên gặp phải (không cố gắng thu thập toàn bộ lỗi trong 1 lần parse) — nhất quán với cách `parse_failed` chỉ mang 1 `error_message`/`line_number`/`reason` duy nhất (đã xác nhận ở Low-Level Design `interface-contracts.md`).
- **Rationale**: Đơn giản hóa cả parser lẫn contract lỗi; Creator sửa lỗi đầu tiên, chạy lại, sẽ thấy lỗi tiếp theo (nếu có) — chấp nhận được cho MVP.
