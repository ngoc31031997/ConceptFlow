# Components

Mỗi component dưới đây tương ứng 1 microservice độc lập (đóng gói 1 container), tổ chức nội bộ theo Hexagonal/Ports & Adapters (xem `architectural-style.md`).

**Cập nhật theo ADR-0007**: Vai trò orchestration tách khỏi API Gateway sang Orchestrator Service riêng, giao tiếp qua Message Queue (RabbitMQ).

**Cập nhật (CR-055, 2026-09-30)**: Content Plugin Service (gỡ ở CR-020) và Script Processing Service (gỡ ở CR-040) không còn; thêm Authoring Service (CR-040, ADR-0029) và LLM Service (CR-039). Nội dung dưới đây theo hệ đang chạy.

## 1. Web GUI
- **Purpose**: Giao diện web cho Creator.
- **Responsibilities**: Tạo project theo wizard; soạn kịch bản cùng AI; chọn ngôn ngữ/giọng/engine; khởi chạy render; hiển thị tiến trình (SSE); phát video preview; cấu hình & kích hoạt publish YouTube.
- **Interface**: Gọi REST + nhận SSE từ API Gateway. Không expose interface nào cho component khác gọi vào.

## 2. API Gateway
- **Purpose**: Điểm vào duy nhất cho GUI; routing.
- **Responsibilities**: Định tuyến yêu cầu REST tới Orchestrator, Authoring Service, Publisher; chuyển tiếp yêu cầu khởi chạy pipeline (render/publish) sang Orchestrator Service; nghe `progress.fanout` và forward qua SSE cho GUI (ADR-0017). **Không còn quản lý state machine hay điều phối tuần tự** (chuyển sang Orchestrator Service).
- **Interface**: Expose REST API + SSE endpoint cho GUI; là REST client của Orchestrator Service, Authoring Service và Publisher Service.

## 3. Orchestrator Service (MỚI — ADR-0007)
- **Purpose**: Saga coordinator — điều phối toàn bộ luồng nghiệp vụ, quản lý state machine của video project/Saga.
- **Responsibilities**: Nhận yêu cầu khởi tạo Saga từ Gateway (REST); gửi command message qua RabbitMQ tới từng service theo đúng thứ tự bước Saga; nhận event message xác nhận hoàn tất/lỗi; cập nhật trạng thái project (danh sách hiện tại: `ProjectStatus` trong `services/orchestrator/internal/domain/project.go`, gồm `failed_at_<step>`); sở hữu project, fork, huỷ/retry bước, channel asset, xoá project; kích hoạt compensating action khi cần; publish sự kiện tiến trình để Gateway forward qua SSE.
- **Interface**: Expose REST API cho Gateway (khởi tạo Saga, truy vấn trạng thái); publish/consume message trên RabbitMQ với các service nghiệp vụ.

## 4. Message Queue — RabbitMQ (MỚI — ADR-0007)
- **Purpose**: Hạ tầng giao tiếp bất đồng bộ.
- **Responsibilities**: Định tuyến command message từ Orchestrator tới đúng queue của từng service; định tuyến event message phản hồi về Orchestrator; đảm bảo durability (message không mất khi consumer tạm thời down) và hỗ trợ retry/dead-letter.
- **Interface**: Không phải REST — là message broker (AMQP), các service khác là publisher/consumer.

## 5. Authoring Service (CR-040, ADR-0029)
- **Purpose**: Soạn kịch bản cùng AI.
- **Responsibilities**: Thư viện prompt; chuỗi soạn 1a/1b/1c (câu chuyện → storyboard → code); wizard; gợi ý metadata và short; thư viện/kế hoạch hình minh hoạ; nhật ký dùng LLM. DB riêng.
- **Interface**: REST cho Gateway; HTTP nội bộ với Orchestrator (`/internal/v1/authoring/*` một chiều, API project của Orchestrator chiều kia); REST client của LLM Service và Rendering Service.

## 6. LLM Service (CR-039)
- **Purpose**: Nơi duy nhất gọi mô hình ngôn ngữ.
- **Responsibilities**: Gọi Hive (OpenAI SDK) hoặc Ollama; pipeline sinh code chia đoạn (bố cục, shot song song, gộp, kiểm biên dịch qua Rendering, sửa). Không DB.
- **Interface**: REST nội bộ (`/v1/chat`, `/v1/storyboard/finalize`, `/v1/code/generate`, `/v1/suggest-*`), chỉ Authoring Service gọi.

## 7. Rendering Service
- **Purpose**: Sinh animation Manim/Remotion từ script đã chốt.
- **Cập nhật (CR-040)**: nhận thêm bước `validate_script` (tìm tên Scene/composition bằng `domain/script_locator.py`, chạy lượt dry; event `script_validated`/`validation_failed`) và HTTP nội bộ `/v1/check/*`, `/v1/illustrations/preview`.
- **Responsibilities**: Nhận command `render_scenes` từ Orchestrator qua RabbitMQ; render animation cho từng scene (code highlight, thuật toán, khái niệm lập trình); đồng bộ thời lượng animation với audio đã sinh sẵn (từ bước Saga "Synthesize Speech" riêng — ADR-0014, KHÔNG còn tự gọi TTS Service); ghi animation clip vào shared volume; publish event tiến trình theo từng scene (`scene_rendered`) và event hoàn tất/lỗi (`rendering_completed`/`rendering_failed`).
- **Interface**: Consumer/Producer trên RabbitMQ. **Revision (2026-08-07, ADR-0014)**: không còn REST client của TTS Service — audio đã có sẵn từ bước Saga trước, truyền vào qua payload của `render_scenes`.

## 8. TTS Service
- **Purpose**: Sinh giọng đọc từ lời thoại.
- **Cập nhật**: từ ADR-0014 là bước Saga `synthesize_speech` qua RabbitMQ (không còn REST); engine Edge/Azure theo ADR-0024/0025. Hai dòng dưới là thiết kế ban đầu.
- **Responsibilities**: Nhận đoạn lời thoại + ngôn ngữ (gọi trực tiếp từ Rendering Service qua REST); sinh audio bằng TTS engine offline (adapter cụ thể: Coqui/Piper); ghi audio clip vào shared volume; trả về thời lượng audio.
- **Interface**: REST API — `POST /tts/synthesize` (không tham gia trực tiếp vào Message Queue vì được gọi đồng bộ nội bộ bởi Rendering Service).

## 9. Video Assembly Service
- **Purpose**: Ghép animation + audio (+ nhạc nền tùy chọn) thành video hoàn chỉnh.
- **Responsibilities**: Nhận command `assemble_video` từ Orchestrator qua RabbitMQ; đọc animation clip & audio clip từ shared volume theo thứ tự scene; ghép bằng ffmpeg; ghi file .mp4 hoàn chỉnh vào shared volume; publish event `video_assembled`/`assembly_failed`.
- **Interface**: Consumer/Producer trên RabbitMQ.

## 10. Publisher Service
- **Purpose**: Xác thực và đăng video lên YouTube.
- **Responsibilities**: Quản lý luồng OAuth 2.0 (expose REST cho GUI thực hiện qua Gateway); lưu credential; nhận command `publish_video` (kèm metadata: tiêu đề, mô tả, tag, chế độ hiển thị) từ Orchestrator qua RabbitMQ; upload video .mp4 lên kênh YouTube của Creator; publish event `video_published`/`publish_failed`.
- **Interface**: REST API — `GET /auth/youtube/start`, `GET /auth/youtube/callback` (luồng OAuth, ngoài Saga); Consumer/Producer trên RabbitMQ cho bước `publish_video`.
