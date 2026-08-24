# Functional Design Plan — Unit 6: Video Assembly Service

## Unit Context
- **Stories**: C4 (ghép animation + audio thành video hoàn chỉnh), C5 (thêm nhạc nền tùy chọn)
- **Scope**: Ghép animation clip (câm, từ Rendering Service) + audio clip (giọng đọc, từ TTS Service) theo thứ tự scene + nhạc nền tùy chọn thành 1 video .mp4 hoàn chỉnh (FR5.1, FR5.2). KHÔNG chịu trách nhiệm render animation hay sinh giọng đọc — chỉ ghép các artifact đã có sẵn.

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `business-logic-model.md`
- [ ] Tạo `business-rules.md`
- [ ] Tạo `domain-entities.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Business Rule — Input Validation Strategy
LLD's `MissingArtifactError` chỉ kiểm tra file tồn tại. Unit 5 (Rendering) đã quyết định "zero trust" — validate toàn bộ input dù đã qua các bước trước. Video Assembly Service nên áp dụng mức validate nào?

A) 💡 Suggested: **Zero trust**, nhất quán Unit 5 — validate toàn bộ input trước khi chạy ffmpeg: (1) `scene_clip_paths` và `scene_audio_paths` không rỗng và có cùng độ dài; (2) mỗi file trong 2 danh sách thực sự tồn tại trên shared volume (không chỉ tin path string hợp lệ cú pháp); (3) nếu có `background_music_path`, file đó cũng phải tồn tại; (4) `project_id` không rỗng. Vi phạm bất kỳ điều nào → `MissingArtifactError`/`InvalidRequestError` rõ ràng, không chạy ffmpeg
   - ✅ Strengths: nhất quán Unit 5, tránh ffmpeg fail với lỗi khó hiểu (vd. "no such file") khi input sai — lỗi được phát hiện sớm và rõ ràng hơn
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Business Logic — Scene Ordering Guarantee
LLD giả định `scene_clip_paths`/`scene_audio_paths` đã được Orchestrator sắp theo `scene_index` tăng dần. Video Assembly Service có nên tự sắp xếp lại hay tin tưởng thứ tự đến từ payload?

A) 💡 Suggested: **Tin tưởng thứ tự mảng** (index trong `scene_clip_paths[i]` tương ứng `scene_audio_paths[i]`, và thứ tự mảng = thứ tự phát trong video cuối cùng) — không cần thêm field `scene_index` tường minh trong payload vì đây là 1 mảng đã sắp sẵn, không phải map theo key. Đây là hợp đồng dữ liệu giữa Orchestrator và Video Assembly Service, không phải business rule cần validate lại (khác việc kiểm tra file tồn tại)
   - ✅ Strengths: đơn giản, đúng bản chất mảng đã sắp — thêm field `scene_index` dư thừa vì thứ tự mảng đã mã hoá thông tin đó
   - ⚠️ Trade-offs: nếu Orchestrator có bug gửi sai thứ tự, Video Assembly Service không phát hiện được (tin tưởng tuyệt đối vào thứ tự mảng) — chấp nhận được vì đây là internal contract giữa 2 unit do cùng 1 hệ thống kiểm soát, không phải input từ user

B) Other (please describe after [Answer]: tag below)

[Answer]: Follow-up resolved — **Thêm `scene_index` tường minh**: payload đổi từ 2 mảng path song song sang 1 mảng object `{scene_index, clip_path, audio_path}`. Video Assembly Service tự sort theo `scene_index` và validate không thiếu/trùng index trước khi ghép — không tin tưởng thứ tự mảng đến từ Orchestrator.

### Question 3: Business Rule — Video/Audio Codec & Container Assumptions
LLD's per-scene mux dùng `-c:v copy` (không re-encode video). Business rule: nếu animation clip có codec/resolution không đồng nhất giữa các scene (vd. do lỗi ở Rendering Service), xử lý sao?

A) 💡 Suggested: KHÔNG tự phát hiện/xử lý trường hợp codec không đồng nhất ở mức business logic — đây là ràng buộc kỹ thuật đã ghi nhận ở LLD (giả định hợp lý vì cùng nguồn Manim với 1 config cố định). Nếu ffmpeg concat thất bại do mismatch, lỗi đó tự nhiên rơi vào `AssemblyEngineError` (đã có ở LLD) — không cần business rule riêng để "phát hiện sớm" vì đây là lỗi hạ tầng/vận hành (Rendering Service config sai), không phải business logic của Video Assembly Service
   - ✅ Strengths: đúng ranh giới trách nhiệm — Video Assembly Service không cần biết chi tiết codec hợp lệ, để ffmpeg tự báo lỗi khi có vấn đề
   - ⚠️ Trade-offs: lỗi codec mismatch sẽ có `error_message` là ffmpeg stderr chung chung, không phải business error rõ ràng — chấp nhận được, đây là trường hợp hiếm gặp (chỉ xảy ra khi Rendering Service có lỗi cấu hình)

B) Other (please describe after [Answer]: tag below)

[Answer]: Follow-up resolved — **Pre-check bằng ffprobe**: trước khi mux, chạy ffprobe trên từng animation clip lấy codec/resolution/framerate. Nếu không đồng nhất giữa các scene → business error rõ ràng (`InconsistentMediaFormatError`) NGAY, không chạy ffmpeg mux/concat.

### Question 4: Business Rule — Background Music Duration Edge Cases (Story C5)
LLD Question 5 đã quyết định: nhạc nền dài hơn → cắt (`-shortest`); ngắn hơn → loop (`-stream_loop -1`). Cần làm rõ business rule: âm lượng giọng đọc (100%) và nhạc nền (20%) có cố định tuyệt đối, hay Creator có thể tùy chỉnh?

A) 💡 Suggested: Cố định 20%/100%, KHÔNG cho Creator tùy chỉnh ở MVP (Story C5 chỉ yêu cầu "thêm nhạc nền tùy chọn", không yêu cầu điều chỉnh mức âm lượng — đó sẽ là 1 story riêng nếu cần trong tương lai). `background_music_path` là `null`/absent → hoàn toàn bỏ qua bước overlay nhạc nền, video chỉ có giọng đọc
   - ✅ Strengths: đúng đúng phạm vi Story C5 hiện tại, tránh over-engineering (thêm field `music_volume` chưa có nhu cầu)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Domain Entity — `VideoAssemblyResult` có cần thêm field không?
LLD đã định nghĩa `VideoAssemblyResult(video_path)`. Có cần thêm field khác (vd. `duration_seconds` tổng, `file_size_bytes`) để phục vụ bước sau (Publisher Service hoặc GUI preview) không?

A) 💡 Suggested: KHÔNG bổ sung — giữ nguyên như LLD. Publisher Service (Unit 7) chỉ cần `video_path` để upload (theo `component-methods.md`'s `publish_video` payload); GUI preview (Story D1, Unit 10) có thể lấy `duration_seconds`/`file_size` trực tiếp từ file video qua API riêng nếu cần, không cần Video Assembly Service tính toán và truyền qua event
   - ✅ Strengths: entity gọn, nhất quán tinh thần "event contract chỉ chứa dữ liệu thực sự cần dùng ở bước sau" (Unit 3/5's Functional Design Question 5)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: Business Scenario — Single Scene (Edge Case)
Nếu project chỉ có 1 scene (`scene_clip_paths`/`scene_audio_paths` có 1 phần tử), luồng xử lý có khác gì so với nhiều scene?

A) 💡 Suggested: KHÔNG khác — vẫn chạy đủ pipeline (mux scene 0 → concat demuxer chỉ với 1 file → overlay nhạc nền nếu có). Concat demuxer với 1 file vẫn hoạt động bình thường (không cần optimize case đặc biệt "bỏ qua bước concat nếu chỉ 1 scene") — giữ code đơn giản, nhất quán 1 code path cho mọi số lượng scene
   - ✅ Strengths: đơn giản, không cần nhánh logic đặc biệt, ffmpeg concat demuxer xử lý đúng với 1 input
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
