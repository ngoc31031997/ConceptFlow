# Interface Contracts — Unit 6: Video Assembly Service

## AMQP Consumer: command `assemble_video`
Queue: `video_assembly.commands`. Dispatched bởi Orchestrator sau khi nhận `rendering_completed` (cung cấp `scene_clip_paths`) — background music path, nếu Creator chọn, đến từ dữ liệu project (ngoài phạm vi Unit 6, ghi nhận như input có sẵn).

**Command payload** (**Revision, Functional Design Question 2**: đổi từ 2 mảng path song song sang 1 mảng object có `scene_index` tường minh — zero trust, không tin tưởng thứ tự mảng đến từ Orchestrator):
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "string",
  "schema_version": "1.0",
  "timestamp": "ISO-8601",
  "payload": {
    "scenes": [
      { "scene_index": 0, "clip_path": "/shared/{project_id}/animations/0.mp4", "audio_path": "/shared/{project_id}/audio/0.wav" },
      { "scene_index": 1, "clip_path": "/shared/{project_id}/animations/1.mp4", "audio_path": "/shared/{project_id}/audio/1.wav" }
    ],
    "background_music_path": "/shared/{project_id}/music/bg.mp3 | null"
  }
}
```
`FfmpegVideoAssembler` tự sort `scenes` theo `scene_index` tăng dần trước khi mux/concat (không tin tưởng thứ tự mảng đến sẵn từ Orchestrator) — validate không thiếu/trùng `scene_index` (dãy liên tục từ 0), xem `business-rules.md` (Functional Design).

## AMQP Producer: events
Publish tới `orchestrator.events` (qua Outbox + `OutboxRelay`, ADR-0013).

**`video_assembled`** (1 lần, khi thành công):
```json
{
  "payload": {
    "event_type": "video_assembled",
    "video_path": "/shared/{project_id}/video/final.mp4"
  }
}
```

**`assembly_failed`** (1 lần, khi lỗi):
```json
{
  "payload": {
    "event_type": "assembly_failed",
    "error_message": "string"
  }
}
```

## Event Sequence Per Command
Đúng 1 Outbox row/command (khác Unit 5's nhiều row per-scene) — vì `assemble_video` là 1 thao tác duy nhất trên toàn bộ project, không có progress event trung gian (Question 10).

## Delivery Guarantee & Idempotency
At-least-once (kế thừa Unit 1). Inbox (`processed_messages`) dedupe `message_id` ở mức COMMAND, cùng transaction với việc ghi Outbox row kết quả — mirror Unit 2/3/4/5. Artifact-level idempotency (Question 7): `AssembleVideoUseCase` kiểm tra file `/shared/{project_id}/video/final.mp4` đã tồn tại trước khi chạy ffmpeg lại; nếu tồn tại → trả kết quả ngay, không assembly lại.

## Correlation ID
`saga_id` từ envelope AMQP, gắn vào mọi log line qua `adapters/logging/correlation.py`. Không có REST endpoint nào (nhất quán TTS/Script Processing/Rendering Service).

## Internal Port Contract (domain/ports.py)
```python
class VideoAssemblerPort(ABC):
    @abstractmethod
    def assemble(self, request: VideoAssemblyRequest, output_path: str) -> None:
        """Muxes per-scene animation+audio, concatenates in scene order,
        overlays optional background music, writes result to output_path."""
