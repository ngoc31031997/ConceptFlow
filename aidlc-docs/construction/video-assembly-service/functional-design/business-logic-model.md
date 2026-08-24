# Business Logic Model — Unit 6: Video Assembly Service

## Core Business Process: Assemble Video (FR5.1, FR5.2 — Stories C4, C5)

**Input**: 1 command `assemble_video` chứa danh sách scene đã render (animation clip câm + audio clip giọng đọc), theo `scene_index`, cộng nhạc nền tùy chọn.

**Output**: 1 video .mp4 hoàn chỉnh trên shared volume, hoặc lỗi rõ ràng nếu bất kỳ bước nào thất bại.

**Processing Steps** (tuần tự, không có khái niệm batch/per-scene progress — Question 10):

1. **Idempotency check**: nếu `/shared/{project_id}/video/final.mp4` đã tồn tại → trả kết quả ngay, bỏ qua toàn bộ các bước sau (Flow 5, LLD Question 7).
2. **Input validation (zero trust)**: `scenes` không rỗng; mỗi `SceneAssemblyInput` có `clip_path`/`audio_path` không rỗng; file tương ứng thực sự tồn tại trên shared volume; nếu có `background_music_path`, file đó cũng tồn tại (Question 1).
3. **Scene ordering validation**: sort `scenes` theo `scene_index` tăng dần; xác nhận dãy liên tục từ 0, không thiếu/trùng index (Question 2, Revision LLD).
4. **Media format consistency check**: chạy ffprobe trên từng `clip_path`, so sánh codec/resolution/framerate giữa các scene — không đồng nhất thì dừng ngay (Question 3, Revision LLD).
5. **Per-scene mux**: với mỗi scene (theo thứ tự đã sort), mux animation clip (video-only) + audio clip (giọng đọc) thành 1 file trung gian, không re-encode video.
6. **Concatenation**: nối các file muxed theo đúng thứ tự `scene_index` thành 1 video liên tục, không re-encode.
7. **Background music overlay** (nếu có `background_music_path`): mix nhạc nền (20% âm lượng cố định) với audio hiện có (100%, giọng đọc), lặp nhạc nền nếu ngắn hơn video, cắt nếu dài hơn (Question 4).
8. **Cleanup**: xoá file trung gian sau khi assembly thành công.
9. **Publish kết quả**: `video_assembled` (thành công) hoặc `assembly_failed` (bất kỳ bước 2–7 lỗi) — đúng 1 event/command.

## Business Process Boundary
Video Assembly Service KHÔNG chịu trách nhiệm:
- Render animation (Unit 5) hay sinh giọng đọc (Unit 3) — chỉ tiêu thụ artifact đã có sẵn.
- Xác định `background_music_path` — dữ liệu này đến từ cấu hình project (ngoài phạm vi Unit 6, do Orchestrator truyền vào).
- Điều chỉnh âm lượng theo yêu cầu Creator — mức 20%/100% cố định ở MVP (Question 4, Story C5 không yêu cầu tùy chỉnh).

## Single-Scene Edge Case (Question 6)
Không có nhánh logic đặc biệt — pipeline chạy đầy đủ (mux 1 scene → concat với 1 input → overlay nhạc nền nếu có) giống mọi số lượng scene khác.
