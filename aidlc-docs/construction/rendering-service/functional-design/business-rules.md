# Business Rules — Unit 5: Rendering Service

## Rule 1: Zero-Trust Input Validation
Rendering Service KHÔNG tin tưởng dữ liệu đến từ Orchestrator, dù đã qua validate ở các bước trước (Script Processing, Content Plugin, TTS). Validate đầy đủ trước khi render:
- `project_id` không rỗng
- `scene_index` không âm
- `narration_text` không rỗng
- `animation_template_id` phải có trong `AnimationTemplateRegistry` → vi phạm: `UnsupportedTemplateError`
- `audio_path` không rỗng
- `duration_seconds` phải > 0 → vi phạm: `InvalidDurationError`

Vi phạm bất kỳ điều nào → raise lỗi tương ứng ngay, không render scene đó.
- **Rationale**: Người dùng xác nhận nguyên tắc zero-trust giữa các service — mỗi service tự bảo vệ mình khỏi dữ liệu sai lệch, không phụ thuộc hoàn toàn vào validation của service khác (phòng trường hợp Orchestrator có lỗi gộp dữ liệu, hoặc 1 unit khác có bug).

## Rule 2: Animation-Audio Duration Matching (FR4.3)
Mỗi Manim Scene tự set `run_time` để khớp `duration_seconds` (audio) với sai lệch cho phép ±0.5s.
- Nếu animation "tự nhiên" (nội dung cố định) NGẮN hơn audio: thêm `self.wait()` ở cuối để kéo dài khớp đúng `duration_seconds`.
- Nếu animation "tự nhiên" DÀI hơn audio: KHÔNG cắt animation dở dang — chấp nhận animation dài hơn audio một chút. `SceneRenderResult.duration_seconds` phản ánh thời lượng THỰC TẾ (có thể > input `duration_seconds`).
- **Rationale**: Ưu tiên toàn vẹn nội dung minh họa hơn đồng bộ tuyệt đối — cắt animation giữa chừng sẽ phá hỏng nội dung giáo dục. Video Assembly Service (Unit 6) xử lý chênh lệch nhỏ này khi ghép audio+video.

## Rule 3: Code Snippet Display Placement (Story B3)
Nếu scene có `code_snippet`: hiển thị NGAY TỪ ĐẦU scene, giữ nguyên trên màn hình XUYÊN SUỐT scene, đặt cố định ở 1 góc màn hình (bên trái); phần animation minh họa (thuật toán/khái niệm) chiếm phần còn lại. Nếu KHÔNG có `code_snippet`: animation dùng toàn bộ màn hình.
- **Rationale**: Người xem cần đối chiếu code liên tục trong lúc nghe giải thích (Story B3's mục đích "dễ theo dõi mã nguồn đang minh họa").

## Rule 4: Unknown/Missing `code_language` Fallback
Nếu `code_language` là `None` hoặc không phải ngôn ngữ Pygments nhận diện được: hiển thị `code_snippet` KHÔNG tô màu syntax (dùng Pygments `TextLexer`), KHÔNG raise lỗi, KHÔNG chặn render. Log warning.
- **Rationale**: Một lỗi nhỏ về khai báo ngôn ngữ (thiếu/sai chính tả) không nên chặn toàn bộ scene — vẫn hiển thị được nội dung code (dù kém thẩm mỹ hơn) tốt hơn là render thất bại hoàn toàn.

## Rule 5: Idempotency (Artifact-Level, không đổi từ Low-Level Design)
Kiểm tra file `.mp4` đã tồn tại tại `/shared/{project_id}/animations/{scene_index}.mp4` trước khi render — nếu có, trả `SceneRenderResult` từ file có sẵn (đọc duration thực tế), không render lại.

## Error Classification Summary
| Error | Retry-able? |
|---|---|
| `UnsupportedTemplateError` | No (permanent — Orchestrator/dữ liệu upstream có bug, cần sửa) |
| `InvalidDurationError` | No (permanent) |
| `AnimationEngineError` (Manim crash/timeout) | Yes (transient) |
