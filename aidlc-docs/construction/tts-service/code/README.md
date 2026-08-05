# TTS Service — Code Summary

## Structure
Implements the module structure from `low-level-design/module-structure.md` (Hexagonal / Ports & Adapters):

```
services/tts/
├── domain/            # TTSEnginePort, SpeechRequest/SpeechResult, domain errors
├── application/        # SynthesizeSpeechUseCase
├── adapters/
│   ├── api/             # FastAPI router (POST /v1/tts/synthesize, GET /health), Pydantic schemas
│   ├── tts_engines/      # PiperTTSAdapter, static voice_registry (vi/en)
│   ├── storage/          # artifact_paths.py — shared-volume path convention + idempotency check
│   └── logging/          # correlation.py — X-Saga-ID propagation into log context
├── main.py              # Composition root (constructor injection, FastAPI lifespan loads voice models)
└── tests/
```

## Engine Implementation Note (Code Generation revision)
Low-Level Design/NFR Requirements assumed the `piper-tts` PyPI package. During Code Generation, that package's
`piper-phonemize` dependency turned out to have no prebuilt wheel available for this build environment. The
adapter (`piper_adapter.py`) was implemented to shell out to the standalone **Piper CLI binary** instead — the
module structure already anticipated this ("Piper CLI/binding") — with no change to `TTSEnginePort`, the API
contract, or any other approved design. `Dockerfile` installs the Piper release binary alongside the voice
models it already bundled.

## Key Behaviors Implemented
- **Idempotency** (Business Rule 4): before calling the engine, checks whether the shared-volume audio file
  already exists at `/shared/{project_id}/audio/{scene_index}_{language}.wav`; if so, reads its duration and
  returns immediately without re-synthesizing.
- **Threadpool + timeout** (NFR Design): Piper subprocess calls run in a `ThreadPoolExecutor`, bounded to 60s —
  a timeout or non-zero exit code becomes `TTSEngineError` → HTTP 502.
- **In-process voice model cache**: voice model paths are resolved once at startup (FastAPI lifespan) and held
  for the process lifetime — no per-request model reload.
- **Correlation**: `X-Saga-ID` request header is attached to every log line for the request.

## Tests
- `tests/application/test_synthesize_speech.py` — business rules (validation, idempotency, duration
  measurement, no text preprocessing) against a `FakeTTSEngine`.
- `tests/adapters/test_api.py` — API layer (success, 400s, 502, health readiness gating) via FastAPI
  `TestClient`.
- `tests/domain/test_models.py` — value object sanity checks.

Run with `cd services/tts && pip install -r requirements-dev.txt && pytest -q` (or under Python 3.12 via
Docker, since `piper`-free unit tests have no native/platform dependency beyond the interpreter version).
