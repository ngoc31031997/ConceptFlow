# Business Rules — Unit 6: Video Assembly Service

## Rule 1: Zero-Trust Input Validation (Question 1)
Không tin tưởng dữ liệu từ Orchestrator dù đã qua các bước trước — validate toàn bộ trước khi chạy ffmpeg:
- `scenes` không rỗng.
- Mỗi `SceneAssemblyInput`: `clip_path` và `audio_path` không rỗng, và file thực sự tồn tại trên shared volume (không chỉ hợp lệ cú pháp string).
- Nếu `background_music_path` khác `null`: file đó cũng phải tồn tại.
- `project_id` không rỗng.
- Vi phạm bất kỳ điều nào → `MissingArtifactError`, không chạy ffmpeg.

## Rule 2: Scene Ordering — Explicit Index, Không Tin Tưởng Thứ Tự Mảng (Question 2, Revision LLD)
Payload mang `scene_index` tường minh cho mỗi scene (không còn dựa vào thứ tự vị trí trong mảng). Video Assembly Service:
- Sort `scenes` theo `scene_index` tăng dần trước khi xử lý.
- Validate dãy `scene_index` liên tục từ 0 (không thiếu, không trùng) — vi phạm → `InvalidSceneIndexError`.
- Thứ tự phát trong video cuối cùng = thứ tự `scene_index` sau khi sort, không phải thứ tự xuất hiện trong payload gốc.

## Rule 3: Media Format Consistency (Question 3, Revision LLD)
Trước khi mux, kiểm tra bằng ffprobe rằng mọi animation clip (`clip_path`) có cùng codec, resolution, framerate. Không đồng nhất → `InconsistentMediaFormatError` (liệt kê scene lệch chuẩn trong `error_message`), KHÔNG chạy ffmpeg mux/concat (tránh lỗi ffmpeg khó chẩn đoán giữa chừng).

## Rule 4: Video/Audio Track Handling Trong Mux & Concat (Question 3, gốc)
Sau khi Rule 3 xác nhận format đồng nhất: mux dùng `-c:v copy` (không re-encode video), concat dùng `-c copy` (nhất quán format nên an toàn). Không có business rule riêng để xử lý mismatch còn sót lại — nếu ffmpeg vẫn thất bại sau khi qua Rule 3 (trường hợp cực hiếm), lỗi đó rơi vào `AssemblyEngineError` (lỗi hạ tầng/vận hành, không phải business logic).

## Rule 5: Background Music Volume — Cố Định, Không Tùy Chỉnh (Question 4)
- Âm lượng giọng đọc: 100% (không đổi).
- Âm lượng nhạc nền: 20% cố định (không cho Creator tùy chỉnh ở MVP — ngoài phạm vi Story C5 hiện tại).
- `background_music_path` là `null`/absent → bỏ qua hoàn toàn bước overlay, video chỉ có giọng đọc.
- Nhạc nền dài hơn video → cắt tại điểm kết thúc video (`-shortest`).
- Nhạc nền ngắn hơn video → lặp lại (`-stream_loop -1`) cho tới khi đủ độ dài, sau đó cắt theo `-shortest`.

## Rule 6: Domain Entity Scope — `VideoAssemblyResult` Tối Giản (Question 5)
`VideoAssemblyResult` chỉ chứa `video_path` — không tính/truyền thêm `duration_seconds` hay `file_size_bytes` qua event. Publisher Service (Unit 7) chỉ cần `video_path`; GUI preview (Unit 10) tự lấy metadata trực tiếp từ file nếu cần.

## Rule 7: Uniform Processing Regardless of Scene Count (Question 6)
Không có nhánh logic đặc biệt cho project 1 scene — cùng 1 code path (mux → concat → overlay) áp dụng cho mọi số lượng scene ≥ 1.

## Rule 8: Idempotency (LLD Question 7, không đổi)
Nếu `/shared/{project_id}/video/final.mp4` đã tồn tại khi nhận command → trả kết quả ngay (publish `video_assembled` với path có sẵn), không chạy lại ffmpeg, không validate lại input.

## Rule 9: Error Classification — Tất Cả Transient (LLD Question 8, không đổi)
Mọi lỗi (`MissingArtifactError`, `InvalidSceneIndexError`, `InconsistentMediaFormatError`, `AssemblyEngineError`) đều publish `assembly_failed` và coi là transient — animation/audio clip input KHÔNG bị xoá, Orchestrator có thể retry command `assemble_video` theo compensating action đã duyệt.
