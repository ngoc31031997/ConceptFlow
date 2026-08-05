# ADR-0009: Selective Polyglot Tech Stack (per-unit language mapping)

## Status
Accepted

## Date
2026-08-05

## Stage
NFR Requirements (Unit 2: Content Plugin Service — decision applies system-wide, refines ADR-0003)

## Context
ADR-0003 (High-Level Design) chọn Python/FastAPI đồng nhất cho mọi backend service. Khi xác nhận tech stack cho Unit 2 ở NFR Requirements, người dùng nêu rõ mục tiêu dự án không chỉ là hoàn thành công cụ nhanh nhất, mà còn để **học sâu hơn về kiến trúc microservices** — bao gồm trải nghiệm giao tiếp liên ngôn ngữ (polyglot), khác biệt concurrency model, và tooling giữa các stack khác nhau.

## Options Considered
### Option A: Giữ nguyên Python/FastAPI đồng nhất (ADR-0003 ban đầu)
- What it is: Mọi backend service dùng Python/FastAPI.
- Strengths: Đơn giản nhất để bảo trì cho 1 người phát triển, tái sử dụng code tối đa giữa các service (message envelope, AMQP client wrapper, schema).
- Trade-offs: Không mang lại giá trị học tập về giao tiếp/vận hành hệ polyglot thực tế — một phần quan trọng của mục tiêu dự án.

### Option B: Polyglot tối đa — mỗi service 1 ngôn ngữ khác nhau
- What it is: Script Processing (Node.js/TypeScript), Video Assembly (Go), Publisher (Java/Kotlin), Orchestrator (Go), API Gateway (Node.js), giữ Python chỉ cho Rendering/TTS (ràng buộc thư viện).
- Strengths: Trải nghiệm học nhiều ngôn ngữ nhất.
- Trade-offs: Mỗi service cần riêng Dockerfile/toolchain/test framework/CI config — chi phí thiết lập và bảo trì cao đáng kể cho 1 người phát triển, có thể làm chậm tiến độ MVP không cần thiết.

### Option C: Polyglot có chọn lọc (Chosen)
- What it is: Giữ Python cho các service không có lý do học tập rõ ràng để đổi (Content Plugin, Script Processing, Video Assembly, Publisher — cộng với Rendering, TTS do ràng buộc thư viện Manim/Coqui/Piper). Dùng **Go** cho Orchestrator Service (goroutine/channel phù hợp tự nhiên với vai trò Saga coordinator — điều phối nhiều luồng command/event concurrency). Dùng **Node.js** cho API Gateway (mô hình reverse-proxy/event-loop, khác biệt rõ với Python async, là pattern phổ biến thực tế cho API Gateway).
- Strengths: 2 điểm học tập rõ ràng, có chủ đích (Go cho concurrency-heavy coordination, Node.js cho gateway/proxy pattern), không dàn trải quá nhiều ngôn ngữ cùng lúc; các service còn lại (không có giá trị học tập rõ ràng từ việc đổi ngôn ngữ) vẫn giữ Python để hạn chế chi phí bảo trì.
- Trade-offs: Vẫn cần quản lý 3 toolchain (Python, Go, Node.js) thay vì 1; mất một phần khả năng tái sử dụng code giữa Orchestrator/Gateway và các service Python.

## Decision
Chọn **Option C: Polyglot có chọn lọc**.

**Language mapping theo unit**:
| Unit | Language/Runtime | Lý do |
|---|---|---|
| Unit 2: Content Plugin Service | Python 3.12 + FastAPI | Đã duyệt ở Low-Level Design, không có giá trị học tập rõ ràng để đổi |
| Unit 3: TTS Service | Python 3.12 | Ràng buộc thư viện (Coqui TTS/Piper là Python-first) |
| Unit 4: Script Processing Service | Python 3.12 + FastAPI | Không có giá trị học tập rõ ràng để đổi |
| Unit 5: Rendering Service | Python 3.12 | Ràng buộc cứng — Manim chỉ có API Python |
| Unit 6: Video Assembly Service | Python 3.12 + FastAPI | Không có giá trị học tập rõ ràng để đổi |
| Unit 7: Publisher Service | Python 3.12 + FastAPI | Không có giá trị học tập rõ ràng để đổi |
| Unit 8: Orchestrator Service | **Go** | Saga coordinator là bài toán điều phối concurrency-heavy (nhiều command/event message song song) — cơ hội học goroutine/channel tự nhiên |
| Unit 9: API Gateway | **Node.js** (Express hoặc Fastify) | Mô hình reverse-proxy/event-loop khác biệt rõ với Python async — pattern phổ biến thực tế cho API Gateway |
| Unit 10: Web GUI | TypeScript/React | Đã quyết định ở ADR-0003 (không đổi) |

## Rationale
Cân bằng giữa mục tiêu học tập (polyglot có chủ đích ở 2 điểm concurrency-model khác biệt rõ rệt: Go cho coordination, Node.js cho gateway) và chi phí thực tế cho 1 người phát triển (không dàn trải toàn bộ hệ thống thành nhiều ngôn ngữ khi không có lý do sư phạm cụ thể). Rendering/TTS giữ Python vì đây là ràng buộc kỹ thuật cứng, không phải lựa chọn.

## Consequences
- **Positive**: Đạt mục tiêu học tập microservices polyglot một cách có kiểm soát; Orchestrator (Go) và Gateway (Node.js) là 2 service có đặc tính kỹ thuật hưởng lợi thực sự từ ngôn ngữ được chọn (không chỉ học vì học).
- **Negative / Accepted Trade-offs**: Cần 3 toolchain riêng biệt (Python, Go, Node.js) trong CI/CD và Docker packaging; message schema (JSON qua RabbitMQ) phải được định nghĩa độc lập với ngôn ngữ (đã là JSON — không cần thay đổi, nhưng cần tự triển khai lại (de)serialization ở Go/Node thay vì dùng chung Pydantic model như các service Python).
- **Follow-ups**: Khi thiết kế Low-Level Design của Unit 8 (Orchestrator, Go) và Unit 9 (API Gateway, Node.js), cần định nghĩa lại tech-stack-decisions.md riêng cho từng unit theo ngôn ngữ tương ứng; `infrastructure-design.md` (Unit 1, RabbitMQ) cần bổ sung ghi chú client AMQP tương ứng cho Go (`amqp091-go`) và Node.js (`amqplib`) bên cạnh `aio-pika` (Python).

## Related
- Design artifact: `aidlc-docs/inception/high-level-design/technology-direction.md` (cập nhật), `aidlc-docs/construction/content-plugin-service/nfr-requirements/tech-stack-decisions.md`
- Related ADRs: Refines ADR-0003 (không supersede hoàn toàn — Rendering/TTS/GUI vẫn giữ nguyên theo ADR-0003)
