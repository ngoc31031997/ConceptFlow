# NFR Requirements — Unit 3: TTS Service

## Performance
- Piper synthesis chạy trong threadpool (không block FastAPI async event loop) — route handler định nghĩa `def` thường hoặc dùng `run_in_threadpool` tường minh.
- Timeout nội bộ cho mỗi lần synthesize: **60 giây**. Nếu vượt timeout → raise `TTSEngineError` → HTTP 502 (theo Business Rule đã thiết kế ở Functional Design), không dựa vào timeout mặc định của HTTP client phía Rendering Service.
- Piper voice model (`vi`, `en`) được load 1 lần lúc service khởi động, giữ trong memory suốt vòng đời process (xem Caching bên dưới) để giảm latency mỗi request.

## Availability
Chấp nhận unavailability tạm thời — không cần multi-instance/failover (nhất quán toàn hệ thống, local Docker single-machine). Nếu TTS Service down khi Rendering Service gọi, lỗi connection propagate thành `rendering_failed` → Orchestrator cho phép retry theo compensating action đã thiết kế (`services.md`).

## Security
Validate input qua Pydantic schema (FastAPI mặc định). Không cần auth/rate-limit riêng — chỉ Rendering Service (nội bộ, cùng Docker network) gọi tới, không expose ra ngoài (Security Baseline extension đã tắt theo `aidlc-state.md`).

## Messaging & Event Participation
N/A — Unit 3 không tham gia RabbitMQ (xác nhận từ `unit-of-work.md`), chỉ REST.

## Distributed Transaction Participation (Saga)
**Vai trò**: Participant gián tiếp trong bước `render_scenes` — không phải Saga coordinator, không tự publish/consume AMQP event; chỉ là 1 REST call bên trong logic của Rendering Service. **Compensating action**: Không cần — stateless, side-effect duy nhất (ghi file audio) đã idempotent (Business Rule 4, Functional Design), nên retry bước `render_scenes` không cần rollback gì ở TTS Service.

## Caching Requirements
Piper voice model (`vi`, `en`) load 1 lần lúc khởi động (composition root `main.py`), giữ in-memory suốt vòng đời process. Không cần cache tầng nào khác — idempotency qua file audio đã có (Functional Design) đủ để tránh tính toán lại.

## Tech Stack Consistency
Xác nhận Python 3.12 + FastAPI, khớp `technology-direction.md` và ADR-0009 (ràng buộc thư viện TTS Python-first). Không lệch hướng cho unit này.
