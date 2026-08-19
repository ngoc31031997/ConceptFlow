# Interface Contracts — Unit 3: TTS Service

**Revision (2026-08-07, ADR-0014)**: `POST /v1/tts/synthesize` (REST) is replaced with an AMQP consumer/producer pair — TTS Service is now its own Saga step, no longer called synchronously by Rendering Service. The old REST section is kept below, struck through in spirit (removed), replaced by the AMQP contract.

## AMQP Consumer: command `synthesize_speech`
Queue: `tts.commands` (theo `unit-of-work.md`, thêm ở ADR-0014). Dispatched bởi Orchestrator như bước Saga độc lập ("Synthesize Speech", `services.md`), batch theo project (mirror cách `classify_scenes` xử lý ở Unit 2).

**Command payload** (envelope chuẩn, theo `messaging-design.md` Unit 1):
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "string",
  "schema_version": "1.0",
  "timestamp": "ISO-8601",
  "payload": {
    "scenes": [
      { "scene_index": 0, "narration_text": "string", "language": "vi" }
    ]
  }
}
```

## AMQP Producer: event `speech_synthesized` / `synthesis_failed`
Publish tới `orchestrator.events` (qua Outbox + `OutboxRelay`, ADR-0013 — không publish trực tiếp từ consumer).

**`speech_synthesized`** (thành công — fail-fast batch, mirror `ClassifyScenesBatchUseCase`):
```json
{
  "payload": {
    "event_type": "speech_synthesized",
    "scenes": [
      { "scene_index": 0, "audio_path": "/shared/{project_id}/audio/0_vi.wav", "duration_seconds": 12.5 }
    ]
  }
}
```

**`synthesis_failed`**:
```json
{
  "payload": {
    "event_type": "synthesis_failed",
    "error_message": "string"
  }
}
```
`error_message` phân biệt permanent (`empty_text`/`unsupported_language` — lỗi input, không nên retry) vs transient (`tts_engine_failure` — Piper lỗi/timeout, Rendering/Orchestrator có thể retry) qua tiền tố trong message, theo đúng phân loại lỗi đã có ở `business-rules.md` (Functional Design) — HTTP status code 400/502 không còn áp dụng (không còn REST), nhưng phân loại permanent/transient vẫn giữ nguyên ý nghĩa cho compensating action ở `services.md`.

## Delivery Guarantee & Idempotency (ADR-0013)
At-least-once (kế thừa Unit 1). Inbox (`processed_messages`) dedupe `message_id` bền vững; Outbox (`outbox_events`) ghi event trong cùng transaction với Inbox mark; `OutboxRelay` publish bất đồng bộ (poll ~1s). Idempotency bổ sung ở tầng business logic: `SynthesizeSpeechUseCase` vẫn kiểm tra file `.wav` đã tồn tại trước khi synthesize lại (Business Rule 4, Functional Design — không đổi) — 2 tầng idempotency độc lập (message-level qua Inbox, artifact-level qua file check).

## Correlation ID
`saga_id` từ envelope AMQP được gắn vào mọi log line qua `adapters/logging/correlation.py`. Không còn header `X-Saga-ID` (không còn REST).

## Internal Port Contract — `TTSEnginePort` (domain/ports.py)
**Không đổi**:
```python
class TTSEnginePort(ABC):
    @abstractmethod
    def synthesize(self, text: str, language: str, output_path: str) -> float:
        """Sinh audio tại output_path, trả về duration_seconds."""
```

## Note — Contract History
Contract gốc `component-methods.md` (Application Design): `{ text, language } -> { audio_path, duration_seconds }` REST. Sau đó mở rộng thành REST với `{ project_id, scene_index, text, language }` (Low-Level Design ban đầu). Nay (ADR-0014) đổi hẳn sang AMQP batch theo project như mô tả ở trên — `component-methods.md` đã cập nhật theo revision này.
