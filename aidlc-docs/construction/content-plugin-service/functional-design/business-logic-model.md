# Business Logic Model — Unit 2: Content Plugin Service

## Core Process: Classify Scene
```
Input: plugin_id, Scene (narration_text, illustration_hint, code_snippet?, category_hint)

1. Lookup plugin trong registry bằng plugin_id
   → nếu không tồn tại: raise PluginNotFoundError (Rule 3)
2. Validate Scene (Rule 4): narration_text không rỗng
   → nếu rỗng: raise InvalidSceneError
3. Validate category_hint nằm trong plugin.supported_categories (Rule 1, Rule 3)
   → nếu không hợp lệ: raise InvalidCategoryError
4. plugin.classify(scene) → tra bảng mapping category → animation_template_id (Rule 2)
5. Return ClassificationResult(category=category_hint, animation_template_id)
```

## Core Process: List Plugins
```
Input: (none)
1. registry.list_all() → trả toàn bộ plugin đã đăng ký lúc khởi động (Low-Level Design, Flow 3)
2. Map mỗi Plugin → PluginDTO (plugin_id, name, supported_categories)
3. Return PluginDTO[]
```

## Core Process: Handle classify_scenes Command (AMQP, multi-scene)
```
Input: AMQP message { saga_id, project_id, payload: { plugin_id, scenes: Scene[] } }

1. Idempotency check: message_id đã xử lý chưa? (Rule 5)
   → nếu rồi: ack, kết thúc (no-op)
2. For each scene trong scenes[]:
   a. Thực hiện "Classify Scene" (ở trên)
   b. Nếu lỗi ở BẤT KỲ scene nào → dừng vòng lặp, publish classification_failed
      với error_message của scene lỗi đầu tiên, ack message
   c. Nếu thành công → thu thập ClassificationResult vào danh sách kết quả
3. Nếu TẤT CẢ scene classify thành công → publish scenes_classified
   với toàn bộ danh sách ClassificationResult, ack message
```

**Lưu ý thiết kế**: Bước 2b áp dụng nguyên tắc "fail-fast toàn batch" — nếu 1 scene trong project lỗi, toàn bộ command `classify_scenes` được coi là thất bại (không classify một phần). Điều này khớp với compensating action đã thiết kế ở `services.md` (Application Design): Saga step `classification_failed` là một đơn vị thất bại/thành công trọn vẹn, Orchestrator không cần xử lý trạng thái "phân loại một phần".
