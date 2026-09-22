# CR-029 — Gộp bước validate, bỏ qc_video khỏi luồng chính, thêm % tiến trình (P1)

## Date
2026-09-22

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: Refactor luồng Render Saga (không đổi nghiệp vụ cốt lõi) + cải thiện trải nghiệm chờ
- **Scope estimate**: 4 unit — `orchestrator` (saga step map), `script-processing`/`rendering` (gộp parse+validate), `video-assembly` (bỏ gọi qc_video), `tts`/`rendering`/`video-assembly` (phát % tiến trình)
- **Complexity estimate**: Moderate. Không đổi state machine idempotency/retry (Rule 5 vẫn giữ nguyên), chỉ đổi số lượng bước và điểm phát event.

## Bối cảnh
Rà soát lại luồng 7 bước của Render Saga (xem `docs/review/data-flow-review.md`,
mục "Giai đoạn B") cho thấy:

1. `validate_script` (bước 2) không phải kiểm tra cú pháp tĩnh — nó là một lượt
   dry-run Manim thật, sinh ra `Beats[]` cần cho đồng bộ TTS/render. Về bản chất
   nó cùng nhóm với `parse_script` (bước 1): cả hai đều là "đọc và xác nhận
   script hợp lệ trước khi tốn chi phí sản xuất", chỉ khác độ sâu kiểm tra. Gộp
   hai bước lại thành một giai đoạn duy nhất giúp Creator có một điểm dừng rõ
   ràng để sửa script/prompt khi có lỗi, thay vì hai điểm dừng rời rạc.
2. `qc_video` (bước 6) chạy **sau** `assemble_video`, tức là toàn bộ chi phí
   TTS + render + ghép đã phát sinh xong. Nó cũng không có nhánh fail (chấm
   không được thì `status="not_scored"`, saga vẫn đi tiếp) — nghĩa là hiện tại
   nó không gate được gì, chỉ ghi thêm một `QCReport` tham khảo. Kiểm soát chất
   lượng đúng chỗ phải nằm **trước khi render**, không phải sau khi đã render
   xong.
3. Các bước sản xuất còn lại (`synthesize_speech`, `render_scenes`,
   `assemble_video`, `generate_clips`) chạy lâu (phút→giờ) nhưng GUI hiện chỉ
   biết trạng thái chung (`status: rendering`...), không có tín hiệu tiến trình
   cho 3/4 bước (chỉ `render_scenes` đã có `scene_rendered` qua
   `progress.fanout`). Không phân biệt được "đang chạy" với "đang treo".

## Quyết định

### 1. Gộp bước 1+2 thành một bước lớn: `parse_and_validate_script`
- Gộp logic `parse_script` (script-processing) và `validate_script` (rendering
  dry-run) dưới **một** trạng thái saga duy nhất, ví dụ `validating_script`.
  Về mặt message, hai command vẫn có thể tuần tự nội bộ (script-processing →
  rendering) nhưng saga chỉ có **một** điểm chờ/một điểm sửa lỗi hướng ra
  Creator, thay vì lộ ra 2 trạng thái trung gian không ai hành động được.
- Nếu bước gộp lỗi (parse lỗi hoặc validate lỗi), Creator sửa script/prompt và
  chạy lại từ đầu bước này (không phải huỷ cả saga).
- Cổng duyệt dàn ý (CR-024) giữ nguyên vị trí: **sau** bước gộp, **trước**
  `synthesize_speech`.

### 2. Bỏ `qc_video` khỏi luồng chính, đưa vào backlog
- Orchestrator không còn dispatch `qc_video` sau `assemble_video`.
- Saga đi thẳng từ `video_assembled` → `generate_clips` (hoặc thẳng
  `ready_to_publish` nếu `VideoOutputMode` không sinh short/clip).
- Code service video-assembly xử lý QC **không xoá** — giữ lại, tắt qua feature
  flag / không gọi, để có thể bật lại khi được thiết kế lại đúng vị trí (khả
  năng: chuyển phần kiểm `LayoutMarks` lên ngay sau `render_scenes`, trước
  `assemble_video` — ghi vào backlog, không thuộc scope CR này).

### 3. Thêm % tiến trình cho `synthesize_speech`, `assemble_video`, `generate_clips`
- Tái dùng pattern event đã có ở `render_scenes` (`scene_rendered` →
  `progress.fanout` → SSE → GUI), áp dụng tương tự cho 3 bước còn lại.
- Nguyên tắc bắt buộc: **bắn event theo đơn vị công việc hoàn thành** (xong 1
  câu TTS, xong 1 giai đoạn ghép ffmpeg — mux audio/sub/intro, xong 1 clip),
  **không** bắn theo tick thời gian/frame. Mục tiêu là vài chục event/saga,
  không tạo tải thêm lên RabbitMQ/SSE.
- GUI chỉ cần hiển thị % hoàn thành, không cần chi tiết sub-step.

## Không nằm trong scope CR này
- Thiết kế lại vị trí đúng cho QC (đưa `LayoutMarks` check lên trước
  `assemble_video`) — ghi backlog, làm CR riêng sau.
- Đổi cơ chế retry/idempotency của saga (Rule 5) — giữ nguyên.

## Ảnh hưởng tài liệu
- `docs/review/data-flow-review.md` — cập nhật lại sơ đồ "Giai đoạn B" từ 7
  bước xuống còn 5 bước hiệu lực (1 gộp, 1 backlog).
- `aidlc-docs/aidlc-state.md` — thêm CR-029 vào bảng Change Requests.
