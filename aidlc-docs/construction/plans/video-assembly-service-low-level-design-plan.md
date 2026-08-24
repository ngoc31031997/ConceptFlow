# Low-Level Design Plan — Unit 6: Video Assembly Service

## Unit Context
- **Responsibility**: Ghép animation clip + audio clip (+ nhạc nền tùy chọn) của tất cả scene thành 1 video .mp4 hoàn chỉnh (FR5.1, FR5.2)
- **Architectural style**: Hexagonal/Ports & Adapters (ADR-0002), Python 3.12 (ADR-0009), ffmpeg qua Python binding/subprocess
- **Interfaces**: AMQP consumer `assemble_video` (queue `video_assembly.commands`) → publish `video_assembled`/`assembly_failed`; PostgreSQL Inbox/Outbox (ADR-0013, mirror Unit 2/3/4/5)
- **Input** (`component-methods.md`): `{ saga_id, project_id, scene_clip_paths: string[], scene_audio_paths: string[], background_music_path? }`
- **Output** (`video_assembled`): `{ saga_id, project_id, video_path }`
- **Depends on**: Unit 1 (RabbitMQ) — không phụ thuộc trực tiếp unit nào khác

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
A) 💡 Suggested: `domain/` (`VideoAssemblyRequest`/`VideoAssemblyResult` model, `AssemblyEngineError`, `VideoAssemblerPort` interface — không import ffmpeg/AMQP/Postgres cụ thể) → `application/` (`AssembleVideoUseCase`: điều phối domain + port, nối scene clip theo thứ tự `scene_index`, mix nhạc nền nếu có) → `adapters/` (`messaging/`, `persistence/` giống hệt Unit 2/3/4/5, `assembly/` chứa `FfmpegVideoAssembler` implement `VideoAssemblerPort`, `storage/` cho shared-volume path convention)
   - ✅ Strengths: nhất quán toàn hệ thống, tách ffmpeg cụ thể khỏi domain/application (có thể đổi công cụ ghép video sau này mà không sửa business logic)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Dependency Injection (BẮT BUỘC)
A) 💡 Suggested: Constructor injection thủ công — `AssembleVideoUseCase` nhận `VideoAssemblerPort` qua constructor; composition root `main.py` wire `FfmpegVideoAssembler` cụ thể. Nhất quán Unit 2/3/4/5
   - ✅ Strengths: nhất quán, cho phép test độc lập (fake assembler)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: ffmpeg Invocation Mechanism
A) 💡 Suggested: `subprocess` gọi ffmpeg CLI trực tiếp (không dùng wrapper library như `ffmpeg-python`) — build command args tường minh trong `FfmpegVideoAssembler`, capture stderr để log lỗi chi tiết. Timeout qua `subprocess.run(..., timeout=...)`
   - ✅ Strengths: kiểm soát đầy đủ command ffmpeg (concat, mix audio, encode), không phụ thuộc wrapper library có thể lỗi thời/thiếu tính năng, dễ debug (command args log được nguyên văn)
   - ⚠️ Trade-offs: phải tự viết command string/args cẩn thận (rủi ro lỗi cú pháp ffmpeg) — chấp nhận được, có thể unit test qua fake subprocess

B) Dùng `ffmpeg-python` (Python binding, xây filter graph bằng API Python thay vì raw command string)
   - ✅ Strengths: code Python thuần, tránh lỗi cú pháp command string thủ công
   - ⚠️ Trade-offs: thêm dependency, một số filter phức tạp (audio mixing/ducking) vẫn cần rơi về raw filter string bên trong wrapper — lợi ích không lớn cho use case tương đối đơn giản này

C) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 4: Assembly Strategy — Per-Scene Merge rồi Concat (BẮT BUỘC — quyết định kỹ thuật cốt lõi)
Mỗi scene có 1 animation clip (video-only, từ Rendering Service/Manim) và 1 audio clip (giọng đọc, từ TTS Service) riêng biệt, cùng độ dài (`duration_seconds` đã đồng bộ ở Unit 5). Cần ghép thành 1 video hoàn chỉnh theo đúng thứ tự scene.

