# Dependency Injection — Unit 3: TTS Service

**Revision (2026-08-07, ADR-0014, ADR-0013)**: FastAPI `Depends()` wiring replaced by AMQP consumer wiring (mirrors Unit 2/Unit 4); `InboxRepository`/`OutboxRepository`/`OutboxRelay` added.

## Mechanism
Constructor injection thủ công (không dùng DI container/framework) — không đổi.

## What Gets Injected vs Constructed Directly
- **Injected (abstraction)**: `TTSEnginePort` — `SynthesizeSpeechUseCase` nhận instance implement `TTSEnginePort` qua constructor (không đổi).
- **Constructed directly**: Value objects, `artifact_paths.py` helper (không đổi); `InboxRepository`/`OutboxRepository`/`OutboxRelay` (Postgres, không cần abstraction, mirror Unit 2/Unit 4).

## Composition Root
`main.py`:
1. Khởi tạo `PiperTTSAdapter()`.
2. Khởi tạo `SynthesizeSpeechUseCase(engine=piper_adapter)`, sau đó `SynthesizeSpeechBatchUseCase(single=synthesize_speech_use_case)`.
3. Kết nối PostgreSQL (`adapters/persistence/db.py`), khởi tạo `InboxRepository`, `OutboxRepository`.
4. Kết nối RabbitMQ, wire `SynthesizeSpeechCommandHandler(batch_use_case, pool, inbox, outbox)`, đăng ký consumer cho queue `tts.commands`.
5. Khởi động `OutboxRelay` như background task.

## Wiring Diagram
```
main.py
  ├── PiperTTSAdapter (implements TTSEnginePort)
  │     └── injected into → SynthesizeSpeechUseCase
  │           └── injected into → SynthesizeSpeechBatchUseCase
  ├── InboxRepository, OutboxRepository (Postgres, ADR-0013)
  │     └── injected into → SynthesizeSpeechCommandHandler (AMQP consumer)
  │           └── consumes command "synthesize_speech" (queue tts.commands)
  │           └── ghi event "speech_synthesized"/"synthesis_failed" vào Outbox
  └── OutboxRelay (background task)
        └── poll Outbox chưa publish → publish qua producer.py → đánh dấu published_at
```
