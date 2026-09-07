# Interface Contracts — Unit 5: Rendering Service

## AMQP Consumer: command `render_scenes`
Queue: `rendering.commands`. Dispatched bởi Orchestrator sau khi nhận đủ `script_parsed`+`scenes_classified`+`speech_synthesized` cho 1 project — Orchestrator gộp dữ liệu 3 bước đó thành payload đầy đủ (Question 4; ghi nhận làm ràng buộc thiết kế cho Unit 8).

**Command payload**:
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "string",
  "schema_version": "1.0",
  "timestamp": "ISO-8601",
  "payload": {
    "scenes": [
      {
        "scene_index": 0,
        "narration_text": "string",
        "illustration_hint": "string | null",
        "code_snippet": "string | null",
        "code_language": "string | null",
        "animation_template_id": "algorithm_visualization | concept_illustration",
        "audio_path": "string",
        "duration_seconds": 12.5
      }
    ]
  }
}
```

## AMQP Producer: events
Publish tới `orchestrator.events` (qua Outbox + `OutboxRelay`, ADR-0013).

**`scene_render_started`** (MỚI — mỗi scene, publish ngay trước khi bắt đầu render, Question 6 follow-up):
```json
{ "payload": { "event_type": "scene_render_started", "scene_index": 0 } }
```

**`scene_rendered`** (mỗi scene, publish ngay sau khi render xong — Question 9):
```json
{
  "payload": {
    "event_type": "scene_rendered",
    "scene_index": 0,
    "animation_path": "/shared/{project_id}/animations/0.mp4",
    "duration_seconds": 12.5
  }
}
```

**`rendering_completed`** (1 lần, cuối batch, khi TẤT CẢ scene thành công):
```json
{ "payload": { "event_type": "rendering_completed", "scene_count": 5 } }
```

**`rendering_failed`** (1 lần, khi fail-fast dừng batch — Question 8):
```json
{
  "payload": {
    "event_type": "rendering_failed",
    "scene_index": 2,
    "error_message": "string"
  }
}
```

## Event Sequence Per Command (Outbox — nhiều row/command, khác Unit 2/3/4)
Với N scene, batch thành công tạo ra: N × (`scene_render_started` + `scene_rendered`) + 1 × `rendering_completed` = 2N+1 Outbox row. Nếu lỗi ở scene k (0-based): (k+1) × `scene_render_started` + k × `scene_rendered` + 1 × `rendering_failed`.

## Delivery Guarantee & Idempotency
At-least-once (kế thừa Unit 1). Inbox (`processed_messages`) dedupe `message_id` bền vững ở mức COMMAND (không phải per-scene — 1 command `render_scenes` = 1 message_id, dù publish nhiều event). Artifact-level idempotency (Question 7): `RenderSceneUseCase` kiểm tra file `.mp4` đã tồn tại tại đường dẫn shared volume trước khi render lại (mirror TTS Service's Business Rule 4).

## Correlation ID
`saga_id` từ envelope AMQP, gắn vào mọi log line qua `adapters/logging/correlation.py`. Không có REST endpoint nào (nhất quán TTS Service/Script Processing Service sau retrofit).

## Internal Port Contracts (domain/ports.py)
```python
class AnimationRendererPort(ABC):
    @abstractmethod
    def render(self, request: SceneRenderRequest, output_path: str) -> float:
        """Renders animation to output_path, returns actual duration_seconds."""

class AnimationTemplatePort(ABC):
    @property
    @abstractmethod
    def template_id(self) -> str: ...

    @abstractmethod
    def build_scene(self, request: SceneRenderRequest) -> "manim.Scene":
        """Constructs a configured Manim Scene ready to .render()."""
```

## Manim Execution Timeout (Question 6)
Đọc từ biến môi trường `RENDER_TIMEOUT_SECONDS` (mặc định 300s = 5 phút) — không hardcode, có thể nâng khi cần theo yêu cầu người dùng. Vượt timeout → `AnimationEngineError` → `rendering_failed`.

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
