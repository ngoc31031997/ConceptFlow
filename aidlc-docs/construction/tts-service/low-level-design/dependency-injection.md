# Dependency Injection — Unit 3: TTS Service

## Mechanism
Constructor injection thủ công (theo `architectural-style.md` HLD, nhất quán với Unit 2), không dùng DI container/framework.

## What Gets Injected vs Constructed Directly
- **Injected (abstraction)**: `TTSEnginePort` — `SynthesizeSpeechUseCase` nhận instance implement `TTSEnginePort` qua constructor. Cho phép thay `PiperTTSAdapter` bằng adapter khác (vd. `CoquiTTSAdapter` sau này) mà không sửa `application/`.
- **Constructed directly**: Value objects (`SpeechRequest`, `SpeechResult`), helper thuần túy (`artifact_paths.py` path computation) — không cần abstraction vì không có lý do thay thế implementation.

## Composition Root
`main.py` là composition root:
1. Đọc config (biến môi trường, vd. `TTS_ENGINE=piper` — hiện tại chỉ hỗ trợ giá trị này theo ADR-0010) để chọn adapter cụ thể.
2. Khởi tạo `PiperTTSAdapter()` (đọc `voice_registry.py` để biết đường dẫn model).
3. Khởi tạo `SynthesizeSpeechUseCase(engine=piper_adapter)`.
4. Wire `SynthesizeSpeechUseCase` vào FastAPI app qua `Depends()` trong `adapters/api/router.py`.

## Wiring Diagram
```
main.py
  └── PiperTTSAdapter (implements TTSEnginePort)
        └── injected into → SynthesizeSpeechUseCase
              └── injected into → FastAPI router (Depends())
                    └── POST /v1/tts/synthesize
```
