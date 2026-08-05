# Functional Design Plan — Unit 3: TTS Service

## Unit Context
- **Stories**: B4 (chọn ngôn ngữ giọng đọc), C2 (sinh giọng đọc tự động từ script, FR4.1/FR4.2), C3 (đồng bộ animation với thời lượng — TTS Service chỉ cung cấp `duration_seconds`, đồng bộ thực tế là trách nhiệm của Rendering Service/Unit 5)
- **Scope**: FR4.1, FR4.2 — TTS Service chỉ chịu trách nhiệm sinh audio + đo thời lượng; KHÔNG chịu trách nhiệm đồng bộ animation (đó là FR4.3, thuộc Unit 5)

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `business-logic-model.md`
- [ ] Tạo `business-rules.md`
- [ ] Tạo `domain-entities.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Business Logic — Text Preprocessing trước khi synthesize
`narration_text` từ script có thể chứa ký tự đặc biệt, số, hoặc thuật ngữ kỹ thuật (vd. tên hàm, ký hiệu code) khi Creator viết lời thoại giải thích code. TTS engine (Piper) cần input là plain text thuần.

A) 💡 Suggested: KHÔNG xử lý/normalize text đặc biệt ở MVP — truyền `narration_text` trực tiếp cho Piper nguyên văn (Piper tự xử lý text-to-phoneme cơ bản). Trách nhiệm viết lời thoại "đọc được" (vd. viết "vòng lặp for" thay vì để nguyên `for`) thuộc về Creator khi soạn script — ngoài phạm vi TTS Service
   - ✅ Strengths: đơn giản nhất cho MVP, đúng ranh giới trách nhiệm (TTS Service không phải NLP service)
   - ⚠️ Trade-offs: nếu Creator viết lời thoại có nhiều ký hiệu code thô, chất lượng giọng đọc có thể kém tự nhiên — chấp nhận được, có thể cải thiện sau nếu cần

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Business Rule — Validation `narration_text`
Ràng buộc nào cho `text` khi gọi `POST /v1/tts/synthesize`?

A) 💡 Suggested: `text` bắt buộc không rỗng (sau khi strip whitespace) → nếu rỗng, HTTP 400 (`{"error": "empty_text"}`). Không giới hạn độ dài tối đa ở tầng domain (Piper tự xử lý được đoạn text dài; nếu phát sinh vấn đề hiệu năng thực tế sẽ giới hạn sau ở NFR Design)
   - ✅ Strengths: ràng buộc tối thiểu, tránh lỗi rõ ràng (audio rỗng vô nghĩa), không over-validate
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Business Logic — Đo `duration_seconds`
Sau khi Piper sinh file `.wav`, cách tính `duration_seconds` trả về cho Rendering Service (dùng để đồng bộ animation ở FR4.3)?

A) 💡 Suggested: Đọc trực tiếp metadata của file `.wav` vừa ghi (frame count / sample rate) bằng thư viện chuẩn (`wave` module trong Python standard library) — không dựa vào ước lượng từ độ dài text (không chính xác). Giá trị trả về là số thực (giây), làm tròn 2 chữ số thập phân
   - ✅ Strengths: chính xác 100% vì đọc từ file thực tế, không cần thư viện ngoài, đủ độ chính xác cho FR4.3
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Business Scenario — Text rất ngắn hoặc chỉ có khoảng trắng/ký hiệu
Nếu `narration_text` chỉ có 1-2 ký tự hoặc toàn ký hiệu (không có âm tiết đọc được), engine có thể sinh audio gần như im lặng hoặc lỗi.

A) 💡 Suggested: Không xử lý đặc biệt ở business logic layer — coi đây là input hợp lệ (đã pass validation Question 2 vì không rỗng), để Piper tự xử lý. Nếu Piper crash/timeout với input này, rơi vào nhánh lỗi chung `TTSEngineError` → HTTP 502 (đã thiết kế ở Low-Level Design)
   - ✅ Strengths: không thêm business rule phức tạp cho edge case hiếm gặp trong thực tế (script do Creator viết, không phải input ngẫu nhiên)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Domain Entity — `SpeechRequest`/`SpeechResult` có cần thêm field nào không?
Low-Level Design đã định nghĩa `SpeechRequest(project_id, scene_index, text, language)` và `SpeechResult(audio_path, duration_seconds)`. Có cần bổ sung field nào phục vụ business logic (vd. `sample_rate`, `voice_model_id` dùng để debug/log) không?

A) 💡 Suggested: Giữ nguyên như Low-Level Design, không bổ sung field. `voice_model_id` cụ thể được dùng nội bộ (`voice_registry.py`) và có thể log ra (structured log), nhưng không cần đưa vào entity trả về API — API contract chỉ cần đúng những gì `component-methods.md`/Rendering Service thực sự cần dùng
   - ✅ Strengths: giữ entity gọn, tránh rò rỉ chi tiết implementation (voice model cụ thể) ra ngoài contract
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
