# Functional Design Plan — Unit 2: Content Plugin Service

## Unit Context
- **Stories**: B1 (chọn plugin), B2 (chọn loại nội dung lập trình cụ thể — thuật toán/khái niệm)
- Theo Story B2: Creator **tự chọn** loại nội dung (thuật toán/cấu trúc dữ liệu hay khái niệm lập trình) qua GUI, chứ không phải hệ thống tự suy luận từ nội dung text.

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `business-logic-model.md`
- [ ] Tạo `business-rules.md`
- [ ] Tạo `domain-entities.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Business Logic — Nguồn quyết định "category"
`component-methods.md` định nghĩa input `classify-scene` có `category_hint` (optional). Theo Story B2, Creator tự chọn loại nội dung qua GUI. Vậy `ClassifySceneUseCase` nên xử lý category như thế nào?

A) 💡 Suggested: `category_hint` là **bắt buộc phải có giá trị hợp lệ** khi gọi từ luồng thực tế (GUI luôn gửi kèm vì Creator đã chọn ở Story B2) — plugin chỉ **validate** hint có nằm trong `supported_categories` của plugin không, KHÔNG tự suy luận từ text. Nếu hint rỗng hoặc không hợp lệ → lỗi validation (`InvalidCategoryError`), Orchestrator nhận `classification_failed`
   - ✅ Strengths: đơn giản, tin cậy quyết định của người dùng (đúng tinh thần Story B2 — "Creator chỉ định"), không cần logic suy luận NLP phức tạp
   - ⚠️ Trade-offs: nếu tương lai muốn tự động gợi ý category từ nội dung, cần thêm logic riêng (ngoài phạm vi hiện tại)

B) Plugin tự suy luận category từ `scene_content` (dùng heuristic/keyword) khi không có hint, dùng hint làm override nếu có
   - ✅ Strengths: linh hoạt hơn, Creator không bắt buộc phải chọn thủ công
   - ⚠️ Trade-offs: cần xây dựng logic suy luận (dù đơn giản) — thêm độ phức tạp không cần thiết vì GUI đã yêu cầu chọn tường minh ở Story B2

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Business Rule — Mapping category → animation_template_id
Với plugin "Lập trình", mỗi category (`algorithm`, `concept`) map tới `animation_template_id` cụ thể như thế nào? Có phải 1-1 hay cần thêm chi tiết phân loại?

A) 💡 Suggested: Mapping tĩnh đơn giản trong MVP — `algorithm` → `animation_template_id = "algorithm_visualization"`, `concept` → `animation_template_id = "concept_illustration"` (chỉ 2 template chung cho mỗi category ở giai đoạn này; Rendering Service (Unit 5) sẽ tự chọn animation cụ thể hơn dựa trên nội dung scene thực tế, không phải trách nhiệm của Content Plugin Service)
   - ✅ Strengths: đơn giản, đúng ranh giới trách nhiệm (Content Plugin chỉ phân loại ở mức category, Rendering Service quyết định animation cụ thể)
   - ⚠️ Trade-offs: nếu sau này cần nhiều template con hơn (vd. riêng cho "sorting" vs "graph traversal"), cần mở rộng mapping

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Error Handling — Plugin không tồn tại
Nếu `plugin_id` gửi lên (vd. từ command `classify_scenes`) không tồn tại trong registry, xử lý thế nào?

A) 💡 Suggested: Raise `PluginNotFoundError` (domain exception) → tầng `adapters/messaging/consumer.py` bắt exception, publish event `classification_failed` với `error_message` mô tả rõ plugin_id không tồn tại, ack message (không retry — đây là lỗi cấu hình/logic, retry không giúp ích)
   - ✅ Strengths: khớp cơ chế compensating action đã thiết kế (Orchestrator đánh dấu `failed_at_classify_scenes`, Creator có thể sửa cấu hình và thử lại)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Domain Entity — Scene validation rules
`Scene` model có `narration_text`, `illustration_hint`, `code_snippet?`. Có ràng buộc validation nào cho scene khi classify không (vd. `narration_text` không được rỗng)?

A) 💡 Suggested: `narration_text` bắt buộc không rỗng (mọi scene phải có lời thoại); `code_snippet` optional (chỉ scene minh họa code walkthrough — Story B3 — mới có); không giới hạn độ dài ở tầng domain (giới hạn kỹ thuật nếu cần sẽ ở tầng API/infra)
   - ✅ Strengths: ràng buộc tối thiểu, đủ để tránh lỗi rõ ràng (scene rỗng), không over-validate
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
