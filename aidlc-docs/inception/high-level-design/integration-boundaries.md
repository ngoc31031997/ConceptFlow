# Integration Boundaries

**Cập nhật theo ADR-0007**: Orchestration được tách khỏi API Gateway sang Orchestrator Service riêng, giao tiếp với service nghiệp vụ qua Message Queue (RabbitMQ) theo mô hình Saga orchestration-based.

## Integration Points

| From | To | Style | Protocol | Purpose |
|---|---|---|---|---|
| GUI | API Gateway | Synchronous | REST/HTTP+JSON | Thao tác cấu hình đơn giản (soạn script, chọn plugin, cấu hình publish) |
| GUI | API Gateway | Asynchronous (server push) | SSE | Cập nhật tiến trình render/publish real-time |
| API Gateway | Orchestrator Service | Synchronous (khởi tạo) | REST/HTTP+JSON | Khởi chạy 1 Saga (render pipeline hoặc publish) |
| API Gateway | Content Plugin Service | Synchronous | REST/HTTP+JSON | Truy vấn danh sách plugin (không thuộc luồng Saga) |
| Orchestrator Service | Content Plugin Service, Script Processing Service, Rendering Service, TTS Service (qua Rendering), Video Assembly Service, Publisher Service | Asynchronous | Message Queue (RabbitMQ) — command message | Gửi lệnh thực hiện từng bước Saga |
| Content Plugin Service, Script Processing Service, Rendering Service, Video Assembly Service, Publisher Service | Orchestrator Service | Asynchronous | Message Queue (RabbitMQ) — event message | Xác nhận hoàn tất/lỗi từng bước, dùng để Orchestrator quyết định bước tiếp theo hoặc kích hoạt compensating action |
| Rendering Service | TTS Service | Synchronous | REST/HTTP+JSON | Yêu cầu sinh audio giọng đọc cho từng scene (nội bộ trong phạm vi 1 bước Saga "Render") |
| Publisher Service | YouTube Data API | Synchronous | HTTPS/OAuth 2.0 | Xác thực và upload video |

## Communication Style Rationale
- **GUI ↔ Gateway**: đồng bộ REST cho thao tác cấu hình; SSE một chiều cho tiến trình.
- **Gateway ↔ Orchestrator**: đồng bộ REST chỉ để khởi tạo Saga (nhận `saga_id`/`project_id` ngay lập tức); tiến trình sau đó được theo dõi qua event.
- **Orchestrator ↔ Service nghiệp vụ**: **bất đồng bộ qua Message Queue** — quyết định thay đổi so với thiết kế ban đầu (ADR-0005, đã bị supersede) theo yêu cầu người dùng, nhằm: (1) không chặn Orchestrator khi Rendering Service xử lý lâu, (2) đảm bảo command không mất khi 1 service tạm thời down, (3) hỗ trợ retry tự nhiên qua cơ chế requeue/dead-letter của RabbitMQ.
- **Rendering ↔ TTS**: vẫn giữ đồng bộ REST vì đây là tương tác nội bộ trong phạm vi thực thi 1 bước Saga (Rendering Service chủ động cần audio ngay để đồng bộ timing animation), không phải ranh giới giữa các bước Saga.

## Orchestration Pattern: Saga (Orchestration-based)
Xem ADR-0007 cho phân tích đầy đủ. Tóm tắt:
- **Orchestrator Service** là Saga coordinator duy nhất — biết toàn bộ định nghĩa các bước và thứ tự.
- **Saga Steps** (luồng Render): `ParseScript → ClassifyScenes (Content Plugin) → RenderScenes (Rendering + TTS nội bộ) → AssembleVideo → (kết thúc: ready_to_publish)`
- **Saga Steps** (luồng Publish): `AuthenticateYouTube (nếu chưa) → UploadVideo (Publisher) → (kết thúc: published)`
- **Compensating Actions** (ví dụ, chi tiết hóa ở Low-Level Design):
  - Nếu `RenderScenes` thất bại giữa chừng → giữ scene đã render thành công, đánh dấu scene lỗi, cho phép retry chỉ scene đó (không cần compensating xóa dữ liệu vì animation clip hợp lệ không cần rollback)
  - Nếu `AssembleVideo` thất bại → giữ animation/audio clip trong shared volume, cho phép Orchestrator retry riêng bước Assembly
  - Nếu `UploadVideo` thất bại → không có compensating action cần thiết vì chưa có gì được tạo ở phía YouTube; Orchestrator cho phép retry bước Publish

## Distributed Consistency Approach
- Sử dụng **Saga orchestration-based** với eventual consistency: mỗi bước hoàn tất độc lập, Orchestrator cập nhật trạng thái tổng thể sau khi nhận event xác nhận.
- Không dùng 2PC. Compensating action được định nghĩa tối thiểu ở mức "cho phép retry từng bước, giữ nguyên kết quả hợp lệ đã có" — không cần rollback phức tạp vì hầu hết các bước tạo mới dữ liệu (idempotent theo `project_id` + `step`) thay vì sửa dữ liệu chia sẻ.
- **Idempotency**: mỗi command message mang `project_id` + `step_id`; consumer phải xử lý idempotent (nếu nhận trùng message do requeue, không tạo lại artifact đã có).

## Data Ownership Map

| Data Entity | Owning Service |
|---|---|
| Content Plugin definitions & scene classification | Content Plugin Service |
| Script & parsed Scene structure | Script Processing Service |
| Animation render output (video clip per scene) | Rendering Service |
| Audio output (voice-over clip per scene) | TTS Service |
| Final assembled video (.mp4) | Video Assembly Service |
| YouTube OAuth credential & upload metadata | Publisher Service |
| Video project state / Saga state (trạng thái tổng thể: draft/rendering/failed-at-step/published) | **Orchestrator Service** (thay đổi từ Gateway → Orchestrator theo ADR-0007) |

Mỗi entity chỉ có đúng 1 service sở hữu — không có shared database giữa các service; mỗi service quản lý storage riêng (file-based cho các artifact media qua Shared Docker Volume, hoặc DB nhẹ cho metadata/Saga state — quyết định cụ thể ở NFR Design/Infrastructure Design).

Xem ADR-0007 cho quyết định orchestration pattern (supersedes ADR-0005).
