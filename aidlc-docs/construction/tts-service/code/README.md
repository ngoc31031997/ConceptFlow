# TTS Service — Code Summary

**Revision (2026-08-07, ADR-0014, ADR-0013)**: TTS Service converted from REST (called synchronously by Rendering Service) to message-driven (own Saga step, "Synthesize Speech"), and retrofitted with PostgreSQL-backed Inbox/Outbox — part of a system-wide learning exercise for the microservices/orchestrator pattern. `adapters/api/` removed entirely; replaced by `adapters/messaging/` + `adapters/persistence/`.

## Structure
Implements the module structure from `low-level-design/module-structure.md` (Hexagonal / Ports & Adapters):

```
services/tts/
├── domain/              # TTSEnginePort, SpeechRequest/SpeechResult, domain errors — unchanged
├── application/
│   ├── synthesize_speech.py        # SynthesizeSpeechUseCase — unchanged
│   └── synthesize_speech_batch.py  # NEW — SynthesizeSpeechBatchUseCase (fail-fast, mirrors Unit 2)
├── adapters/
│   ├── messaging/         # NEW — replaces adapters/api/: consumer.py (synthesize_speech), producer.py (envelope builders)
│   ├── persistence/       # NEW (ADR-0013) — db.py, inbox.py, outbox.py, relay.py (identical shape to Unit 2)
│   ├── tts_engines/       # PiperTTSAdapter, static voice_registry (vi/en) — unchanged
│   ├── storage/           # artifact_paths.py — unchanged
│   └── logging/           # correlation.py — saga_id now from AMQP envelope, not HTTP header
├── main.py               # Composition root — plain asyncio entrypoint (no more FastAPI), writes /tmp/ready sentinel
└── tests/
```

## Engine Implementation Note (from original Code Generation, still applies)
`piper-tts`'s PyPI package `piper-phonemize` dependency has no prebuilt wheel for this build environment, so
`piper_adapter.py` shells out to the standalone **Piper CLI binary** instead — no change to `TTSEnginePort`.

## Key Behaviors Implemented
- **Idempotency, 2 tầng** (artifact-level unchanged; message-level new): shared-volume file-exists check
  (Business Rule 4) is unchanged; `InboxRepository` adds durable message-id dedupe (ADR-0013), replacing the
  "N/A — not on RabbitMQ" status TTS Service had before.
- **Threadpool + timeout** (unchanged): Piper subprocess calls run in a `ThreadPoolExecutor`, bounded to 60s —
  a timeout or non-zero exit code becomes `TTSEngineError` → now a `synthesis_failed` event instead of HTTP 502.
- **In-process voice model cache** (unchanged): loaded once at `PiperTTSAdapter` construction in `main.py`.
- **Correlation**: `saga_id` now comes from the AMQP command envelope (`adapters/logging/correlation.py`), not
  an `X-Saga-ID` HTTP header.
- **Outbox** (NEW): consumer writes `speech_synthesized`/`synthesis_failed` to `outbox_events` in the same DB
  transaction as the Inbox mark; `OutboxRelay` (background polling task, ~1s) is the only thing that actually
  publishes to RabbitMQ.

## Tests
- `tests/application/test_synthesize_speech.py` — unchanged (business rules against `FakeTTSEngine`).
- `tests/application/test_synthesize_speech_batch.py` — NEW — fail-fast batch semantics.
- `tests/adapters/test_messaging.py` — NEW — consumer + Inbox/Outbox (replaces the old `test_api.py`).
- `tests/adapters/test_persistence.py`, `test_relay.py` — NEW — Inbox/Outbox/Relay unit tests (identical in
  shape to Unit 2's, via `tests/adapters/fake_postgres.py`).
- `tests/domain/test_models.py` — unchanged.

22 tests passing, `ruff check` clean — verified under Python 3.12 via Docker (`docker run python:3.12-slim`).
`main.py` import-sanity-checked with `DATABASE_URL`/`RABBITMQ_URL` env vars set (no live Postgres/RabbitMQ
connection attempted at import time).
