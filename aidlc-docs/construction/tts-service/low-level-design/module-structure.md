# Module Structure — Unit 3: TTS Service

## Layering (Hexagonal / Ports & Adapters — ADR-0002)

```
services/tts/
├── domain/
│   ├── ports.py               # TTSEnginePort (abstract interface)
│   ├── models.py               # SpeechRequest, SpeechResult (value objects, pure Python)
│   └── errors.py               # Domain-specific exceptions (UnsupportedLanguageError, TTSEngineError)
├── application/
│   └── synthesize_speech.py    # SynthesizeSpeechUseCase
├── adapters/
│   ├── api/
│   │   ├── router.py            # FastAPI router, POST /v1/tts/synthesize (ADR-0008)
│   │   └── schemas.py           # Pydantic request/response models
│   ├── tts_engines/
│   │   ├── voice_registry.py    # Static language -> voice model path mapping
│   │   └── piper_adapter.py     # PiperTTSAdapter implements TTSEnginePort (ADR-0010)
│   ├── storage/
│   │   └── artifact_paths.py    # Shared-volume path convention helper (project_id/scene_index -> path)
│   └── logging/
│       └── correlation.py       # Correlation ID (X-Saga-ID) injection into log context
├── main.py                      # Composition root — wiring, FastAPI app
└── tests/
    ├── domain/
    ├── application/
    └── adapters/
```

## Dependency Direction
`adapters/` → `application/` → `domain/`. `domain/` không import bất kỳ thứ gì từ `adapters/` hay `application/`, không import FastAPI/Piper. `application/` chỉ phụ thuộc `domain/` (qua port interface `TTSEnginePort`), không biết chi tiết FastAPI/Piper cụ thể nào. `adapters/tts_engines/piper_adapter.py` implement `domain/ports.py::TTSEnginePort` — cơ chế pluggable cho phép thêm engine khác (Coqui, …) sau này mà không sửa `domain/`/`application/` (ADR-0010).

## Module Responsibilities

| Module | Responsibility |
|---|---|
| `domain/ports.py` | Định nghĩa `TTSEnginePort` (abstract: `synthesize(text: str, language: str, output_path: str) -> float` — trả về `duration_seconds`) |
| `domain/models.py` | `SpeechRequest` (value object: `project_id`, `scene_index`, `text`, `language`), `SpeechResult` (`audio_path`, `duration_seconds`) |
| `domain/errors.py` | `UnsupportedLanguageError` (map sang HTTP 400), `TTSEngineError` (map sang HTTP 502) |
| `application/synthesize_speech.py` | `SynthesizeSpeechUseCase(engine: TTSEnginePort)` — tính đường dẫn artifact (qua `artifact_paths.py`), kiểm tra idempotency (file đã tồn tại → trả kết quả có sẵn), gọi `engine.synthesize(...)` nếu chưa có, trả về `SpeechResult` |
| `adapters/api/router.py` | FastAPI route `POST /v1/tts/synthesize` — validate input, đọc header `X-Saga-ID`, gọi `SynthesizeSpeechUseCase`, map domain error → HTTP status |
| `adapters/tts_engines/voice_registry.py` | Static mapping `{"vi": "<piper-vi-model-path>", "en": "<piper-en-model-path>"}`, raise `UnsupportedLanguageError` nếu language không có trong mapping |
| `adapters/tts_engines/piper_adapter.py` | `PiperTTSAdapter` — implement `TTSEnginePort`, gọi Piper CLI/binding với voice model từ `voice_registry.py`, đo thời lượng file audio sinh ra |
| `adapters/storage/artifact_paths.py` | Sinh đường dẫn quy ước `/shared/{project_id}/audio/{scene_index}_{language}.wav`, kiểm tra file tồn tại (idempotency) |
| `adapters/logging/correlation.py` | Đọc header `X-Saga-ID`, đưa vào log context (structured logging) cho mọi log line trong request |