A) 💡 Suggested: 2 bước ffmpeg:
   1. **Per-scene mux**: với mỗi scene, mux animation clip (video) + audio clip (audio) thành 1 file trung gian `scene_{i}_muxed.mp4` (ffmpeg `-map 0:v -map 1:a -c:v copy -c:a aac`, không re-encode video vì đã đúng codec từ Manim)
   2. **Concat**: dùng ffmpeg `concat demuxer` (file danh sách `.txt` liệt kê các `scene_{i}_muxed.mp4` theo thứ tự `scene_index`) để nối thành 1 video liên tục, `-c copy` (không re-encode, vì tất cả clip cùng codec/resolution do cùng render từ Manim)
   3. **Background music** (nếu có `background_music_path`): overlay lên video đã concat, dùng `amix` filter với volume nhạc nền giảm (ducking) so với giọng đọc — video track giữ nguyên (`-c:v copy`), chỉ re-encode audio track
   - ✅ Strengths: `-c copy` cho video ở bước 1 và 2 tránh re-encode tốn CPU (animation clip có thể dài/nặng), concat demuxer đơn giản và tin cậy khi tất cả input cùng codec (đảm bảo vì cùng nguồn Manim)
   - ⚠️ Trade-offs: giả định tất cả animation clip cùng codec/resolution/framerate (hợp lý vì Unit 5 dùng 1 Manim config cố định) — nếu sau này có nhiều codec khác nhau sẽ cần re-encode ở bước concat

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Background Music Ducking (FR5.2)
A) 💡 Suggested: Volume nhạc nền cố định giảm còn **20%** (áp dụng `volume=0.2` filter trước khi `amix` với giọng đọc ở 100%) — không cần dynamic ducking (audio-level detection) phức tạp cho MVP. Nếu nhạc nền dài hơn video → cắt (`-shortest`); nếu ngắn hơn → loop (`-stream_loop -1` trước `-shortest`)
   - ✅ Strengths: đơn giản, đủ dùng cho video giáo dục (nhạc nền chỉ là phụ trợ, không cần tinh chỉnh phức tạp), tránh over-engineering
   - ⚠️ Trade-offs: volume cố định 20% có thể không tối ưu cho mọi trường hợp — chấp nhận được ở MVP, có thể điều chỉnh qua biến môi trường sau

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 6: Execution Model — ffmpeg (CPU/IO-bound, tuỳ độ dài video)
A) 💡 Suggested: Chạy ffmpeg trong `ThreadPoolExecutor` (mirror Unit 3/5's pattern), timeout đọc từ biến môi trường, mặc định **180 giây (3 phút)** cho toàn bộ assembly (per-scene mux + concat + music overlay cộng dồn, thường nhanh hơn Manim render vì phần lớn dùng `-c copy`). Vượt timeout → `AssemblyEngineError` → `assembly_failed`
   - ✅ Strengths: nhất quán pattern threadpool+timeout đã có ở Unit 3/5, không block event loop
   - ⚠️ Trade-offs: 180s là ước lượng ban đầu, có thể cần điều chỉnh sau khi có dữ liệu thực tế (tương tự Unit 3/5)

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 7: Idempotency (Artifact-Level)
A) 💡 Suggested: Giống Unit 3/5 — kiểm tra file video cuối cùng đã tồn tại tại đường dẫn quy ước (`/shared/{project_id}/video/final.mp4`) trước khi assembly lại. Nếu tồn tại → trả kết quả có sẵn ngay, không chạy lại ffmpeg. File trung gian (`scene_{i}_muxed.mp4`) lưu ở thư mục tạm riêng (`/shared/{project_id}/video/_tmp/`), dọn sau khi concat xong thành công
   - ✅ Strengths: nhất quán, hỗ trợ "retry không làm lại" (yêu cầu đã ghi nhận cho Unit 8)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 8: Correlation ID & Error Handling
A) 💡 Suggested: `saga_id` từ envelope AMQP, nhất quán Unit 2/3/4/5. Lỗi ffmpeg (non-zero exit code, timeout, file thiếu ở `scene_clip_paths`/`scene_audio_paths`) → `AssemblyEngineError` → `assembly_failed` với `error_message` (chứa stderr ffmpeg rút gọn). Toàn bộ lỗi coi là transient (Orchestrator có thể retry theo compensating action đã duyệt: "giữ animation/audio clip, retry chỉ bước Assembly")
   - ✅ Strengths: nhất quán, đúng khớp thiết kế compensating action đã có ở `services.md`
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 9: State Management
A) 💡 Suggested: Stateless ngoài video output + file trung gian trên shared volume (không lưu business data khác). Postgres chỉ chứa Outbox/Inbox (ADR-0013)
   - ✅ Strengths: đúng bản chất, nhất quán Unit 3/5
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 10: Batch/Single-Command Semantics
Khác với Unit 2/3/4/5 (xử lý nhiều scene trong 1 command theo batch), `assemble_video` là **1 thao tác duy nhất** trên toàn bộ project (không có khái niệm "per-scene" ở mức output — chỉ có 1 kết quả `video_assembled`/`assembly_failed` cho cả command).

A) 💡 Suggested: Không cần fail-fast batch logic như Unit 2/3/4/5 — xử lý tuần tự nội bộ (mux từng scene → concat → music overlay) nhưng chỉ publish 1 event kết quả duy nhất cho cả command (không có progress event per-scene, vì không có ý nghĩa nghiệp vụ tương đương "per-scene rendering progress" của Unit 5 — cả quá trình assembly thường nhanh hơn nhiều so với rendering)
   - ✅ Strengths: đơn giản, đúng bản chất thao tác (1 input → 1 output), không over-engineer progress tracking cho tác vụ ngắn
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A
