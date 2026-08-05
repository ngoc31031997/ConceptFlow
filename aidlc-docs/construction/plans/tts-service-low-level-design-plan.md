# Low-Level Design Plan — Unit 3: TTS Service

## Unit Context
- **Responsibility**: Sinh giọng đọc offline song ngữ Việt/Anh (FR4.1, FR4.2) từ đoạn lời thoại
- **Architectural style**: Hexagonal/Ports & Adapters (ADR-0002), tech stack Python 3.12 (ADR-0009 — ràng buộc thư viện Coqui TTS/Piper)
- **Interfaces**: REST duy nhất — `POST /tts/synthesize`, gọi trực tiếp (đồng bộ) bởi Rendering Service, KHÔNG tham gia RabbitMQ (`unit-of-work.md`)
- **Contract hiện có** (`component-methods.md`): Input `{ text: string, language: "vi" | "en" }` → Output `{ audio_path: string, duration_seconds: number }`
- **Depends on**: Không có unit nào (độc lập)

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `module-structure.md`
- [ ] Tạo `dependency-injection.md`
- [ ] Tạo `interface-contracts.md`
- [ ] Tạo `sequence-flows.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Layering & Dependency Direction (BẮT BUỘC)
Theo Hexagonal (ADR-0002), cần xác nhận cách map layer nội bộ, nhất quán với Unit 2 (Content Plugin Service).

A) 💡 Suggested: 3 layer — `domain/` (`TTSEnginePort` interface, `SpeechRequest`/`SpeechResult` model, validation ngôn ngữ hỗ trợ — không import FastAPI/thư viện TTS cụ thể) → `application/` (`SynthesizeSpeechUseCase`: điều phối domain + port, ghi file vào shared volume, đo `duration_seconds`) → `adapters/` (`api/` cho FastAPI REST, `tts_engines/` chứa các adapter cụ thể như `CoquiTTSAdapter`/`PiperTTSAdapter` implement `TTSEnginePort`)
   - ✅ Strengths: nhất quán với module structure Unit 2, cho phép đổi TTS engine không sửa domain/application
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Dependency Injection (BẮT BUỘC)
Cơ chế DI cho unit này?

A) 💡 Suggested: Constructor injection thủ công (theo `architectural-style.md` HLD) — `SynthesizeSpeechUseCase` nhận `TTSEnginePort` (abstraction) qua constructor; composition root là `main.py`, nơi đọc config (biến môi trường) để chọn adapter cụ thể (Coqui hoặc Piper) và wire vào FastAPI app qua `Depends()`
   - ✅ Strengths: nhất quán với Unit 2, đơn giản, cho phép đổi engine qua config không cần sửa code
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: TTS Engine Selection & Adapter Strategy
FR4.1 cho phép "Coqui TTS, Piper, hoặc tương đương". Cần xác nhận engine cụ thể dùng cho MVP và có cần hỗ trợ nhiều engine cùng lúc không.

A) 💡 Suggested: Dùng **Piper** làm engine duy nhất cho MVP (nhẹ, chạy CPU nhanh, có sẵn voice model tiếng Việt và tiếng Anh, phù hợp máy dev cá nhân không cần GPU). Chỉ implement `PiperTTSAdapter` — port `TTSEnginePort` vẫn được định nghĩa đầy đủ để có thể thêm `CoquiTTSAdapter` sau mà không sửa domain/application (đúng tinh thần Extensibility NFR1), nhưng KHÔNG implement Coqui ngay ở Unit 3 để tránh over-engineering khi chưa có nhu cầu thực tế
   - ✅ Strengths: đơn giản, đủ nhanh để phát triển MVP, vẫn giữ khả năng mở rộng qua port
   - ⚠️ Trade-offs: nếu sau này cần chất lượng giọng đọc tốt hơn (Coqui thường tự nhiên hơn Piper), phải làm thêm adapter — chấp nhận được vì port đã sẵn sàng

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Voice Model Mapping (Việt/Anh)
`language: "vi" | "en"` cần map sang voice model cụ thể của engine đã chọn.

A) 💡 Suggested: Cấu hình mapping tĩnh trong code/config (`{"vi": "<piper-vi-model-path>", "en": "<piper-en-model-path>"}`), model file được đóng gói sẵn trong Docker image (download ở build stage của Dockerfile) để tránh phụ thuộc network lúc runtime. Nếu `language` không có trong mapping → trả lỗi 400 rõ ràng
   - ✅ Strengths: đơn giản, không cần network lúc chạy, dễ mở rộng thêm ngôn ngữ (thêm entry vào mapping)
   - ⚠️ Trade-offs: tăng kích thước Docker image do bundle model file (chấp nhận được, model Piper nhỏ ~50-100MB/voice)

B) Other (please describe after [Answer]: tag below)

[Answer]:A (clarified via follow-up: "language" is the client input per existing contract; server maps language → voice model via static config, as originally proposed)

### Question 5: Artifact Storage (Shared Volume)
Output `audio_path` phải là đường dẫn Rendering Service đọc lại được. HLD đã xác định dùng Shared Docker Volume nhưng chưa có quy ước đường dẫn cụ thể (`application-design.md` line 40: "Chi tiết quy ước đường dẫn shared volume → Low-Level Design").

A) 💡 Suggested: Quy ước đường dẫn `/shared/{project_id}/audio/{scene_index}_{language}.wav` (mount volume `shared_artifacts` chung giữa TTS và Rendering trong `docker-compose.yml`, xác định cụ thể ở Infrastructure Design). `project_id` và `scene_index` được truyền vào `POST /tts/synthesize` như một phần input để hệ thống đặt tên file đúng quy ước (cần mở rộng contract so với `component-methods.md` hiện tại: `{ project_id, scene_index, text, language }`)
   - ✅ Strengths: đường dẫn dự đoán được, hỗ trợ idempotency (kiểm tra file đã tồn tại trước khi synthesize lại — theo nguyên tắc idempotency ở `services.md`)
   - ⚠️ Trade-offs: mở rộng contract input so với bản gốc ở `component-methods.md` — cần cập nhật tài liệu đó sau khi duyệt

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: API Versioning
Đã có tiền lệ từ Unit 2 (LLD Question 4): áp dụng URI versioning (`/v1/...`) cho toàn bộ REST endpoint của mọi service.

A) 💡 Suggested: Áp dụng nhất quán — endpoint là `POST /v1/tts/synthesize`. Deprecation policy giữ nguyên như đã quyết định: version cũ giữ tối thiểu 1 chu kỳ phát triển, thông báo qua changelog nội bộ
   - ✅ Strengths: nhất quán toàn hệ thống, không cần quyết định lại
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 7: Distributed Tracing & Correlation ID
Unit này được gọi trực tiếp (REST đồng bộ) bởi Rendering Service, trong phạm vi 1 bước Saga (`render_scenes`) mang `saga_id`+`project_id`.

A) 💡 Suggested: Rendering Service truyền `saga_id` qua header `X-Saga-ID` khi gọi `POST /v1/tts/synthesize`; TTS Service đưa `saga_id` vào mọi log line (không cần sinh mới, vì luôn có caller nội bộ trong context Saga). Không cần header `X-Request-ID` riêng vì đây không phải REST public — luôn có `saga_id` sẵn
   - ✅ Strengths: tái sử dụng `saga_id`, nhất quán với cách Unit 2 xử lý luồng thuộc Saga (AMQP command)
   - ⚠️ Trade-offs: nếu sau này có client khác gọi TTS ngoài luồng Saga (không có `saga_id`), cần bổ sung fallback sinh `X-Request-ID` — chưa cần ở MVP

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 8: State Management
Unit này có cần lưu trạng thái (database) không, hay hoàn toàn stateless ngoài việc ghi file audio?

A) 💡 Suggested: Stateless — không có database. Duy nhất trạng thái "bền" là file audio đã ghi vào shared volume (dùng làm cơ chế idempotency ở Question 5: nếu file đã tồn tại tại đường dẫn quy ước, trả về ngay `audio_path` + `duration_seconds` đọc từ file có sẵn, không synthesize lại)
   - ✅ Strengths: đơn giản nhất, nhất quán với nguyên tắc idempotency toàn hệ thống (`services.md`)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 9: Error Handling — Unsupported Language / Engine Failure
Khi `language` không hợp lệ hoặc engine TTS lỗi lúc synthesize (crash, timeout), phản hồi lỗi thế nào?

A) 💡 Suggested: `language` không hợp lệ → HTTP 400 với message rõ ràng (`{"error": "unsupported_language", "supported": ["vi", "en"]}`). Engine lỗi lúc synthesize → HTTP 502 (`{"error": "tts_engine_failure", "detail": "..."}`) để Rendering Service phân biệt lỗi input (400, không nên retry) với lỗi engine tạm thời (502, Rendering Service có thể retry theo cơ chế compensating action đã thiết kế ở `services.md`)
   - ✅ Strengths: phân biệt rõ lỗi permanent vs transient, khớp với chiến lược retry đã có ở Saga level
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
