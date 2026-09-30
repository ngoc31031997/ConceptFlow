# Architectural Style

## Chosen Style: Hexagonal / Ports & Adapters (per-service)

Mỗi microservice (đặc biệt là **Rendering Service**, **TTS Service**; thiết kế ban đầu còn gồm Content Plugin Service, đã gỡ ở CR-020) được tổ chức nội bộ theo kiến trúc **Hexagonal (Ports & Adapters)**:
- **Domain core**: Logic nghiệp vụ thuần (vd. plugin resolution rule, scene model, orchestration của thứ tự animation) không phụ thuộc trực tiếp vào bất kỳ thư viện công nghệ cụ thể nào.
- **Ports**: Interface trừu tượng mà domain core định nghĩa (vd. `TTSEnginePort`, `AnimationRendererPort`).
- **Adapters**: Cài đặt cụ thể cắm vào port (vd. `ManimAnimationAdapter` cài đặt `AnimationRendererPort`, `EdgeTTSAdapter`/`AzureTTSAdapter` cài đặt `TTSEnginePort`).

## Dependency Rule
Domain core không import/phụ thuộc vào bất kỳ adapter cụ thể nào (Manim, thư viện TTS cụ thể, YouTube API client). Adapter phụ thuộc vào domain core (thông qua việc implement port interface), không theo chiều ngược lại. Điều này cho phép thay thế adapter (vd. đổi TTS engine, thêm content plugin mới) mà không sửa domain core.

## Áp dụng thực tế (CR-055)
- tts, rendering, video-assembly, publisher: `domain/ → application/ → adapters/<công nghệ>/`.
- orchestrator, authoring-service (Go): `cmd/<svc>/` + `internal/{domain,application,adapters,config}`.
- api-gateway: phân lớp `src/{routes,handlers,clients,middleware,config}` (không có domain, ADR-0020).
- llm-service: layout phẳng `app/` — service mỏng bọc lời gọi LLM, không áp hexagonal.

## Rationale
Yêu cầu **NFR1 (Extensibility)** trong requirements.md xác định rõ: kiến trúc phải hỗ trợ thêm content-type plugin mới (vd. domain Tiếng Anh) và có khả năng thay đổi TTS provider mà không phải viết lại core logic. Hexagonal là style tự nhiên nhất cho việc này — "port" cho content-type và TTS chính là cơ chế pluggable được yêu cầu, "adapter" là nơi cắm implementation cụ thể cho từng domain/provider.

## Dependency Injection
- **Cơ chế**: Constructor injection thủ công (manual constructor injection) ở mỗi service — mỗi service khởi tạo domain core với các adapter cụ thể được truyền vào qua constructor, cấu hình qua biến môi trường/config file (vd. các engine TTS mà `RoutingTTSEngine` chọn theo giọng).
- **Vì sao không dùng DI container/framework**: Ở quy mô mỗi service (single responsibility, ít adapter), constructor injection thủ công đủ rõ ràng và không cần thêm framework DI phức tạp. Chi tiết wiring cụ thể từng service sẽ được xác định ở Low-Level Design.

Xem ADR-0002 cho phân tích lựa chọn Architectural Style.
