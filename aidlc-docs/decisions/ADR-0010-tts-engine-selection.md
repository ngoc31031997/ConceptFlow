# ADR-0010: TTS Engine Selection — Piper for MVP

## Status
Accepted

## Date
2026-08-05

## Stage
Low-Level Design (Unit 3: TTS Service)

## Context
FR4.1 requires an offline/open-source TTS engine (Coqui TTS, Piper, hoặc tương đương), FR4.2 requires bilingual Vietnamese/English support. ADR-0009 already constrained Unit 3 to Python 3.12 for library compatibility, but did not commit to a specific engine. Low-Level Design needed to decide which engine(s) to actually implement for the MVP.

## Options Considered
### Option A: Piper only (Chosen)
- What it is: Implement a single `PiperTTSAdapter` behind the `TTSEnginePort` abstraction. The port is fully defined so a `CoquiTTSAdapter` (or other) can be added later without touching domain/application code.
- Strengths: Lightweight, fast on CPU (no GPU dependency, fits a personal dev machine), pre-built voice models available for both Vietnamese and English, minimal implementation scope for MVP.
- Trade-offs: Piper's voice quality is generally less natural than Coqui's; if higher fidelity is needed later, a second adapter must be built (accepted, since the port already supports it).

### Option B: Coqui TTS only
- What it is: Implement a single `CoquiTTSAdapter` instead of Piper.
- Strengths: Generally more natural-sounding voices, larger model ecosystem.
- Trade-offs: Heavier runtime (larger models, slower CPU inference, often expects GPU for reasonable speed), higher setup/maintenance cost for an MVP running on a personal machine.

### Option C: Both engines implemented simultaneously
- What it is: Implement both `PiperTTSAdapter` and `CoquiTTSAdapter` at Unit 3 MVP time, selectable via config.
- Strengths: Maximum flexibility from day one.
- Trade-offs: Doubles adapter implementation/testing effort for a capability with no confirmed near-term need — over-engineering relative to current requirements.

## Decision
Chọn **Option A: Piper only** cho MVP, với `TTSEnginePort` được thiết kế đầy đủ để thêm engine khác (vd. Coqui) sau này mà không sửa domain/application.

## Rationale
Piper cân bằng tốt nhất giữa tốc độ phát triển MVP (chạy nhanh trên CPU, không cần GPU, phù hợp máy dev cá nhân) và việc đáp ứng đủ FR4.1/FR4.2 (voice model có sẵn cho cả tiếng Việt và tiếng Anh). Việc chỉ implement 1 adapter tránh over-engineering (Option C) trong khi vẫn giữ khả năng mở rộng theo NFR1 (Extensibility) nhờ port abstraction đã có sẵn — đúng tinh thần Hexagonal Architecture (ADR-0002).

## Consequences
- **Positive**: Thời gian phát triển Unit 3 ngắn hơn, không phụ thuộc GPU, dễ chạy trên Docker local.
- **Negative / Accepted Trade-offs**: Chất lượng giọng đọc có thể kém tự nhiên hơn Coqui — chấp nhận được cho MVP; cần đánh giá lại nếu chất lượng giọng đọc trở thành vấn đề thực tế.
- **Follow-ups**: Nếu cần đổi/thêm engine sau này, chỉ cần implement `CoquiTTSAdapter` mới trong `adapters/tts_engines/` và đổi config — không cần sửa `domain/`/`application/`.

## Related
- Design artifact: `aidlc-docs/construction/tts-service/low-level-design/module-structure.md`
- Related ADRs: Refines ADR-0009 (Python 3.12 for Unit 3, library constraint), consistent with ADR-0002 (Hexagonal/Ports & Adapters)
