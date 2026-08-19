# Module Structure — Unit 3: TTS Service

**Revision (2026-08-07, ADR-0014, ADR-0013)**: `adapters/api/` (FastAPI, REST) thay bằng `adapters/messaging/` (AMQP consumer/producer) — TTS Service không còn REST-only, nay là bước Saga độc lập (ADR-0014). Đồng thời thêm `adapters/persistence/` cho Inbox/Outbox pattern (ADR-0013), theo đúng khuôn mẫu đã dùng ở Unit 2/Unit 4. `domain/`, `application/synthesize_speech.py`, `adapters/tts_engines/`, `adapters/storage/` **không đổi** — business logic (validate, idempotency-by-file, đo duration) độc lập với transport.

## Layering (Hexagonal / Ports & Adapters — ADR-0002)

```
services/tts/
├── domain/
│   ├── ports.py               # TTSEnginePort (abstract interface) — không đổi
│   ├── models.py               # SpeechRequest, SpeechResult (value objects) — không đổi
│   └── errors.py               # EmptyTextError, UnsupportedLanguageError, TTSEngineError — không đổi
├── application/
│   └── synthesize_speech.py    # SynthesizeSpeechUseCase — không đổi
│   └── synthesize_speech_batch.py  # MỚI: SynthesizeSpeechBatchUseCase (fail-fast, mirror Unit 2's ClassifyScenesBatchUseCase)
├── adapters/
│   ├── messaging/               # MỚI — thay adapters/api/
│   │   ├── consumer.py           # AMQP consumer cho command synthesize_speech (queue tts.commands)
│   │   └── producer.py           # Envelope builders cho speech_synthesized/synthesis_failed (không publish trực tiếp)
│   ├── persistence/              # MỚI (ADR-0013, mirror Unit 2/Unit 4)
│   │   ├── db.py                  # PostgreSQL pool + schema bootstrap
│   │   ├── inbox.py               # InboxRepository — durable dedupe
│   │   ├── outbox.py              # OutboxRepository — transactional event enqueue
│   │   └── relay.py               # OutboxRelay — polling publisher
│   ├── tts_engines/
│   │   ├── voice_registry.py    # Static language -> voice model path mapping — không đổi
│   │   └── piper_adapter.py     # PiperTTSAdapter implements TTSEnginePort (ADR-0010) — không đổi
│   ├── storage/
│   │   └── artifact_paths.py    # Shared-volume path convention helper — không đổi
│   └── logging/
│       └── correlation.py       # saga_id injection vào log context (từ envelope AMQP, không còn header HTTP)
├── main.py                      # Composition root — wiring AMQP consumer + OutboxRelay (không còn FastAPI app, trừ health check tối thiểu — xem Infrastructure Design)
└── tests/
    ├── domain/
    ├── application/
    └── adapters/
```

## Dependency Direction
Không đổi: `adapters/` → `application/` → `domain/`. `adapters/messaging/consumer.py` gọi `SynthesizeSpeechBatchUseCase` (application layer, phụ thuộc `TTSEnginePort` abstraction) — không biết chi tiết Piper/RabbitMQ/Postgres cụ thể nào.

## Module Responsibilities (chỉ liệt kê phần thay đổi/mới — phần còn lại giữ nguyên như trước)

| Module | Responsibility |
|---|---|
| `application/synthesize_speech_batch.py` | `SynthesizeSpeechBatchUseCase(single: SynthesizeSpeechUseCase)` — xử lý toàn bộ scene trong 1 project, fail-fast ở scene lỗi đầu tiên (mirror `ClassifyScenesBatchUseCase`, Unit 2) |
| `adapters/messaging/consumer.py` | Consume `synthesize_speech` (queue `tts.commands`); trong 1 DB transaction: kiểm tra Inbox, gọi `SynthesizeSpeechBatchUseCase`, ghi kết quả vào Outbox (`speech_synthesized`/`synthesis_failed`), mark Inbox — commit, ack |
| `adapters/messaging/producer.py` | Envelope builders `success_envelope`/`failure_envelope` — publish thực tế do `OutboxRelay` đảm nhiệm |
| `adapters/persistence/*` | Giống hệt Unit 2/Unit 4 (ADR-0013) — schema `outbox_events`/`processed_messages` |
| `adapters/logging/correlation.py` | Đọc `saga_id` từ envelope AMQP (không còn header `X-Saga-ID` vì không còn REST) |
