# Code Generation Plan — Unit 3: TTS Service

## Unit Context
- **Stories**: B4, C2 (xem `unit-of-work-story-map.md`)
- **Dependencies**: Không có (độc lập, không tham gia RabbitMQ)
- **Interfaces**: `POST /v1/tts/synthesize` (REST, gọi bởi Rendering Service — Unit 5, chưa phát triển)
- **Owned entities**: Không có database; owned artifact: file audio trong shared volume `/shared`

## Coding Standards (Step 3.5 — đã xác nhận ở Low-Level Design, nhất quán với Unit 2)
- **Naming**: Python snake_case cho function/variable, PascalCase cho class (chuẩn PEP8)
- **SOLID**: Bắt buộc — đặc biệt Dependency Inversion (`SynthesizeSpeechUseCase` phụ thuộc `TTSEnginePort` abstraction, không phụ thuộc `PiperTTSAdapter` cụ thể)
- **Documentation**: Python docstring (Google style) cho public class/method — giải thích WHY, không lặp lại WHAT
- **Linting**: `ruff`

## Steps

- [x] **Step 1 — Project Structure Setup**: Tạo `services/tts/` theo `module-structure.md` (domain/, application/, adapters/, main.py, tests/), `requirements.txt`, `Dockerfile`
- [x] **Step 2 — Business Logic Generation (domain/)**: `ports.py` (`TTSEnginePort`), `models.py` (`SpeechRequest`, `SpeechResult`), `errors.py` (`EmptyTextError`, `UnsupportedLanguageError`, `TTSEngineError`)
- [x] **Step 3 — Business Logic Generation (application/)**: `synthesize_speech.py` (`SynthesizeSpeechUseCase`) — theo `business-logic-model.md` (validate → path → idempotency check → engine call → duration → result)
- [x] **Step 4 — Business Logic Unit Testing**: `tests/domain/`, `tests/application/` — dùng `FakeTTSEngine` (implements `TTSEnginePort`) và temp directory giả lập shared volume, cover mọi business rule (Rule 1-6)
- [x] **Step 5 — Storage Adapter Generation**: `adapters/storage/artifact_paths.py` (đường dẫn quy ước + kiểm tra tồn tại + đọc duration từ `.wav` có sẵn)
- [x] **Step 6 — TTS Engine Adapter Generation**: `adapters/tts_engines/voice_registry.py` (static mapping vi/en), `adapters/tts_engines/piper_adapter.py` (`PiperTTSAdapter` implements `TTSEnginePort` — threadpool, 60s timeout, in-process voice model cache, đo duration từ `.wav` output). **Revision during generation**: dùng Piper CLI binary qua `subprocess` thay vì package `piper-tts` (dependency `piper-phonemize` không có wheel sẵn cho môi trường build) — xem `code/README.md`
- [x] **Step 7 — Logging Adapter Generation**: `adapters/logging/correlation.py` (đọc `X-Saga-ID`, gắn vào structured log context)
- [x] **Step 8 — API Layer Generation**: `adapters/api/router.py` (`POST /v1/tts/synthesize`, `GET /health` — gated on model-loaded), `adapters/api/schemas.py` (Pydantic request/response models, error mapping 400/502)
- [x] **Step 9 — API Layer Unit Testing**: `tests/adapters/test_api.py` — FastAPI `TestClient`, dùng `FakeTTSEngine` qua dependency override
- [x] **Step 10 — Composition Root**: `main.py` — wiring toàn bộ (theo `dependency-injection.md`), load voice model lúc startup (FastAPI lifespan event)
- [x] **Step 11 — Documentation Generation**: Cập nhật `README.md` gốc (thêm mục cho TTS Service), tạo `aidlc-docs/construction/tts-service/code/README.md`
- [x] **Step 12 — Deployment Artifacts**: Thêm service `tts` + named volume `shared_artifacts` vào `docker-compose.yml` gốc (theo `deployment-architecture.md`)

**Không áp dụng**: Repository Layer (không có database), Frontend Components, Database Migration Scripts, Messaging Layer (không tham gia RabbitMQ).

**Note về Piper model thực tế**: Việc tải file voice model `.onnx` thật (network access lúc build Docker image, theo `deployment-architecture.md`) nằm ngoài phạm vi code generation — Dockerfile sẽ chứa lệnh `curl` tới URL model cụ thể, nhưng việc build/pull model thật cần chạy trên máy có network khi build image, không phải lúc sinh code. Unit test dùng `FakeTTSEngine` nên không phụ thuộc model thật.
