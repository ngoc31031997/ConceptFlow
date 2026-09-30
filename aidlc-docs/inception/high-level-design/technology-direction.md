# Technology Direction

## Selected Stack

| Layer | Technology | Rationale |
|---|---|---|
| Animation Rendering | Python + Manim; Remotion (React/Node) là engine thứ hai (CR-038) | Yêu cầu cốt lõi của dự án — Manim là thư viện Python duy nhất phù hợp cho phong cách animation 3Blue1Brown |
| Backend Services | Python cho tts, rendering, video-assembly, publisher, llm-service; **Go cho Orchestrator Service và Authoring Service (ADR-0018, ADR-0029); Node.js cho API Gateway** (ADR-0009, selective polyglot phục vụ mục tiêu học microservices) | Cùng ngôn ngữ với Manim cho phần lớn service (đơn giản hóa gọi Manim trực tiếp trong Rendering); Go phù hợp vai trò Saga coordinator concurrency-heavy; Node.js phù hợp mô hình reverse-proxy/event-loop cho Gateway |
| Frontend GUI | React (Web) | Hệ sinh thái mature cho GUI phức tạp (soạn script, theo dõi tiến trình render real-time, phát video preview) |
| GUI ↔ Backend (config/CRUD) | REST (HTTP/JSON) qua API Gateway | Phù hợp thao tác dạng request/response (soạn script, cấu hình project, cấu hình publish) |
| GUI ↔ Backend (tiến trình render) | Server-Sent Events (SSE) | Phù hợp luồng cập nhật một chiều (server → client) cho tiến trình render dài hạn, đơn giản hơn WebSocket cho use case một chiều này |
| TTS Engine | Edge Read Aloud (nền), Azure Speech (tuỳ chọn), Google Cloud TTS (ngủ đông) | ADR-0024/0025 thay quyết định Piper offline ban đầu (ADR-0010, superseded) |
| Mô hình ngôn ngữ | Hive (OpenAI SDK) hoặc Ollama local, chỉ qua LLM Service | CR-039 |
| Video Assembly | ffmpeg (qua Python binding) | Công cụ chuẩn để ghép video/audio, tương thích tốt với output Manim |
| Inter-service Communication (Orchestrator ↔ Services) | Message Queue — RabbitMQ | Theo ADR-0007 (thay thế quyết định REST đồng bộ ban đầu tại ADR-0005, đã supersede): hỗ trợ Saga orchestration-based, command/event message durable, retry qua requeue/dead-letter. **Revision (ADR-0014)**: áp dụng cho MỌI service nghiệp vụ, kể cả TTS Service — không còn ngoại lệ REST nội bộ nào giữa Rendering↔TTS |
| API Gateway | Node.js + Express (ADR-0020) | Định tuyến request từ GUI đến đúng backend service/Orchestrator, gộp luồng SSE tiến trình render |
| Orchestration | Orchestrator Service riêng biệt (Go — ADR-0018) | Theo ADR-0007: tách trách nhiệm điều phối Saga khỏi API Gateway |
| Containerization | Docker + docker-compose | Yêu cầu bắt buộc (FR8.1) — toàn bộ hệ thống chạy qua 1 lệnh trên máy cá nhân |
| YouTube Integration | Google API Python Client + OAuth 2.0 | Thư viện chính thức của Google cho YouTube Data API |

## Key Constraints Driving These Choices
- **Manim là ràng buộc cứng** cho animation engine → kéo theo Python cho mọi service cần gọi Manim trực tiếp
- **Không có yêu cầu tốc độ render / scale** → không cần công nghệ phức tạp cho xử lý phân tán (không cần message broker, không cần orchestration engine)
- **Chỉ 1 người dùng, chạy local** → không cần service discovery phức tạp; docker-compose với DNS nội bộ giữa các container là đủ
- **Kiến trúc plugin bắt buộc (NFR1)** (ban đầu hiện thực bằng Content Plugin Service, đã gỡ ở CR-020) → ảnh hưởng đến Architectural Style (xem `architectural-style.md`), không ảnh hưởng trực tiếp đến lựa chọn ngôn ngữ/framework

Xem ADR-0003 cho phân tích lựa chọn tech stack, và ADR-0007 cho quyết định Message Queue/Orchestrator Service.
