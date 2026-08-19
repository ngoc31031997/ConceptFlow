# NFR Design Patterns — Unit 3: TTS Service

**Revision (2026-08-07, ADR-0014, ADR-0013)**: Saga Pattern, Event-Driven Design, and Inbox/Outbox Pattern below are superseded — TTS Service is now message-driven with its own Saga step and PostgreSQL-backed Inbox/Outbox. Resilience/Caching/Idempotency(-by-file) patterns carry over unchanged.

## CRUD vs CQRS
**Revised**: Simple CRUD on `outbox_events`/`processed_messages` (messaging-plumbing tables only, not business data — same as Unit 2/Unit 4, ADR-0013). Audio files remain outside the DB (blob on shared volume, unchanged).

## Resilience Pattern
Không retry nội bộ trong TTS Service — không đổi về ý tưởng, chỉ đổi kết quả lỗi: 1 lần gọi Piper thất bại (timeout 60s) → `TTSEngineError` → event `synthesis_failed` (không còn HTTP 502), để Orchestrator (qua Saga compensating action) quyết định retry command `synthesize_speech`.

## Caching Pattern
Không đổi — in-process cache cho Piper voice model, load eager lúc startup.

## Idempotency Pattern
**Bổ sung 2 tầng** (trước đây chỉ có artifact-level):
- **Artifact-level** (không đổi): kiểm tra file `.wav` tồn tại tại đường dẫn shared volume trước khi synthesize lại (Business Rule 4).
- **Message-level** (MỚI, ADR-0013): `InboxRepository` — dedupe `message_id` bền vững qua bảng `processed_messages`, thay cho việc trước đây TTS Service hoàn toàn không cần dedupe message nào (vì không tham gia RabbitMQ).

## Saga Pattern
**Superseded — see Revision above.** Participant TRỰC TIẾP cho bước Saga độc lập "Synthesize Speech" (ADR-0014) — publish `speech_synthesized`/`synthesis_failed`. Không có compensating action (không đổi — stateless, side-effect ghi file audio đã idempotent).

## Event-Driven Design
**Superseded.** TTS Service nay consume command `synthesize_speech` (queue `tts.commands`) và publish event `speech_synthesized`/`synthesis_failed` (→ `orchestrator.events`) — là integration event, theo cùng envelope/versioning convention của toàn hệ thống (`messaging-design.md`, Unit 1).

## Inbox/Outbox Pattern
**Superseded.** PostgreSQL-backed (`tts-db`, ADR-0013), giống hệt kiến trúc ở Unit 2/Unit 4:
- **Outbox**: consumer ghi `speech_synthesized`/`synthesis_failed` vào `outbox_events` trong CÙNG transaction với việc mark Inbox — atomicity giữa "đã xử lý command" và "đã ghi nhận event kết quả."
- **Relay**: `OutboxRelay` polling task publish event chưa gửi.

## Security Pattern
**Revised**: Validate input trong domain layer (`SynthesizeSpeechUseCase`, đã có sẵn — không phải Pydantic/FastAPI validation nữa, vì không còn REST). Không auth/rate-limit riêng — chỉ Orchestrator gửi command qua RabbitMQ nội bộ.
