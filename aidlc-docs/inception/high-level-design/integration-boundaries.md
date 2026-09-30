# Integration Boundaries

**Cập nhật theo ADR-0007**: Orchestration được tách khỏi API Gateway sang Orchestrator Service riêng, giao tiếp với service nghiệp vụ qua Message Queue (RabbitMQ) theo mô hình Saga orchestration-based.

**Cập nhật (CR-055, 2026-09-30)**: Content Plugin Service (gỡ ở CR-020) và Script Processing Service (gỡ ở CR-040) không còn; Authoring Service (CR-040, ADR-0029), LLM Service và Ollama (CR-039) được thêm. Bảng dưới đây theo hệ đang chạy.

**Revision (2026-08-07, ADR-0014)**: Rendering ↔ TTS REST đồng bộ (dòng dưới đây) đã bị loại bỏ — TTS Service nay là bước Saga độc lập ("Synthesize Speech"), message-driven hoàn toàn qua RabbitMQ, không còn bất kỳ tương tác trực tiếp nào giữa Rendering Service và TTS Service.

## Integration Points

| From | To | Style | Protocol | Purpose |
|---|---|---|---|---|
| GUI | API Gateway | Synchronous | REST/HTTP+JSON | Tạo/cấu hình project, soạn kịch bản, duyệt, cấu hình publish |
| GUI | API Gateway | Asynchronous (server push) | SSE | Cập nhật tiến trình render/publish/soạn real-time |
| API Gateway | Orchestrator Service | Synchronous | REST/HTTP+JSON | Khởi chạy Saga (render, publish), đọc/sửa project, huỷ/retry bước |
| API Gateway | Authoring Service | Synchronous | REST/HTTP+JSON | Soạn kịch bản (1a/1b/1c), prompt, gợi ý metadata, hình minh hoạ |
| API Gateway | Publisher Service | Synchronous | REST/HTTP+JSON | OAuth YouTube (app, tài khoản) |
| Orchestrator Service ↔ Authoring Service | — | Synchronous | HTTP nội bộ (`/internal/v1/authoring/*` và API project của Orchestrator) | Mỗi bên đọc phần dữ liệu bên kia sở hữu (ADR-0029); không `depends_on`, mỗi bên tự suy giảm khi bên kia down |
| Authoring Service | LLM Service | Synchronous | REST/HTTP+JSON | Mọi lượt gọi mô hình ngôn ngữ |
| LLM Service | Hive API / Ollama | Synchronous | HTTPS / HTTP (OpenAI SDK) | Sinh văn bản/code |
| LLM Service, Authoring Service | Rendering Service | Synchronous | HTTP nội bộ (`/v1/check/*`, `/v1/illustrations/preview`) | Kiểm biên dịch code sinh ra, preview hình minh hoạ |
| Orchestrator Service | TTS, Rendering, Video Assembly, Publisher Service | Asynchronous | Message Queue (RabbitMQ) — command message | Gửi lệnh thực hiện từng bước Saga |
| TTS, Rendering, Video Assembly, Publisher Service | Orchestrator Service | Asynchronous | Message Queue (RabbitMQ) — event message | Xác nhận hoàn tất/lỗi từng bước, dùng để Orchestrator quyết định bước tiếp theo hoặc kích hoạt compensating action |
| Orchestrator Service | TTS, Rendering, Video Assembly | Asynchronous | RabbitMQ `control.fanout` | Lệnh huỷ bước đang chạy |
| Các service | API Gateway | Asynchronous | RabbitMQ `progress.fanout` (ADR-0017) | Tiến độ cho SSE |
| Publisher Service | YouTube Data API | Synchronous | HTTPS/OAuth 2.0 | Xác thực và upload video |

## Communication Style Rationale
- **GUI ↔ Gateway**: đồng bộ REST cho thao tác cấu hình; SSE một chiều cho tiến trình.
- **Gateway ↔ Orchestrator**: đồng bộ REST chỉ để khởi tạo Saga (nhận `saga_id`/`project_id` ngay lập tức); tiến trình sau đó được theo dõi qua event.
- **Orchestrator ↔ Service nghiệp vụ**: **bất đồng bộ qua Message Queue** cho MỌI service nghiệp vụ, kể cả TTS Service (ADR-0014) — quyết định thay đổi so với thiết kế ban đầu (ADR-0005, đã bị supersede), nhằm: (1) không chặn Orchestrator khi 1 bước xử lý lâu, (2) đảm bảo command không mất khi 1 service tạm thời down, (3) hỗ trợ retry tự nhiên qua cơ chế requeue/dead-letter của RabbitMQ, (4) mỗi bước là 1 sự kiện độc lập, quan sát/lưu trữ được (hỗ trợ mục tiêu Orchestrator persist kết quả từng bước).

## Orchestration Pattern: Saga (Orchestration-based)
Xem ADR-0007 cho phân tích đầy đủ. Tóm tắt:
- **Orchestrator Service** là Saga coordinator duy nhất — biết toàn bộ định nghĩa các bước và thứ tự.
- **Saga Steps** (luồng Render, sau CR-020/CR-040): `ValidateScript (Rendering) → SynthesizeSpeech (TTS) → RenderScenes (Rendering, animation-only) → AssembleVideo (Video Assembly) → (kết thúc: ready_to_publish)`; `GenerateClips` (Video Assembly) chạy theo yêu cầu. Danh sách bước: `services/orchestrator/internal/domain/project.go` (`StepName`).
- **Saga Steps** (luồng Publish): `PublishVideo (Publisher) → (kết thúc: published)`; OAuth làm trước qua REST, không phải bước saga.
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
| Prompt, kịch bản đang soạn (câu chuyện, storyboard, code), hình minh hoạ, nhật ký dùng LLM | Authoring Service |
| Animation render output (`rendered.mp4`, `timing.json`) | Rendering Service |
| Audio output (voice-over clip per scene) | TTS Service |
| Final assembled video (.mp4) | Video Assembly Service |
| YouTube OAuth credential & upload metadata | Publisher Service |
| Video project state / Saga state (trạng thái tổng thể: draft/rendering/failed-at-step/published) | **Orchestrator Service** (thay đổi từ Gateway → Orchestrator theo ADR-0007) |

Mỗi entity chỉ có đúng 1 service sở hữu — không có shared database giữa các service; mỗi service quản lý storage riêng: Postgres riêng (ADR-0013) cho metadata/Saga state, file media qua volume `shared_artifacts` theo hợp đồng `docs/contracts/shared-artifacts.md`.

Xem ADR-0007 cho quyết định orchestration pattern (supersedes ADR-0005).
