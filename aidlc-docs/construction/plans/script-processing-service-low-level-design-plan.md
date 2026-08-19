# Low-Level Design Plan — Unit 4: Script Processing Service

## Unit Context
- **Responsibility**: Parse script thô thành danh sách scene chuẩn hóa (FR2.1, FR2.2), gọi Content Plugin Service để gắn category cho từng scene
- **Architectural style**: Hexagonal/Ports & Adapters (ADR-0002), tech stack Python 3.12 + FastAPI (ADR-0009, không có lý do học tập để đổi ngôn ngữ)
- **Interfaces**: AMQP consumer `parse_script` (từ `script_processing.commands`) → publish `script_parsed`/`parse_failed`; gọi Content Plugin Service để classify (cơ chế cụ thể — REST trực tiếp hay qua Orchestrator — CHƯA quyết định, để lại cho Low-Level Design theo `component-methods.md`)
- **Scene schema đã có** (`component-methods.md`): `{ scene_index, narration_text, illustration_hint, code_snippet? }`
- **Depends on**: Unit 1 (RabbitMQ), Unit 2 (Content Plugin Service) — cả hai đã hoàn thành

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
Nhất quán với Unit 2/Unit 3 (Hexagonal 3-layer)?

A) 💡 Suggested: `domain/` (`Scene`, `ParsedScript` model, `ScriptSyntaxError`, parsing rule thuần Python — không import FastAPI/aio-pika) → `application/` (`ParseScriptUseCase`: điều phối domain parser + gọi `ContentPluginPort` để classify) → `adapters/` (`messaging/` cho AMQP consumer/producer, `parsing/` chứa parser cụ thể theo cú pháp đã chọn ở Question 3, `content_plugin_client/` cho việc gọi Content Plugin Service)
   - ✅ Strengths: nhất quán toàn hệ thống, tách parser cụ thể khỏi domain (có thể đổi cú pháp script sau này mà không sửa `application/`)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Dependency Injection (BẮT BUỘC)
A) 💡 Suggested: Constructor injection thủ công — `ParseScriptUseCase` nhận `ScriptParserPort` và `ContentPluginPort` (2 abstraction) qua constructor; composition root `main.py` wire adapter cụ thể
   - ✅ Strengths: nhất quán, cho phép test độc lập từng abstraction (fake parser, fake content plugin client)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Script Syntax/Grammar (BẮT BUỘC, quyết định mới quan trọng)
Story A2 yêu cầu "script hợp lệ theo cú pháp được công cụ hỗ trợ", nhưng chưa có cú pháp cụ thể nào được định nghĩa ở Inception. Cần chọn 1 format cụ thể để parser có thể implement.

