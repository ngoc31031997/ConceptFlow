# Functional Design Plan — Unit 8: Orchestrator Service

## Unit Context
- **Stories**: C1 (khởi chạy render — orchestration nền), C6 (theo dõi tiến trình — nguồn sự kiện), E3 (đăng video — orchestration nền)
- **Scope**: Saga coordination cho 2 Saga (Render Pipeline 5 bước, Publish 1 bước); quản lý state machine của video project; xử lý lỗi + compensating action (retry-by-step). KHÔNG chịu trách nhiệm business logic của từng bước (đó là các service nghiệp vụ) — chỉ điều phối thứ tự, gộp dữ liệu giữa các bước, và theo dõi trạng thái.

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `business-logic-model.md`
- [x] Tạo `business-rules.md`
- [x] Tạo `domain-entities.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Business Logic — Data Aggregation for `render_scenes` (BẮT BUỘC — đã ghi nhận là ràng buộc thiết kế từ Unit 5)
Unit 5's Functional Design đã xác nhận: Orchestrator gộp dữ liệu từ 3 event (`script_parsed`, `scenes_classified`, `speech_synthesized`) thành 1 payload đầy đủ cho command `render_scenes`. Cần quy tắc gộp cụ thể.

A) 💡 Suggested: Gộp theo `scene_index` làm khóa — với mỗi scene, lấy `narration_text`/`illustration_hint`/`code_snippet`/`code_language` từ `script_parsed`, `category`/`animation_template_id` từ `scenes_classified`, `audio_path`/`duration_seconds` từ `speech_synthesized`. Cả 3 event PHẢI có cùng số lượng scene với cùng tập `scene_index` (đảm bảo vì cùng xuất phát từ 1 `script_parsed` ban đầu) — nếu không khớp (số lượng scene khác nhau giữa các event), đây là lỗi hệ thống nghiêm trọng (bug ở 1 trong 3 service), Orchestrator raise lỗi rõ ràng thay vì gộp sai lệch âm thầm, chuyển status → `failed_at_render_scenes` với `error_message` mô tả cụ thể mismatch
   - ✅ Strengths: đơn giản, phát hiện lỗi hệ thống sớm thay vì để Rendering Service nhận payload sai lệch khó debug
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Business Logic — Data Aggregation for `assemble_video`
Tương tự Question 1, `assemble_video` cần `scene_clip_paths`+`scene_audio_paths` (từ `rendering_completed`) — nhưng `speech_synthesized` (bước 3) đã có `audio_path` per scene rồi. Dùng nguồn nào?

A) 💡 Suggested: Dùng `audio_path` đã lưu từ `speech_synthesized` (bước 3) — KHÔNG lấy lại từ `rendering_completed` (Rendering Service không trả lại audio_path, chỉ trả `animation_path` theo Unit 5's interface-contracts.md). Orchestrator gộp: `clip_path` từ `rendering_completed`'s `scene_clip_paths` (theo `scene_index`), `audio_path` từ `speech_synthesized` đã lưu ở bước trước đó (`Project.scenes`)
   - ✅ Strengths: đúng nguồn dữ liệu thực tế theo interface đã duyệt ở Unit 5/6, không có nguồn nào khác cung cấp audio_path
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Business Rule — `background_music_path` Source (Unit 6's Question, chưa trả lời ở đâu)
Unit 6's Low-Level Design ghi nhận `background_music_path` "đến từ dữ liệu project, ngoài phạm vi Unit 6". Ai cung cấp giá trị này cho Orchestrator?

A) 💡 Suggested: `background_music_path` là optional field trong `POST /v1/sagas/render`'s input (Creator chọn nhạc nền lúc cấu hình project ở GUI, Story C5 — nếu không chọn, `null`). Orchestrator lưu vào `Project` lúc khởi tạo Saga, dùng lại khi dispatch `assemble_video` — KHÔNG cần thêm bước Saga riêng để lấy nhạc nền, chỉ là 1 field trong input ban đầu
   - ✅ Strengths: đơn giản, đúng bản chất (nhạc nền là lựa chọn cấu hình tĩnh của Creator, không phải kết quả của 1 bước xử lý nào)
   - ⚠️ Trade-offs: `interface-contracts.md`'s `POST /v1/sagas/render` input hiện chưa có field này — cần bổ sung (Revision)

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Business Rule — `Project.Status` Transition Validity
A) 💡 Suggested: Mỗi state transition chỉ hợp lệ từ ĐÚNG trạng thái liền trước theo thứ tự Saga (vd. `synthesizing_speech` → `rendering` chỉ hợp lệ nếu status hiện tại là `synthesizing_speech`, không phải bất kỳ trạng thái nào khác) — validate ở `HandleStepEventUseCase` trước khi update, khớp với Low-Level Design Question 5's "unexpected event" safeguard (Flow 6). `POST /v1/sagas/publish` chỉ hợp lệ khi status = `ready_to_publish` (đã ghi ở interface-contracts.md, xác nhận lại là business rule chính thức)
   - ✅ Strengths: đảm bảo state machine không bị corrupt bởi event đến sai thứ tự
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Business Rule — Retry Payload Reconstruction (Question 8 của LLD, chi tiết hóa)
`RetryStepUseCase` cần "tái tạo command payload từ dữ liệu đã lưu". Với từng bước cụ thể, dữ liệu nào cần lưu ở `Project` để tái tạo được?

A) 💡 Suggested: `Project` lưu đủ dữ liệu tích lũy qua từng bước: `script_content` (từ input ban đầu), `scenes` (cập nhật dần: thêm category/animation_template_id sau bước 2, audio_path/duration sau bước 3, clip_path sau bước 4), `plugin_id`/`voice_language` (input ban đầu), `background_music_path` (Question 3), `video_path` (sau bước 5), youtube metadata (input Saga Publish). Retry bước N chỉ cần đọc `Project` hiện tại + tái tạo payload theo đúng format command bước N (không cần gọi lại các bước trước)
   - ✅ Strengths: đơn giản — `Project` đã là "single source of truth" tích lũy, không cần lưu riêng từng payload command đã gửi
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Business Scenario — Concurrent Projects
Hệ thống chỉ 1 Creator nhưng có thể có NHIỀU project đồng thời (mỗi project 1 Saga độc lập). Có giới hạn số Saga chạy song song không?

A) 💡 Suggested: KHÔNG giới hạn số project/Saga chạy đồng thời ở tầng Orchestrator — mỗi `project_id` độc lập hoàn toàn (không share state), giới hạn thực tế tự nhiên đến từ tài nguyên máy dev cá nhân (Rendering/TTS/Video Assembly mỗi service tự prefetch=1, nhất quán các unit trước) chứ không phải logic nghiệp vụ ở Orchestrator
   - ✅ Strengths: đơn giản, đúng bản chất (không có ràng buộc nghiệp vụ nào giới hạn số project đồng thời)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7: Domain Entity — Progress Message Content cho `scene_rendered` (Question 4/ADR-0017)
A) 💡 Suggested: Giữ nguyên format `ProgressMessage` đã định nghĩa ở Low-Level Design (`project_id, step, status, scene_index?, scene_total?, error_message?`) — với `scene_rendered`, map `step="render_scenes"`, `status="in_progress"`, `scene_index`+`scene_total` từ event payload. KHÔNG cần thêm field nào khác (GUI chỉ cần hiển thị "đang render scene X/Y", Story C6)
   - ✅ Strengths: đơn giản, đủ cho nhu cầu GUI đã xác nhận
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
