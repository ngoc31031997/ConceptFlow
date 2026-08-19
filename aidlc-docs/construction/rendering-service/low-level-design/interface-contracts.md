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