A) 💡 Suggested: **Markdown với scene delimiter** — mỗi scene là 1 heading `## Scene N` (hoặc tự động đánh số theo thứ tự heading `##`), nội dung dưới heading gồm: đoạn text thường là `narration_text`, dòng bắt đầu bằng `> ` (blockquote) là `illustration_hint`, code block ```` ``` ```` (nếu có) là `code_snippet`. Ví dụ:
     ```markdown
     ## Scene 1
     > minh họa vòng lặp for
     Đây là lời thoại giải thích vòng lặp for hoạt động như thế nào.
     ```python
     for i in range(10):
         print(i)
     ```
     ```
   - ✅ Strengths: Markdown là format Creator đã quen thuộc (Story A1 nói "script/markdown"), dễ đọc/viết tay, thư viện parse Markdown Python có sẵn (`markdown-it-py` hoặc parse thủ công bằng regex vì cú pháp đơn giản)
   - ⚠️ Trade-offs: cần định nghĩa lỗi rõ ràng khi thiếu heading/scene rỗng (Story A2 yêu cầu "thông báo lỗi rõ ràng chỉ ra vị trí")

B) YAML/JSON có cấu trúc tường minh (`scenes: [{narration_text, illustration_hint, code_snippet}]`) — không phải "script" tự nhiên như Markdown, nhưng parse đơn giản/không cần viết parser riêng
   - ✅ Strengths: parse cực đơn giản (dùng thư viện YAML/JSON chuẩn), không cần viết grammar riêng
   - ⚠️ Trade-offs: không tự nhiên để viết lời thoại dài (Story A1 mô tả Creator "soạn script" như văn bản, không phải điền form YAML) — trải nghiệm viết kém hơn

C) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Content Plugin Service Integration Mechanism (BẮT BUỘC — điểm mở từ Application Design)
`component-methods.md` để ngỏ: gọi Content Plugin Service "REST nội bộ hoặc qua Orchestrator, xác định ở Low-Level Design".

A) 💡 Suggested: **REST trực tiếp nội bộ** — sau khi parse xong danh sách scene (chưa có category), Script Processing Service gọi thẳng `GET`/`POST` tới Content Plugin Service (REST, cùng docker network) để lấy category cho từng scene, TRƯỚC KHI publish `script_parsed` (event chứa scene đã có đầy đủ category). Tương tự cách Rendering Service gọi TTS Service (REST đồng bộ nội bộ trong phạm vi 1 bước Saga)
   - ✅ Strengths: nhất quán với pattern đã có (Rendering→TTS), giảm số bước round-trip qua Orchestrator (không cần thêm bước Saga `classify_scenes` riêng — đơn giản hóa luồng Saga tổng thể so với thiết kế Saga ban đầu ở `services.md`), Content Plugin Service (Unit 2) đã có sẵn `GET /v1/plugins` nhưng CHƯA có endpoint classify qua REST (hiện tại `classify_scenes` chỉ nhận qua AMQP) — cần bổ sung 1 REST endpoint mới cho Content Plugin Service (thay đổi nhỏ, ngoài phạm vi Unit 4 nhưng cần ghi nhận)
   - ⚠️ Trade-offs: Content Plugin Service (Unit 2) đã được code generation xong với AMQP consumer `classify_scenes`, không có REST endpoint tương đương — cần một trong hai: (1) mở rộng Unit 2 thêm REST endpoint mới (revisit unit đã "complete"), hoặc (2) Script Processing Service gọi qua AMQP RPC pattern (publish + chờ event, phức tạp hơn REST)

B) **Giữ nguyên qua Orchestrator/AMQP** — Script Processing Service publish `script_parsed` với scene CHƯA có category; Orchestrator nhận event, tự phát lệnh `classify_scenes` (AMQP) tới Content Plugin Service như bước Saga riêng tiếp theo (đúng thiết kế Saga ban đầu ở `services.md`: bước 1 Parse Script → bước 2 Classify Scenes, 2 bước tách biệt)
   - ✅ Strengths: khớp chính xác thiết kế Saga đã duyệt ở Application Design (`services.md` — Parse Script và Classify Scenes là 2 bước Saga riêng biệt), không cần sửa Unit 2 đã hoàn thành, Script Processing Service không cần biết về Content Plugin Service (giảm coupling giữa 2 unit)
   - ⚠️ Trade-offs: scene trong event `script_parsed` chưa có category — Orchestrator/GUI cần đợi thêm 1 round-trip Saga nữa mới có category đầy đủ (chấp nhận được, đây là hành vi đã thiết kế từ đầu)

C) Other (please describe after [Answer]: tag below)

[Answer]:B

### Question 5: Idempotency
Consumer `parse_script` cần idempotency (tránh parse lại + publish trùng khi message được requeue)?

A) 💡 Suggested: In-memory `set[message_id]` với TTL 24h, giống Unit 2 (`nfr-design-patterns.md` Unit 2) — nhất quán cách tiếp cận
   - ✅ Strengths: nhất quán, đã có tiền lệ đơn giản
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: API Versioning / Event Schema Versioning
A) 💡 Suggested: Không có REST endpoint public cho unit này ở phương án Question 4 = B (chỉ AMQP) → N/A cho URI versioning (ADR-0008). Event schema (`script_parsed`/`parse_failed`) theo envelope chuẩn additive-only đã quyết định ở Unit 1 (`messaging-design.md`). Nếu Question 4 = A (REST tới Content Plugin), áp dụng `/v1/...` theo ADR-0008 nhất quán
   - ✅ Strengths: nhất quán, điều kiện theo lựa chọn Question 4
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 7: Distributed Tracing & Correlation ID
A) 💡 Suggested: `saga_id` từ message envelope AMQP (`parse_script` command) được dùng làm correlation ID xuyên suốt — đưa vào mọi log line, và nếu gọi REST tới Content Plugin Service (Question 4 = A), truyền qua header `X-Saga-ID` (nhất quán cách Rendering Service→TTS Service đã làm ở Unit 3)
   - ✅ Strengths: nhất quán với pattern đã có ở Unit 2 và Unit 3
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 8: Error Handling — Script Syntax Error
Story A2 yêu cầu: "hệ thống hiển thị thông báo lỗi rõ ràng chỉ ra vị trí và nguyên nhân lỗi".

A) 💡 Suggested: Parser raise `ScriptSyntaxError(line_number, reason)` khi gặp cú pháp không hợp lệ (vd. thiếu heading `## Scene N`, code block không đóng) → consumer bắt exception, publish `parse_failed` với `error_message` chứa `line_number` + `reason` cụ thể (không chỉ generic "invalid syntax"), ack message (không retry — lỗi cú pháp cần Creator sửa script, không tự phục hồi)
   - ✅ Strengths: đáp ứng đúng acceptance criteria Story A2, khớp compensating action đã thiết kế (`parse_failed` → Creator sửa script và retry từ bước 1)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 9: State Management
A) 💡 Suggested: Stateless — parse là pure function (script text → danh sách scene), không lưu trữ script hay kết quả parse trong Script Processing Service (script gốc được lưu ở phía GUI/Orchestrator theo Story A1, không phải trách nhiệm unit này)
   - ✅ Strengths: đơn giản nhất, đúng ranh giới trách nhiệm
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
