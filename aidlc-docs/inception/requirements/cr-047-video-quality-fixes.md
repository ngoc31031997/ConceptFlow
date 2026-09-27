# CR-047 — Sửa 3 vấn đề hệ thống phát hiện qua đánh giá video (prompt, khoảng nghỉ, bố cục)

## Date
2026-09-27

## Stage
Requirements Analysis + Construction — thực hiện trong cùng phiên theo yêu cầu trực tiếp của Creator ("implement phần hệ thống: 1,3,4"), dựa trên một đánh giá video cụ thể (project `a070963d-0ae7-4825-a8b9-4007ef84275a`, chủ đề "Sâu răng thực sự là gì"). **Chưa duyệt merge vào `main`** — chờ Creator xem kết quả rebuild.

## Bối cảnh
Xem video kết quả của project trên cho thấy 3 nhóm lỗi hệ thống (không phải lỗi riêng của project đó):

1. **Prompt Story Architect (kiểu NGHỊCH-LÝ)** sinh lời thoại đọc thành tiếng nguyên văn nhãn cấu trúc nội bộ ("Phần bị che khuất: ...") 4/6 lần, lặp lại gần y hệt kết luận đã nói ở beat lõi (pattern) sang các beat ví dụ (variation), và ranh giới chọn kiểu NGHỊCH-LÝ so với DẤU-VẾT không đủ rõ nên một chủ đề dạng cơ chế ("X thực sự là gì") bị ép vào khuôn "nhiều lĩnh vực" trong khi mọi ví dụ đều cùng một đối tượng (cái răng) chỉ đổi nhãn lĩnh vực.
2. **Rendering (Remotion)**: các shot được ghép sát nhau tuyệt đối (`_segments_from` đặt `startFrame` của shot sau ngay tại điểm kết thúc shot trước, gap = 0), nên video đọc liên tục không có khoảng thở nào giữa hai câu thoại — kể cả giữa hai beat.
3. **Storyboard / Remotion Engineer**: không có ràng buộc định lượng nào cho kích thước tối thiểu của vật chính trong khung (luật F chỉ giới hạn vùng an toàn — biên TỐI ĐA — không có sàn TỐI THIỂU), và không có trần số shot mỗi cảnh, nên một cảnh dài (ví dụ beat "pattern" có ngân sách tới 120 giây) có thể bị chẻ thành rất nhiều shot ngắn, tăng bề mặt lỗi hiển thị (khung gần trống, vật quá nhỏ) khi model sinh code dưới áp lực (nhiều lượt lỗi/đổi model như log `project_errors` của project trên cho thấy).

Đây là 3/6 khuyến nghị hệ thống trong đánh giá; Creator chọn làm mục 1, 3, 4 (đánh số theo đánh giá gốc) = prompt Story, khoảng nghỉ khi ghép, ràng buộc bố cục storyboard. Các mục còn lại (kiểm thời lượng tối thiểu cho Remotion, QC hình tự động, cảnh báo intro/outro thiếu asset, lệch 1080p60↔30fps) NGOÀI phạm vi CR này.

## Quyết định
Không có phiên hỏi-đáp Creator riêng cho CR này — 3 hướng sửa dưới đây là cách trực tiếp nhất để đóng đúng 3 lỗi đã quan sát được, tối thiểu hoá rủi ro (chỉ sửa prompt text + một hàm thuần tính toán, không đổi contract giữa các service).

1. **Prompt Story Architect**: thêm quy tắc cấm đọc thành lời các nhãn cấu trúc, cấm lặp kết luận giữa pattern và variation, siết lại "khuôn ví dụ" thành hướng dẫn NỘI DUNG chứ không phải cụm từ chép nguyên; siết mô tả `when_to_use` của NGHỊCH-LÝ (loại trừ chủ đề "cơ chế/quy trình đơn nhất") và DẤU-VẾT (nhận rõ chủ đề "X thực sự là gì / hoạt động thế nào" dù chỉ có một thế giới xuyên suốt).
2. **Rendering (remotion_renderer.py)**: chèn một khoảng lặng cố định giữa mọi shot liên tiếp khi tính `segments`/`wait_offsets`, để cả video (qua `<Segments>`) và audio (qua `adelay` ở `video-assembly`, vốn đọc đúng `wait_offsets` này) cùng giãn ra theo cùng một điểm neo — không cần đổi contract giữa rendering và video-assembly.
   - Không làm khoảng nghỉ lớn hơn ở ranh giới beat trong CR này: `NarrationSegment` hiện không mang `beat_id` (chỉ có `scene_index` phẳng); thêm trường đó là việc xuyên nhiều service (orchestrator → message schema → llm-service → rendering), coi là theo dõi riêng, ghi lại ở mục "Nợ kỹ thuật/Backlog" bên dưới.
3. **Storyboard (Visual Director) + Remotion Engineer**: thêm luật kích thước tối thiểu cho vật đang là trọng tâm của shot (luật F mới — F14), thêm trần mềm số shot một cảnh với hướng dẫn gộp nhiều câu thoại ngắn cùng một thay đổi hình vào một shot thay vì 1 câu = 1 shot mặc định, và thêm mục tự kiểm tương ứng ở cả hai đầu (Visual Director + Remotion Engineer, cả bản thủ công lẫn AI flow).

