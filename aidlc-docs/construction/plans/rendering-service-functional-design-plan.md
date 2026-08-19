# Functional Design Plan — Unit 5: Rendering Service

## Unit Context
- **Stories**: B3 (code syntax highlight), C1 (khởi chạy render — ngoài phạm vi unit), C3 (đồng bộ animation với audio, FR4.3)
- **Scope**: Render animation cho 1 scene (FR3.1, FR3.2) + đồng bộ thời lượng với audio (FR4.3). KHÔNG chịu trách nhiệm ghép audio vào video (đó là FR5.1, Video Assembly Service, Unit 6) — Rendering Service chỉ sinh animation clip CÂM (không có âm thanh), thời lượng khớp với audio để Video Assembly ghép sau.

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `business-logic-model.md`
- [ ] Tạo `business-rules.md`
- [ ] Tạo `domain-entities.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Business Rule — Validation Input
`SceneRenderRequest` nhận dữ liệu đã qua validate ở các bước trước (Script Processing, Content Plugin, TTS). Rendering Service cần validate lại gì?

A) 💡 Suggested: Chỉ validate 2 điều: (1) `animation_template_id` phải có trong `AnimationTemplateRegistry` — không có → `UnsupportedTemplateError`; (2) `duration_seconds` phải > 0 — không hợp lệ (≤0) → lỗi rõ ràng (`InvalidDurationError`). KHÔNG validate lại `narration_text`/`code_snippet` (đã validate ở Script Processing Service, tin tưởng dữ liệu từ Orchestrator)
   - ✅ Strengths: tránh validate trùng lặp giữa các unit, đúng ranh giới trách nhiệm (mỗi unit chỉ validate phần dữ liệu unit đó thực sự dùng để quyết định logic riêng)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: B — Zero trust: validate TOÀN BỘ input dù đã qua các bước trước, không tin tưởng dữ liệu từ Orchestrator. Cụ thể: `narration_text` không rỗng, `animation_template_id` có trong registry, `duration_seconds` > 0, `audio_path` không rỗng, `project_id`/`scene_index` hợp lệ (không rỗng/âm). Vi phạm bất kỳ điều nào → lỗi rõ ràng, không render.

### Question 2: Business Logic — Animation-Audio Duration Matching (FR4.3)
LLD đã xác định mỗi template tự set `run_time` khớp `duration_seconds`. Cần business rule cụ thể hơn: sai lệch cho phép là bao nhiêu, và nếu animation "tự nhiên" (nội dung cố định, không co giãn được) ngắn/dài hơn audio thì xử lý sao?

A) 💡 Suggested: Sai lệch cho phép ±0.5s (khớp FR4.3 "không quá ngưỡng cho phép được cấu hình" ở Story C3's AC — 0.5s là giá trị mặc định hợp lý cho video giáo dục). Nếu animation "tự nhiên" ngắn hơn audio: thêm `self.wait()` (khoảng lặng hình ảnh) ở cuối để kéo dài khớp đúng. Nếu animation tự nhiên DÀI hơn audio: chấp nhận animation dài hơn audio (KHÔNG cắt animation dở dang — cắt animation giữa chừng sẽ hỏng nội dung minh họa) — trường hợp này Video Assembly Service (Unit 6) sẽ xử lý (video dài hơn audio một chút, audio kết thúc trước, đây là quyết định business hợp lý hơn là cắt hình)
   - ✅ Strengths: ưu tiên không phá vỡ tính toàn vẹn nội dung animation (quan trọng hơn đồng bộ tuyệt đối), đơn giản để implement (`self.wait()` cho trường hợp phổ biến)
   - ⚠️ Trade-offs: nếu animation dài hơn audio đáng kể, video có thể có đoạn "im lặng nhìn animation" ở cuối — chấp nhận được, hiếm gặp nếu animation được thiết kế tốt theo tỷ lệ

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Business Rule — Code Snippet Display (Story B3)
Khi scene có `code_snippet`+`code_language`, animation cần hiển thị code với syntax highlight. Cần quy tắc: code hiển thị ở đâu trong timeline (đầu/cuối/xuyên suốt scene)?

A) 💡 Suggested: Code hiển thị NGAY TỪ ĐẦU scene (xuất hiện cùng lúc animation bắt đầu) và giữ nguyên trên màn hình XUYÊN SUỐT scene (không biến mất giữa chừng) — đặt cố định ở 1 góc màn hình (vd. bên trái), phần animation minh họa (thuật toán/khái niệm) diễn ra ở phần còn lại. Nếu KHÔNG có `code_snippet`, animation dùng toàn bộ màn hình
   - ✅ Strengths: đơn giản, code luôn hiển thị để người xem đối chiếu trong suốt phần giải thích (đúng tinh thần Story B3 "để người xem dễ theo dõi mã nguồn đang minh họa")
   - ⚠️ Trade-offs: layout cố định có thể không tối ưu cho code dài — chấp nhận được ở MVP, có thể tinh chỉnh sau

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Business Rule — Unknown `code_language`
Nếu `code_language` không phải ngôn ngữ Pygments hỗ trợ (hoặc `None` dù có `code_snippet`), xử lý sao?

A) 💡 Suggested: Dùng Pygments' lexer mặc định (`TextLexer`, hiển thị code không màu/không highlight) thay vì raise lỗi — không chặn cả quá trình render chỉ vì thiếu/sai thông tin ngôn ngữ. Log warning để dễ debug
   - ✅ Strengths: resilient — không để lỗi nhỏ (ngôn ngữ không nhận diện được) chặn toàn bộ scene, vẫn hiển thị code (dù không màu) tốt hơn là lỗi hoàn toàn
   - ⚠️ Trade-offs: video có thể có code không đẹp bằng mong đợi nếu Creator gõ sai tên ngôn ngữ — chấp nhận được, không phải lỗi nghiêm trọng

B) Other (please describe after [Answer]: tag below)

[Answer]: A — Hiển thị code không tô màu (plain text) nếu ngôn ngữ không xác định được, không chặn render. Log warning để debug. (Làm rõ qua follow-up AskUserQuestion sau khi giải thích Pygments/lexer.)

### Question 5: Domain Entity — `SceneRenderResult` có cần thêm field không?
LLD đã định nghĩa `SceneRenderResult(animation_path, duration_seconds)`. Có cần thêm field khác (vd. `template_id` đã dùng, để debug/log) không?

A) 💡 Suggested: KHÔNG bổ sung — giữ nguyên như LLD, nhất quán tinh thần "API/event contract chỉ chứa dữ liệu thực sự cần dùng ở bước sau" (giống quyết định tương tự ở TTS Service Functional Design Question 5)
   - ✅ Strengths: entity gọn, tránh rò rỉ chi tiết implementation
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
