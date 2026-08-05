# NFR Design Patterns — Unit 3: TTS Service

## CRUD vs CQRS
**N/A** — Không có database, không có data model nghiệp vụ cần persist ngoài file audio (blob, không phải structured data cần query). Unit hoàn toàn stateless.

## Resilience Pattern
Không retry nội bộ trong TTS Service. Một lần gọi Piper thất bại (crash/timeout sau 60s, theo NFR Requirements) → trả lỗi ngay (`TTSEngineError` → HTTP 502), để Rendering Service (qua Saga compensating action ở `services.md`) quyết định có retry toàn bộ request hay không. Tránh 2 tầng retry chồng lên nhau gây khó đoán tổng thời gian chờ.

## Caching Pattern
In-process cache (không phải Redis/distributed) cho Piper voice model:
- **Placement**: biến module-level trong `adapters/tts_engines/piper_adapter.py`, load 1 lần lúc composition root (`main.py`) khởi tạo `PiperTTSAdapter`.
- **Key design**: `language` (`"vi"`/`"en"`) → voice model instance đã load.
- **TTL/Invalidation**: Không có — model không đổi trong vòng đời process; đổi model yêu cầu restart service.
- **Strategy**: Không phải cache-aside/write-through — load eager 1 lần tại startup, không load lazy theo request.

## Idempotency Pattern
Idempotency dựa trên kiểm tra tồn tại file tại đường dẫn shared volume quy ước (`/shared/{project_id}/audio/{scene_index}_{language}.wav`, Functional Design Rule 4). Không cần lock/race-condition handling ở MVP — Rendering Service gọi tuần tự từng scene (không có 2 request trùng `project_id`+`scene_index` đồng thời trong luồng bình thường); nếu race condition hiếm gặp xảy ra, rủi ro tối đa là ghi đè file với nội dung giống hệt (vô hại).

## Saga Pattern
Participant gián tiếp trong bước `render_scenes` (không phải coordinator, không tự publish/consume AMQP event). Không có compensating action — stateless, side-effect duy nhất (ghi file audio) đã idempotent.

## Event-Driven Design
**N/A** — Unit 3 không publish/consume event nào (không tham gia RabbitMQ theo `unit-of-work.md`).

## Inbox/Outbox Pattern
**N/A** — Không có database transaction cần đồng bộ với việc publish event (không publish event nào).

## Security Pattern
Input validation qua Pydantic schema (FastAPI) cho `project_id`, `scene_index`, `text`, `language`. Không auth/rate-limit riêng — chỉ Rendering Service gọi nội bộ, cùng Docker network (Security Baseline extension tắt).