```

## ffmpeg Execution Timeout (Question 6)
Đọc từ biến môi trường `ASSEMBLY_TIMEOUT_SECONDS` (mặc định 180s = 3 phút, áp dụng cho toàn bộ chuỗi mux+concat+overlay). Vượt timeout → `AssemblyEngineError` → `assembly_failed`.

## Error Classification (Question 8)
- `MissingArtifactError` (1 trong `clip_path`/`audio_path`/`background_music_path` không tồn tại trên shared volume khi bắt đầu xử lý) → `assembly_failed` với `error_message` mô tả file thiếu.
- `InvalidSceneIndexError` (**Revision, Functional Design Question 2**: `scene_index` bị thiếu/trùng trong dãy liên tục từ 0) → `assembly_failed` với `error_message` mô tả index sai.
- `InconsistentMediaFormatError` (**Revision, Functional Design Question 3**: ffprobe pre-check phát hiện codec/resolution/framerate không đồng nhất giữa các animation clip) → `assembly_failed` với `error_message` liệt kê scene lệch chuẩn.
- `AssemblyEngineError` (ffmpeg exit code khác 0, hoặc timeout) → `assembly_failed` với `error_message` chứa stderr ffmpeg rút gọn (dòng cuối cùng, đủ để chẩn đoán).
- Toàn bộ lỗi coi là transient — Orchestrator retry theo compensating action đã duyệt ở `services.md`: "giữ animation/audio clip, retry chỉ bước Assembly" (input không bị xoá dù `assemble_video` lỗi).

---

## Revision 2026-09-07 — CR-002: timeline offsets thay cho ghép nối liền

**Vấn đề**: timeline của video Manim là `Σ(self.play) + Σ(self.wait)`. Hợp đồng cũ chỉ chuyển danh sách audio path, và Video Assembly nối chúng liền nhau — tức ngầm giả định video **chỉ gồm narration**. Không phải vậy: mỗi đoạn narration từ đoạn thứ 2 trở đi bị phát sớm đúng bằng thời gian animation đã chạy trước nó, và sai số **cộng dồn**. Đo được **61.6s lệch** ở cuối một video 3.6 phút (xem `aidlc-docs/construction/build-and-test/long-form-baseline.md`).

### `rendering_completed` (Unit 5 → Orchestrator) — payload mở rộng
```json
{
  "payload": {
    "event_type": "rendering_completed",
    "video_path": "/shared/{project_id}/video/rendered.mp4",
    "wait_offsets": [0.0, 9.57, 20.47],
    "video_duration_seconds": 214.9
  }
}
```
- `wait_offsets[i]` = giây (tính từ đầu video) mà `self.wait(AUTO)` thứ i **bắt đầu** — tức nơi narration thứ i phải phát. **Không phải** tổng dồn thời lượng narration.
- Đo bằng cách thay `self.wait(AUTO)` → `(_cf_mark(self, i), self.wait(D))` + preamble ghi `scene.renderer.time` vào `cf_marks.jsonl`. Vẫn là thay thế văn bản thuần — Rendering không bao giờ tự execute script của Creator ngoài subprocess đã cô lập.
- `video_duration_seconds` lấy từ `ffprobe` trên file thật, không tính bằng số học (Manim làm tròn theo frame nên số học lệch nhẹ).
- Sidecar `/shared/{project_id}/video/timing.json` lưu cùng dữ liệu này, để đường đi idempotent (video đã tồn tại) vẫn báo cáo được offset. Video thiếu sidecar sẽ được **render lại**, không tái sử dụng.

### `assemble_video` (Orchestrator → Unit 6) — payload đổi
```json
{
  "payload": {
    "video_path": "/shared/{project_id}/video/rendered.mp4",
    "narration_segments": [
      { "audio_path": "/shared/{project_id}/audio/0.wav", "start_time": 0.0 },
      { "audio_path": "/shared/{project_id}/audio/1.wav", "start_time": 9.57 }
    ],
    "video_duration_seconds": 214.9,
    "background_music_path": "... | null",
    "subtitle_cues": [ { "scene_index": 0, "text": "...", "start_time": 0.0, "end_time": 2.5 } ],
    "subtitle_style": { }
  }
}
```
- **`audio_segments` (mảng string) đã bị thay bằng `narration_segments`**. Consumer của Unit 6 vẫn đọc được shape cũ để xử lý command còn tồn trong queue lúc deploy, nhưng ghi log cảnh báo — project đó cần render lại.
- `subtitle_cues[i].start_time` cũng lấy từ `wait_offsets[i]`, vì cùng lý do.
- Unit 6 dựng filtergraph `adelay=<ms>:all=1` cho từng đoạn rồi `amix=inputs=N:normalize=0` (bắt buộc `normalize=0`, nếu không amix chia volume cho N).
- `-shortest` được thay bằng `-t <target>` với `target = max(video_duration, kết thúc narration cuối)`, kèm `tpad=stop_mode=clone` để giữ frame cuối — không bao giờ cắt cụt câu kết (FR10.6).

### Ràng buộc bắt buộc (FR10.5)
Số phần tử `wait_offsets` PHẢI bằng số scene. Orchestrator validate và **fail saga** khi lệch, thay vì dựng video lệch tiếng. Nguyên nhân thường gặp: `self.wait(AUTO)` nằm trong vòng lặp hoặc `if` nên chạy số lần khác với số marker `# NARRATION:`.
