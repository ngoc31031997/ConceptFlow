# Interface Contracts — Unit 3: TTS Service

## External REST API

### `POST /v1/tts/synthesize`
API versioning theo ADR-0008 (URI versioning, system-wide). Gọi trực tiếp (đồng bộ) bởi Rendering Service — không qua RabbitMQ.

**Headers**:
| Header | Required | Description |
|---|---|---|
| `X-Saga-ID` | Yes | Correlation ID của Saga hiện tại (Rendering Service luôn có sẵn trong context `render_scenes`), đưa vào mọi log line |

**Request body**:
```json
{
  "project_id": "string",
  "scene_index": 0,
  "text": "string",
  "language": "vi"
}
```
- `language`: enum `"vi" | "en"` (mở rộng contract gốc trong `component-methods.md` với `project_id`, `scene_index` để hỗ trợ quy ước đường dẫn shared volume — xem Note dưới)

**Response body (200 OK)**:
```json
{
  "audio_path": "/shared/{project_id}/audio/{scene_index}_{language}.wav",
  "duration_seconds": 12.5
}
```

**Error responses**:
| Status | Body | Khi nào |
|---|---|---|
| 400 | `{"error": "unsupported_language", "supported": ["vi", "en"]}` | `language` không hợp lệ (permanent error, Rendering Service không nên retry) |
| 502 | `{"error": "tts_engine_failure", "detail": "<message>"}` | Engine TTS lỗi lúc synthesize — crash/timeout (transient error, Rendering Service có thể retry theo compensating action đã thiết kế ở `services.md`) |

## Deprecation Policy
Theo ADR-0008: version cũ (`/v1/`) được giữ tối thiểu 1 chu kỳ phát triển sau khi version mới (`/v2/`, nếu có) ra mắt. Thông báo qua changelog nội bộ (chỉ 1 client nội bộ — Rendering Service).

## Correlation/Trace ID Propagation
- **Inbound**: `router.py` đọc header `X-Saga-ID` từ request, truyền vào `correlation.py` để gắn vào structured logging context cho toàn bộ vòng đời request.
- **Outbound**: Không có — TTS Service không gọi service nào khác (leaf node, không tham gia RabbitMQ).

## Internal Port Contract — `TTSEnginePort` (domain/ports.py)
```python
class TTSEnginePort(ABC):
    @abstractmethod
    def synthesize(self, text: str, language: str, output_path: str) -> float:
        """Sinh audio tại output_path, trả về duration_seconds."""
```

## Note — Contract Extension vs `component-methods.md`
Contract gốc trong `component-methods.md` (Application Design) là `{ text, language } -> { audio_path, duration_seconds }`. Low-Level Design mở rộng input thành `{ project_id, scene_index, text, language }` để hỗ trợ quy ước đường dẫn shared volume xác định tại Question 5 (idempotency theo `project_id`+`scene_index`, nhất quán với nguyên tắc idempotency toàn hệ thống ở `services.md`). `component-methods.md` cần được cập nhật để phản ánh thay đổi này sau khi Low-Level Design được duyệt.
