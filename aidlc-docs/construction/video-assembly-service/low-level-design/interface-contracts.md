# Interface Contracts — Unit 6: Video Assembly Service

## AMQP Consumer: command `assemble_video`
Queue: `video_assembly.commands`. Dispatched bởi Orchestrator sau khi nhận `rendering_completed` (cung cấp `scene_clip_paths`) — background music path, nếu Creator chọn, đến từ dữ liệu project (ngoài phạm vi Unit 6, ghi nhận như input có sẵn).

**Command payload**:
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "string",
  "schema_version": "1.0",
  "timestamp": "ISO-8601",
  "payload": {
    "scene_clip_paths": ["/shared/{project_id}/animations/0.mp4", "/shared/{project_id}/animations/1.mp4"],
    "scene_audio_paths": ["/shared/{project_id}/audio/0.wav", "/shared/{project_id}/audio/1.wav"],
    "background_music_path": "/shared/{project_id}/music/bg.mp3 | null"
  }
}
```
`scene_clip_paths`/`scene_audio_paths` được sắp theo `scene_index` tăng dần (đảm bảo bởi Orchestrator, khớp thứ tự scene gốc).

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
- `MissingArtifactError` (1 trong `scene_clip_paths`/`scene_audio_paths` không tồn tại trên shared volume khi bắt đầu xử lý) → `assembly_failed` với `error_message` mô tả file thiếu.
- `AssemblyEngineError` (ffmpeg exit code khác 0, hoặc timeout) → `assembly_failed` với `error_message` chứa stderr ffmpeg rút gọn (dòng cuối cùng, đủ để chẩn đoán).
- Toàn bộ lỗi coi là transient — Orchestrator retry theo compensating action đã duyệt ở `services.md`: "giữ animation/audio clip, retry chỉ bước Assembly" (input không bị xoá dù `assemble_video` lỗi).
