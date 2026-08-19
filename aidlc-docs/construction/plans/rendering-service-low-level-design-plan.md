# Low-Level Design Plan — Unit 5: Rendering Service

## Unit Context
- **Responsibility**: Sinh animation Manim từ cấu trúc scene (FR3.1, FR3.2), đồng bộ thời lượng animation với audio đã sinh sẵn (FR4.3)
- **Architectural style**: Hexagonal/Ports & Adapters (ADR-0002), Python 3.12 (ADR-0009)
- **Interfaces**: AMQP consumer `render_scenes` (queue `rendering.commands`) → publish `scene_rendered` (per scene, tiến trình) + `rendering_completed`/`rendering_failed`; PostgreSQL Inbox/Outbox (ADR-0013, mirror Unit 2/3/4)
- **KHÔNG còn** gọi TTS Service (ADR-0014) — audio đã có sẵn, truyền vào qua payload `render_scenes`
- **Depends on**: Unit 1 (RabbitMQ) — không phụ thuộc trực tiếp unit nào khác (theo `unit-of-work-dependency.md` đã cập nhật)

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
A) 💡 Suggested: `domain/` (`SceneRenderRequest`/`SceneRenderResult` model, `AnimationEngineError`, `AnimationRendererPort` interface — không import Manim/AMQP/Postgres cụ thể) → `application/` (`RenderSceneUseCase`: điều phối domain + port; `RenderScenesBatchUseCase`: fail-fast batch, mirror Unit 2/3) → `adapters/` (`messaging/`, `persistence/` giống hệt Unit 2/3/4, `rendering/` chứa `ManimAnimationRenderer` implement `AnimationRendererPort`, `storage/` cho shared-volume path convention)
   - ✅ Strengths: nhất quán toàn hệ thống, tách Manim cụ thể khỏi domain/application (có thể đổi animation engine sau này mà không sửa business logic)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Dependency Injection (BẮT BUỘC)
A) 💡 Suggested: Constructor injection thủ công — `RenderSceneUseCase` nhận `AnimationRendererPort` qua constructor; composition root `main.py` wire `ManimAnimationRenderer` cụ thể. Nhất quán Unit 2/3/4
   - ✅ Strengths: nhất quán, cho phép test độc lập (fake renderer)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Animation Template Selection Mechanism (BẮT BUỘC — quyết định mới quan trọng)
FR3.1 yêu cầu các pattern minh họa tái sử dụng: code syntax highlight (Story B3), animation từng bước thuật toán, sơ đồ cấu trúc dữ liệu. Content Plugin Service (Unit 2) đã gán `animation_template_id` cho mỗi scene (`algorithm_visualization` hoặc `concept_illustration`, theo Unit 2's Functional Design Question 2). Cần xác định Rendering Service dùng `animation_template_id` này thế nào để chọn đúng Manim Scene class.

A) 💡 Suggested: **Static mapping** trong code — `{"algorithm_visualization": AlgorithmVisualizationScene, "concept_illustration": ConceptIllustrationScene}` (Python dict, tương tự `voice_registry.py` ở TTS Service). Nếu `animation_template_id` không có trong mapping → lỗi rõ ràng (`UnsupportedTemplateError`). Nếu scene có `code_snippet` (bất kể template nào) → LUÔN chèn thêm `Code` mobject (Manim's syntax-highlight component, dùng Pygments) hiển thị code trước/sau phần animation chính — đáp ứng Story B3 độc lập với category
   - ✅ Strengths: đơn giản, không over-engineer (chỉ 2 template ở MVP theo Unit 2), đủ linh hoạt vì `code_snippet` xử lý tách biệt khỏi category-based template
   - ⚠️ Trade-offs: thêm template mới cần sửa code Rendering Service (chấp nhận được ở quy mô MVP — khác với dynamic plugin loading của Content Plugin Service, vì Manim Scene class phức tạp hơn nhiều 1 plugin phân loại đơn giản, không đáng để xây cơ chế discovery động ở giai đoạn này)

B) **Dynamic plugin loading**, tương tự Content Plugin Service (ADR-0006) — mỗi template là 1 plugin riêng trong `adapters/rendering/templates/`, tự động discover lúc khởi động
   - ✅ Strengths: nhất quán pattern với Unit 2, dễ mở rộng thêm template mà không sửa code lõi
   - ⚠️ Trade-offs: phức tạp hơn đáng kể cho lợi ích chưa rõ ràng ở MVP (chỉ có 2 template cố định) — over-engineering theo YAGNI

C) Other (please describe after [Answer]: tag below)

[Answer]:B

### Question 4: `render_scenes` Command Payload (BẮT BUỘC)
Rendering Service cần đủ dữ liệu mỗi scene để render: `narration_text`, `illustration_hint`, `code_snippet`, `category`/`animation_template_id` (từ Content Plugin), `audio_path`/`duration_seconds` (từ TTS). Dữ liệu này đến từ 3 bước Saga khác nhau (Parse Script, Classify Scenes, Synthesize Speech) — ai gộp lại?

