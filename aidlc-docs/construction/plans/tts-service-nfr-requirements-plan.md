# NFR Requirements Plan — Unit 3: TTS Service

## Unit Context
- Stateless REST service, CPU-bound work (Piper synthesis), gọi đồng bộ bởi Rendering Service trong 1 bước Saga (`render_scenes`), không tham gia RabbitMQ

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-requirements.md`
- [ ] Tạo `tech-stack-decisions.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Tech Stack Consistency (BẮT BUỘC)
`technology-direction.md`/ADR-0003 chọn Python/FastAPI mặc định toàn hệ thống. ADR-0009 (polyglot có chọn lọc) đã xác nhận Unit 3 giữ **Python 3.12** do ràng buộc thư viện TTS (Piper/Coqui là Python-first).

A) 💡 Suggested: Xác nhận Python 3.12 + FastAPI cho Unit 3, khớp `technology-direction.md`/ADR-0009 — không có lý do kỹ thuật hay sư phạm để lệch hướng
   - ✅ Strengths: nhất quán với quyết định đã có, đúng ràng buộc kỹ thuật (thư viện Piper)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 2: Performance — Xử lý CPU-bound đồng bộ trong FastAPI
Piper synthesis là tác vụ CPU-bound, chạy trong vài giây tùy độ dài text. FastAPI mặc định dùng async event loop — nếu gọi Piper trực tiếp trong async route handler (blocking call), sẽ block toàn bộ event loop, ảnh hưởng khả năng phục vụ request khác.

A) 💡 Suggested: Chạy Piper synthesis trong threadpool (FastAPI's `run_in_threadpool`, hoặc route handler định nghĩa là `def` thường thay vì `async def` để FastAPI tự đưa vào threadpool) — tránh block event loop, dù MVP chỉ có 1 client nội bộ (Rendering Service) gọi tuần tự nên đây chủ yếu là best practice hơn là yêu cầu hiệu năng cấp bách
   - ✅ Strengths: đúng pattern chuẩn cho CPU-bound work trong FastAPI, không cần thay đổi lớn nếu sau này có nhiều client gọi song song
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 3: Performance — Timeout cho request
Rendering Service gọi đồng bộ `POST /v1/tts/synthesize`. Nếu Piper bị treo (hang) vì lý do bất thường, cần timeout để tránh Rendering Service chờ vô hạn.

A) 💡 Suggested: Đặt timeout nội bộ trong `PiperTTSAdapter` (vd. 30 giây — đủ dư cho scene narration text thông thường ở quy mô video giáo dục ngắn) — nếu vượt timeout, raise `TTSEngineError` (mapped sang HTTP 502, theo Business Rule đã thiết kế), không dựa vào timeout mặc định của HTTP client phía Rendering Service (đảm bảo lỗi được phân loại đúng ở nguồn)
   - ✅ Strengths: kiểm soát lỗi timeout tập trung ở TTS Service, trả lỗi rõ ràng thay vì để Rendering Service tự đoán nguyên nhân khi connection timeout
   - ⚠️ Trade-offs: 30s là giá trị ước lượng, có thể cần điều chỉnh sau khi có dữ liệu thực tế

B) Other (please describe after [Answer]: tag below)

[Answer]: 60s đi

### Question 4: Availability
Yêu cầu uptime/failover cho Unit 3?

A) 💡 Suggested: Chấp nhận unavailability tạm thời — không cần multi-instance/failover (nhất quán với Unit 2 và toàn hệ thống, local Docker single-machine). Nếu TTS Service down khi Rendering Service gọi, Rendering Service nhận lỗi connection → propagate thành `rendering_failed` → Orchestrator cho retry theo compensating action đã thiết kế
   - ✅ Strengths: nhất quán với các unit khác, không over-engineer cho quy mô 1 người dùng/máy cá nhân
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 5: Security
Yêu cầu bảo mật cho endpoint REST này?

A) 💡 Suggested: Validate input qua Pydantic schema (FastAPI mặc định), không cần auth/rate-limit riêng vì chỉ Rendering Service (nội bộ, cùng Docker network) gọi tới, không expose ra ngoài (Security Baseline extension đã tắt theo `aidlc-state.md`)
   - ✅ Strengths: nhất quán với Unit 2, đúng mức độ cần thiết cho hệ thống nội bộ
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 6: Messaging & Event Participation
Unit 3 có tham gia RabbitMQ không?

A) 💡 Suggested: Không — đã xác nhận ở `unit-of-work.md` (Unit 3 "không tham gia RabbitMQ, chỉ REST"). N/A cho category này
   - ✅ Strengths: đúng theo Units Generation đã duyệt
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 7: Distributed Transaction Participation (Saga)
Unit 3 được gọi trong phạm vi bước Saga `render_scenes` (do Rendering Service điều phối), nhưng không tự publish/consume AMQP message nào.

A) 💡 Suggested: Vai trò **Participant gián tiếp** — không phải Saga coordinator, không tự publish event Saga; chỉ là 1 REST call bên trong logic xử lý của Rendering Service cho bước `render_scenes`. Không cần compensating action riêng ở Unit 3 (stateless, side-effect duy nhất là ghi file audio — đã idempotent theo Business Rule 4, Functional Design) — nếu bước `render_scenes` thất bại và cần retry, TTS Service không cần rollback gì (file audio hợp lệ vẫn giữ nguyên, tái sử dụng)
   - ✅ Strengths: đúng ranh giới trách nhiệm, khớp thiết kế idempotency đã có ở Functional Design
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 8: Caching Requirements
File audio đã sinh được lưu ở shared volume và tái sử dụng qua idempotency check (Functional Design). Có cần thêm cơ chế cache nào khác không (vd. cache Piper model đã load trong memory)?

A) 💡 Suggested: Load Piper voice model (`vi`, `en`) 1 lần lúc service khởi động (trong `main.py`/composition root), giữ trong memory suốt vòng đời process — tránh load lại model (tốn thời gian I/O) mỗi request. Không cần cache tầng nào khác (idempotency qua file đã đủ cho việc tránh tính toán lại)
   - ✅ Strengths: giảm latency đáng kể cho mỗi request (load model 1 lần thay vì mỗi lần synthesize), đơn giản (chỉ là biến in-memory, không cần cache framework)
   - ⚠️ Trade-offs: tăng memory footprint lúc khởi động (chấp nhận được, model Piper nhỏ)

B) Other (please describe after [Answer]: tag below)

[Answer]:A
