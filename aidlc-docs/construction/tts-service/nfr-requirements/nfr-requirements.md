# NFR Requirements — Unit 3: TTS Service

**Revision (2026-08-07, ADR-0014, ADR-0013)**: Messaging & Saga Participation and Tech Stack sections updated below — TTS is now message-driven with its own Saga step. Performance/Availability/Security/Caching sections carry over unchanged (threadpool + 60s timeout logic lives inside the same `PiperTTSAdapter`, just invoked from an AMQP consumer instead of a FastAPI route handler).

## Performance
- Piper synthesis chạy trong threadpool (không block asyncio event loop) — không đổi.
- Timeout nội bộ cho mỗi lần synthesize: **60 giây**. Nếu vượt timeout → raise `TTSEngineError` → mapped thành `synthesis_failed` event (không còn HTTP 502, không còn REST).
- Piper voice model (`vi`, `en`) được load 1 lần lúc service khởi động — không đổi.

## Availability
Chấp nhận unavailability tạm thời — không cần multi-instance/failover (không đổi). Nếu TTS Service down khi Orchestrator gửi command, message ở lại queue `tts.commands` (RabbitMQ durability) cho tới khi service khởi động lại — khác với REST trước đây (nơi lỗi connection ngay lập tức propagate thành `rendering_failed`); nay Orchestrator có thể set timeout riêng cho bước Saga để phát hiện TTS Service không phản hồi.

## Security
Validate input trong `SynthesizeSpeechUseCase` (không đổi — domain error, không phải Pydantic/FastAPI validation nữa). Không cần auth/rate-limit — chỉ Orchestrator (nội bộ, qua RabbitMQ) gửi command.

## Messaging & Event Participation
**Revised**: Consumer của `synthesize_speech` (queue `tts.commands`), producer của `speech_synthesized`/`synthesis_failed` (→ `orchestrator.events`, qua Outbox). Kế thừa delivery guarantee at-least-once từ Unit 1; idempotency 2 tầng — message-level (Inbox, ADR-0013) + artifact-level (file check, Business Rule 4, không đổi).

## Distributed Transaction Participation (Saga)
**Revised — vai trò**: **Participant trực tiếp** cho bước Saga độc lập "Synthesize Speech" (ADR-0014) — không còn "gián tiếp bên trong render_scenes". **Compensating action**: Vẫn không cần rollback (stateless, side-effect ghi file audio đã idempotent) — nhưng nay Orchestrator TỰ retry command `synthesize_speech` (thay vì Rendering Service tự quyết định retry REST call như trước).

## Caching Requirements
Không đổi — Piper voice model in-memory, load 1 lần lúc khởi động.

## Tech Stack Consistency
**Revised**: Python 3.12 (không đổi, ADR-0009). FastAPI KHÔNG còn cần thiết cho luồng chính (không còn REST) — có thể giữ lại tối thiểu cho `/health` endpoint (quyết định cụ thể ở Infrastructure Design) hoặc bỏ hẳn, dùng health check kiểu khác (vd. script kiểm tra kết nối RabbitMQ/Postgres). `aio-pika` + `asyncpg` thêm vào tech stack (mirror Unit 2/Unit 4).
