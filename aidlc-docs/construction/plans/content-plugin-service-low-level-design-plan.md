# Low-Level Design Plan — Unit 2: Content Plugin Service

## Unit Context
- **Responsibility**: Quản lý content-type plugin nạp động (ADR-0006), phân loại scene theo domain (FR1.1–FR1.3)
- **Architectural style**: Hexagonal/Ports & Adapters (ADR-0002), tech stack Python/FastAPI (ADR-0003)
- **Interfaces**: REST (`GET /plugins`, gọi bởi Gateway) + AMQP consumer/producer (`classify_scenes` command/event, theo `component-methods.md`)
- **Depends on**: Unit 1 (RabbitMQ)

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `module-structure.md`
- [ ] Tạo `dependency-injection.md`
- [ ] Tạo `interface-contracts.md`
- [ ] Tạo `sequence-flows.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Layering & Dependency Direction (BẮT BUỘC)
Theo Hexagonal (ADR-0002), cần xác nhận cách map layer nội bộ cho unit này.

A) 💡 Suggested: 3 layer — `domain/` (Plugin interface `ContentPluginPort`, Scene model, classification rule thuần Python, không import FastAPI/RabbitMQ) → `application/` (use-case: `ClassifySceneUseCase`, `ListPluginsUseCase`, điều phối domain + port) → `adapters/` (`api/` cho FastAPI REST, `messaging/` cho AMQP consumer/producer, `plugins/` chứa các plugin cụ thể như `ProgrammingPlugin` implement `ContentPluginPort`)
   - ✅ Strengths: khớp chuẩn Hexagonal, dependency chỉ đi vào trong (`adapters` → `application` → `domain`), plugin là adapter thay thế được
   - ⚠️ Trade-offs: nhiều thư mục/file hơn so với viết phẳng

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Dependency Injection (BẮT BUỘC)
Cơ chế DI cho unit này?

A) 💡 Suggested: Constructor injection thủ công (theo `architectural-style.md` HLD) — `ClassifySceneUseCase` nhận `ContentPluginRegistry` (abstraction) qua constructor; composition root là `main.py`, nơi khởi tạo registry (quét `plugins/` — ADR-0006) và wire vào FastAPI app qua `Depends()`
   - ✅ Strengths: đơn giản, khớp quyết định HLD, FastAPI's `Depends()` hỗ trợ tự nhiên cho constructor injection kiểu này
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Dynamic Plugin Discovery Mechanism
ADR-0006 chọn dynamic loading từ thư mục `plugins/`. Cơ chế discovery cụ thể?

A) 💡 Suggested: Khi service khởi động, quét toàn bộ file `.py` trong `adapters/plugins/`, dùng `importlib` để import động, tìm class implement `ContentPluginPort` (kiểm tra qua `issubclass`), đăng ký vào `ContentPluginRegistry` (dict `plugin_id -> instance`). Nếu 1 file lỗi (import error, không implement đúng interface), log warning và bỏ qua file đó, KHÔNG crash toàn service
   - ✅ Strengths: tự động, không cần sửa code khi thêm plugin mới, resilient với plugin lỗi
   - ⚠️ Trade-offs: cần validate kỹ để tránh lỗi runtime khó debug; import động luôn có overhead nhỏ lúc khởi động (chấp nhận được vì chỉ chạy 1 lần lúc startup)

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: API Versioning
Unit này expose API cho Gateway (REST) và Orchestrator (qua AMQP). Cần chiến lược versioning không?

A) 💡 Suggested: Chưa cần URI versioning (`/v1/...`) ở giai đoạn này — chỉ 1 client nội bộ (Gateway) do chính bạn kiểm soát, thay đổi API đồng bộ với thay đổi Gateway. Áp dụng nguyên tắc "additive-only" cho message schema (đã quyết định ở `messaging-design.md` Unit 1) để tránh breaking change không cần thiết
   - ✅ Strengths: đơn giản, tránh over-engineering khi chỉ có 1 consumer nội bộ
   - ⚠️ Trade-offs: nếu sau này API được consume bởi bên ngoài, cần bổ sung versioning sau

B) Other (please describe after [Answer]: tag below)

[Answer]: B (Other) — Có, áp dụng URI versioning (`/v1/...`) ngay từ đầu cho toàn bộ REST endpoint của mọi service (không chỉ unit này), thay vì đợi đến khi cần breaking change. Deprecation policy: version cũ được giữ tối thiểu 1 chu kỳ phát triển sau khi version mới ra mắt; vì hệ thống chỉ có 1 client nội bộ (Gateway) do chính người dùng kiểm soát, thông báo deprecation qua changelog nội bộ (không cần cơ chế thông báo tự động).

### Question 5: Distributed Tracing & Correlation ID
Unit này được gọi bởi Gateway (REST) và Orchestrator (AMQP command `classify_scenes` có `saga_id`, `project_id` — theo `component-methods.md`). Cách propagate correlation ID?

A) 💡 Suggested: Dùng `saga_id` đã có sẵn trong message envelope (theo `messaging-design.md` Unit 1) làm correlation ID cho luồng AMQP; với REST (`GET /plugins`, không thuộc Saga), dùng header `X-Request-ID` (sinh mới nếu Gateway chưa gửi) — cả 2 loại ID đều được đưa vào mọi log line của service
   - ✅ Strengths: tái sử dụng `saga_id` đã thiết kế sẵn, không cần thêm cơ chế mới; header `X-Request-ID` là chuẩn phổ biến cho REST
   - ⚠️ Trade-offs: 2 loại ID khác nhau cho 2 kiểu request (chấp nhận được vì bản chất luồng khác nhau — 1 thuộc Saga, 1 không)

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: State Management
Unit này có cần lưu trạng thái (database) không, hay hoàn toàn stateless?

A) 💡 Suggested: Stateless — danh sách plugin được nạp lại từ thư mục `plugins/` mỗi lần service khởi động (in-memory registry), không cần database. Việc phân loại scene là pure function (input scene → output category), không lưu lịch sử
   - ✅ Strengths: đơn giản nhất, không cần thêm database cho unit này
   - ⚠️ Trade-offs: nếu sau này cần audit lịch sử phân loại, phải bổ sung storage — chưa cần ở MVP

B) Other (please describe after [Answer]: tag below)

[Answer]: A