## Yêu cầu chức năng
- **FR1** — `storyArchitectVI` (prompt_template_seeds.go): quy tắc lời thoại không được đọc thành lời các cụm nhãn cấu trúc kiểu "Phần bị che khuất:"; mỗi beat variation phải có câu mở khác nhau; cấm lặp lại nguyên câu/kết luận đã nói ở beat pattern. Cập nhật mục TRÁNH TUYỆT ĐỐI và TỰ KIỂM tương ứng.
- **FR2** — `SystemVideoArchetypes()` (video_archetype.go): sửa `WhenToUse` của NGHỊCH-LÝ (loại trừ rõ chủ đề cơ chế/quy trình đơn nhất) và DẤU-VẾT (nhận rõ "X là gì/hoạt động thế nào" dù một thế giới xuyên suốt, không cần nhiều lĩnh vực). Đây là system row, seeder upsert lại khi authoring-service khởi động lại (ghi đè theo `ID` cố định).
- **FR3** — `remotion_renderer.py::_segments_from`: thêm hằng `INTER_SHOT_GAP_SECONDS` (mặc định 0.3s = hằng `FPS`), chèn số frame tương ứng vào `frame_cursor` giữa các shot liên tiếp (không chèn sau shot cuối). `wait_offsets` tiếp tục tính từ `segment["startFrame"] / FPS` như cũ nên tự động phản ánh khoảng nghỉ mới.
- **FR4** — Luật bố cục (`remoFVI`, dùng chung cho cả bản thủ công và AI flow): thêm **L14 — Kích thước tối thiểu**, quy định vật đang là trọng tâm của một shot (vật gắn với LAYOUT chính, hoặc vật lớn nhất khung) phải cao/rộng tối thiểu khoảng 30% chiều tương ứng của khung ở cỡ trung/cận cảnh (không áp cho vật phụ, vật nền, hay toàn cảnh có nhiều vật nhỏ theo đúng ý kịch bản).
- **FR5** — Luật đạo diễn (`visualDirectorHeadVI`, dùng chung cho cả hai flow): thêm quy tắc trần mềm shot mỗi cảnh — một cảnh không nên vượt quá khoảng 8 shot; cảnh có ngân sách lời thoại dài thì gộp nhiều câu ngắn liền ý vào cùng một shot (giữ nguyên "hình luôn sống" bằng cách cho nhiều thay đổi nhỏ trong cùng một shot) thay vì tách shot theo từng câu.
- **FR6** — Cập nhật mục tự kiểm tương ứng ở cả 4 nơi bị ảnh hưởng: `storyArchitectVI`, `visualDirectorTailVI`, `visualDirectorTailAIVI`, `remoGVI`, `remoGAIVI`.

## Phạm vi kỹ thuật
- `authoring-service/internal/domain/prompt_template_seeds.go` — sửa `storyArchitectVI`, `visualDirectorHeadVI`, `remoFVI`, `remoGVI`.
- `authoring-service/internal/domain/prompt_template_seeds_ai.go` — sửa `visualDirectorTailAIVI`, `remoGAIVI` (đối chiếu để không lệch với bản thủ công).
- `authoring-service/internal/domain/video_archetype.go` — sửa `WhenToUse` của NGHỊCH-LÝ, DẤU-VẾT trong `SystemVideoArchetypes()`.
- `services/rendering/adapters/rendering/remotion_renderer.py` — thêm hằng gap, sửa `_segments_from`.
- Không đổi schema DB, không đổi contract message giữa service, không đổi API công khai.

## Nợ kỹ thuật / Backlog (ghi nhận, không làm trong CR này)
- Khoảng nghỉ lớn hơn ở ranh giới BEAT (so với ranh giới shot thường) cần `NarrationSegment` mang thêm `beat_id`, xuyên orchestrator (payload saga) → llm-service (đã biết `Storyboard.scenes[].id` nhưng không truyền xuống) → rendering. Để riêng vì đụng message schema.
- Kiểm thời lượng tối thiểu/tối đa theo `video_formats` cũng cần áp cho engine Remotion (hiện `validation_warnings` chỉ cảnh báo thiếu `self.beat(...)` — quy ước của Manim — trên cả project dùng Remotion).
- QC hình tự động (lấy mẫu frame phát hiện khung gần trống/mảng màu phẳng lớn) — `qc_reports` rỗng cho project được xem xét dù `qc_video` vẫn còn trong code (tắt khỏi luồng chính từ CR-029 theo `aidlc-state.md`).
- Cảnh báo khi bật intro/outro mà không chọn asset; và lệch cấu hình 1080p60 ↔ video xuất ra chỉ 30fps.

## Kiểm thử
- `authoring-service`: `go test ./...` (đặc biệt `internal/domain` — golden prompt tests, `internal/application` — `prompts_test.go`, `render_prompt_test.go`, `prompt_golden_test.go` — các test này thường snapshot nguyên văn prompt nên cần cập nhật cùng lúc).
- `rendering`: `pytest tests/adapters/test_remotion_renderer.py` (cập nhật kỳ vọng `segments`/`wait_offsets` có gap) + chạy lại toàn bộ suite rendering để không phạm chỗ khác dùng `_segments_from`.
- Rebuild + restart `authoring-service` và `rendering`, xác nhận healthy (theo chính sách rebuild của CLAUDE.md).
- Không re-render lại video project cũ trong CR này (chỉ đổi hệ thống cho lần chạy mới); Creator có thể tự tạo project mới hoặc chạy lại bước Story/Code để thấy hiệu quả.
