# Business Rules — Unit 3: TTS Service

## Rule 1: Text Validation
`text` (sau khi strip whitespace) bắt buộc không rỗng. Nếu rỗng → raise domain error (`EmptyTextError`) → HTTP 400 (`{"error": "empty_text"}`).
- **Rationale**: Tránh sinh audio vô nghĩa (silence-only file). Không giới hạn độ dài tối đa ở tầng domain — Piper tự xử lý được text dài, giới hạn thực tế (nếu cần) thuộc phạm vi NFR Design.

## Rule 2: Language Validation
`language` phải thuộc `{"vi", "en"}` (đã có trong `voice_registry.py`). Nếu không → raise `UnsupportedLanguageError` → HTTP 400 (`{"error": "unsupported_language", "supported": ["vi", "en"]}`).
- **Rationale**: FR4.2 chỉ yêu cầu hỗ trợ Việt/Anh cho MVP.

## Rule 3: No Text Preprocessing
`text` được truyền nguyên văn cho engine TTS (Piper) — không normalize, không strip ký hiệu đặc biệt/thuật ngữ kỹ thuật.
- **Rationale**: Trách nhiệm viết lời thoại "đọc được" thuộc về Creator khi soạn script (ngoài phạm vi TTS Service — TTS Service không phải NLP service).

## Rule 4: Idempotency via Shared-Volume Path
Trước khi gọi engine synthesize, kiểm tra file đã tồn tại tại đường dẫn quy ước (`/shared/{project_id}/audio/{scene_index}_{language}.wav`). Nếu tồn tại → đọc `duration_seconds` từ file có sẵn, KHÔNG synthesize lại.
- **Rationale**: Khớp nguyên tắc idempotency toàn hệ thống (`services.md`) — Rendering Service có thể retry scene lỗi mà không tạo lại audio đã sinh thành công.

## Rule 5: Duration Measurement
`duration_seconds` được đo trực tiếp từ metadata file `.wav` vừa ghi (frame count / sample rate, thư viện chuẩn `wave`), làm tròn 2 chữ số thập phân — KHÔNG ước lượng từ độ dài text.
- **Rationale**: Chính xác 100%, cần thiết cho FR4.3 (đồng bộ animation với audio ở Unit 5).

## Rule 6: No Special Handling for Edge-Case Text
Text ngắn/toàn ký hiệu (đã pass Rule 1) được coi là input hợp lệ, không có business rule riêng. Nếu engine lỗi khi xử lý → rơi vào lỗi chung (`TTSEngineError` → HTTP 502).
- **Rationale**: Tránh thêm business rule phức tạp cho edge case hiếm gặp trong thực tế (script do Creator viết).

## Error Classification Summary
| Error | HTTP Status | Retry-able? |
|---|---|---|
| `EmptyTextError` | 400 | No (permanent — client phải sửa input) |
| `UnsupportedLanguageError` | 400 | No (permanent — client phải sửa input) |
| `TTSEngineError` | 502 | Yes (transient — Rendering Service có thể retry theo compensating action) |