A) 💡 Suggested: **Orchestrator gộp dữ liệu** từ 3 event trước đó (`script_parsed`, `scenes_classified`, `speech_synthesized`) thành 1 payload đầy đủ khi dispatch `render_scenes` — đây là hệ quả tự nhiên của yêu cầu "Orchestrator persist kết quả từng bước" (đã ghi nhận cho Unit 8). Payload: `{ scenes: [{ scene_index, narration_text, illustration_hint, code_snippet, category, animation_template_id, audio_path, duration_seconds }] }`
   - ✅ Strengths: Rendering Service không cần biết Content Plugin/TTS tồn tại (giữ decoupling đúng tinh thần Saga orchestration-based), đúng trách nhiệm Orchestrator là nơi duy nhất biết toàn bộ luồng
   - ⚠️ Trade-offs: đây là điểm cần Unit 8 (Orchestrator) implement đúng khi tới lượt — ghi nhận như 1 ràng buộc thiết kế cho Unit 8, không phải rủi ro cho Unit 5

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Animation-Audio Timing Sync (FR4.3)
A) 💡 Suggested: Mỗi Manim Scene nhận `duration_seconds` (từ audio) làm tham số, tự set `run_time` tổng của animation con để khớp đúng thời lượng đó (Manim hỗ trợ `self.wait()`/scale run_time của từng animation con theo tỷ lệ). Sai lệch cho phép: làm tròn tới 0.1s (đủ chính xác cho video giáo dục, không cần frame-perfect sync)
   - ✅ Strengths: đơn giản, Manim's API hỗ trợ trực tiếp việc set run_time
   - ⚠️ Trade-offs: animation phức tạp (nhiều animation con) cần logic scale tỷ lệ cẩn thận — nhưng đây là chi tiết implementation của từng Scene class cụ thể, không phải kiến trúc chung

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: Execution Model — Manim Rendering (CPU-bound, chậm)
Manim render là tác vụ CPU-bound NẶNG (có thể mất hàng chục giây tới vài phút mỗi scene, khác biệt hoàn toàn so với parsing Markdown của Unit 4).

A) 💡 Suggested: Chạy Manim trong `ThreadPoolExecutor` (mirror TTS Service's `PiperTTSAdapter`), với timeout dài hơn đáng kể — **300 giây (5 phút)/scene** — do animation phức tạp hơn nhiều so với TTS synthesis. Vượt timeout → `AnimationEngineError` → `rendering_failed`
   - ✅ Strengths: nhất quán pattern threadpool+timeout đã có ở TTS, không block event loop
   - ⚠️ Trade-offs: 300s là ước lượng, có thể cần điều chỉnh sau khi có dữ liệu thực tế (tương tự TTS's 60s ban đầu)

B) Other (please describe after [Answer]: tag below)

[Answer]: B — Timeout giữ 300s mặc định nhưng đọc từ biến môi trường (không hardcode, có thể nâng khi cần). BỔ SUNG: publish thêm event `scene_render_started` (qua Outbox) ngay khi bắt đầu render 1 scene — trước `scene_rendered` khi xong — để GUI biết hệ thống đang chạy, không "im lặng" trong lúc chờ render lâu (làm rõ qua follow-up AskUserQuestion).

### Question 7: Idempotency (Artifact-Level)
A) 💡 Suggested: Giống TTS Service — kiểm tra file animation clip đã tồn tại tại đường dẫn quy ước (`/shared/{project_id}/animations/{scene_index}.mp4`) trước khi render lại. Nếu tồn tại → trả kết quả có sẵn ngay, không render lại
   - ✅ Strengths: nhất quán, hỗ trợ "retry không làm lại" (yêu cầu đã ghi nhận cho Unit 8)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 8: Batch Processing Semantics
A) 💡 Suggested: Fail-fast, mirror Unit 2/3/4 (`ClassifyScenesBatchUseCase`/`SynthesizeSpeechBatchUseCase`/tương tự) — dừng ngay ở scene lỗi đầu tiên, publish `rendering_failed`. Do idempotency (Question 7), lần retry sau (Orchestrator gửi lại `render_scenes`) sẽ skip các scene đã render thành công, chỉ render lại từ scene lỗi — khớp mô tả compensating action đã có ở `services.md` ("giữ scene đã render thành công, retry chỉ scene lỗi")
   - ✅ Strengths: nhất quán toàn hệ thống, đúng khớp thiết kế compensating action đã duyệt
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 9: Per-Scene Progress Event (`scene_rendered`)
`component-methods.md` xác nhận publish `scene_rendered` "per scene" (không chỉ khi cả batch xong) — khác với Unit 2/3/4 chỉ publish 1 event duy nhất cho cả batch.

A) 💡 Suggested: Publish `scene_rendered` (qua Outbox) NGAY sau khi mỗi scene render xong (trong vòng lặp batch, trước khi tiếp tục scene tiếp theo) — nhiều Outbox row cho 1 lần xử lý command, KHÔNG chỉ 1 row cuối cùng. Cuối cùng, khi hết batch (thành công hết) → thêm 1 event `rendering_completed`; nếu fail-fast ở giữa → thêm 1 event `rendering_failed` thay vì `rendering_completed`
   - ✅ Strengths: đáp ứng đúng yêu cầu "tiến trình theo từng scene" (Story C6 — theo dõi tiến trình render trong GUI), Outbox pattern vẫn hoạt động bình thường với nhiều row (không giới hạn 1 event/command)
   - ⚠️ Trade-offs: nhiều Outbox row hơn Unit 2/3/4 — chấp nhận được, đúng bản chất yêu cầu tiến trình real-time

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 10: Correlation ID & Error Handling
A) 💡 Suggested: `saga_id` từ envelope AMQP, nhất quán Unit 2/3/4. Lỗi Manim (crash/timeout) → `AnimationEngineError` → `rendering_failed` với `error_message` (transient, Orchestrator có thể retry). Không có lỗi "permanent" nào đặc thù ở Rendering Service (khác Unit 4 với lỗi cú pháp) vì input đã được validate ở các bước trước
   - ✅ Strengths: nhất quán, đúng phân loại lỗi
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 11: State Management
A) 💡 Suggested: Stateless ngoài animation clip trên shared volume (không lưu business data khác) — nhất quán TTS Service. Postgres chỉ chứa Outbox/Inbox (ADR-0013)
   - ✅ Strengths: đúng bản chất, nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
