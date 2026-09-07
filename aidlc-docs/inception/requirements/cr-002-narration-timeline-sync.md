# CR-002 — Đồng bộ narration/phụ đề theo timeline thật của video (P0, blocker)

## Date
2026-09-07

## Stage
Requirements Analysis (Change Request trên `requirements.md` + CR-001 đã duyệt)

## Intent Analysis
Creator muốn xuất video dài 5–10 phút để đăng YouTube/TikTok và bật kiếm tiền. Với video dài, lỗi lệch tiếng/hình hiện tại của pipeline làm mọi video không dùng được.

## Vấn đề (đã xác minh trong code)

Timeline video do Manim sinh ra là:

```
video_duration = Σ(thời gian self.play(...))  +  Σ(thời gian self.wait(...))
```

- `services/rendering/adapters/rendering/manim_renderer.py` (`_patch_auto_waits`) thay `self.wait(AUTO)` thứ i bằng đúng `duration_seconds` của narration thứ i. Nghĩa là **narration i phải phát đúng trong khoảng `wait` thứ i**, khoảng này nằm sau toàn bộ animation đã chạy trước đó.
- `services/video-assembly/adapters/assembly/ffmpeg_assembler.py` (`_run_pipeline`) lại **nối các file audio liền nhau** bằng `concat=n=N:v=0:a=1`. Track audio kết quả chỉ dài `Σ(narration)`, **không chừa chỗ cho `Σ(self.play)`**.
- `services/orchestrator/internal/application/handle_step_event.go` (`subtitleCues`) tính `start_time` bằng cách cộng dồn `DurationSeconds` — mắc **cùng một lỗi**.

**Hệ quả**: narration thứ i bị phát sớm hơn hình đúng bằng tổng thời gian animation chạy trước nó. Sai số **cộng dồn**: video 10 phút có ~3–4 phút `self.play` thì đến cuối video tiếng lệch hình vài phút. Phụ đề lệch y hệt. Ngoài ra `-shortest` sẽ cắt cụt phần video không có audio phủ.

Lỗi này ẩn ở video rất ngắn (1–2 scene, ít animation) nên đã lọt qua CR-001.

## Functional Requirements

### FR10 — Timeline chuẩn (mới)
- **FR10.1**: Rendering Service PHẢI xác định và báo cáo **offset thật** (giây, tính từ đầu video) của từng `self.wait(AUTO)` trong video đã render.
- **FR10.2**: Rendering Service PHẢI trả các offset này lên `rendering_completed` event, kèm tổng thời lượng video thực tế.
- **FR10.3**: Video Assembly PHẢI đặt mỗi đoạn narration vào đúng offset của nó trên track audio (không nối liền nhau nữa).
- **FR10.4**: Phụ đề PHẢI dùng chính các offset đó làm `start_time` (`end_time = start_time + duration`).
- **FR10.5**: Số offset trả về PHẢI khớp số narration segment; lệch nhau ⇒ saga fail với lỗi rõ ràng, KHÔNG được render ra video lệch tiếng.
- **FR10.6**: Nếu narration cuối tràn quá thời lượng video, PHẢI kéo dài video (giữ frame cuối) thay vì cắt cụt audio.

### FR5 — Ảnh hưởng dây chuyền
- **FR5.5 (mới)**: `VideoAssemblyRequest` mang `narration_segments: [{audio_path, start_time}]` thay cho `audio_segments: [path]`.
- **FR3.5 (mới)**: `ScriptRenderResult` mang thêm `wait_offsets: float[]` và `video_duration_seconds: float`.

## Ràng buộc / Rủi ro kỹ thuật

- **C1 — Cách lấy offset**: Manim không có API xuất timeline. Phương án đề xuất: thay `self.wait(AUTO)` bằng biểu thức `(_cf_mark(self, i), self.wait(D))` và chèn preamble định nghĩa `_cf_mark` ghi `self.renderer.time` vào file `marks.jsonl` trong `media_dir`. Đây vẫn là **thay thế văn bản thuần** (giữ nguyên nguyên tắc "Script Processing không bao giờ execute script").
- **C2 — Phụ thuộc nội bộ Manim**: `scene.renderer.time` là API nội bộ của Manim CE, có thể đổi giữa các version. PHẢI pin version Manim trong `requirements.txt` và có test chạy render thật kiểm chứng offset.
- **C3 — Bảo mật không đổi**: preamble chạy trong subprocess đã bị strip env + resource limit như hiện tại (ADR pending về sandbox không đổi).
- **C4 — Tương thích ngược**: project cũ trong DB không có `wait_offsets`. Khi thiếu, fallback về hành vi cũ (nối liền) và ghi log cảnh báo — hoặc yêu cầu re-render. Quyết ở Low-Level Design.
- **C5 — `amix` vs `adelay`**: dùng `adelay=<ms>|<ms>` cho từng input rồi `amix=inputs=N:normalize=0` (bắt buộc `normalize=0`, nếu không ffmpeg chia đều volume và narration bị nhỏ đi N lần).

## Phạm vi tác động (unit)
- **Rendering** (Unit 5): `manim_renderer.py` sinh marker + đọc `marks.jsonl`; `render_script.py`; `producer.py` (event payload); `domain/models.py`.
- **Orchestrator** (Unit 8): lưu `wait_offsets` trên `domain.Project`; `subtitleCues()` dùng offset; payload `assemble_video`; schema Postgres.
- **Video Assembly** (Unit 6): `ffmpeg_assembler.py` (`adelay`+`amix` thay `concat`), `domain/models.py`, `tpad` cho FR10.6.

## Tiêu chí nghiệm thu
1. Video test 5 phút, ≥15 narration xen kẽ `self.play` dài: đo lệch audio/hình ở narration cuối **< 200ms**.
2. Phụ đề khớp giọng đọc trong cả 2 chế độ (bật TTS / tắt TTS).
3. Narration cuối không bị cắt cụt.
4. Số offset ≠ số segment ⇒ saga fail có message rõ ràng, không sinh video.

## Câu hỏi còn mở
1. Project cũ thiếu `wait_offsets`: fallback nối liền (có cảnh báo) hay bắt buộc re-render? → quyết ở LLD (C4).
