# Application Design — Consolidated Overview

**Dự án**: Manim-based Educational Video Generation Tool
**Ngày**: 2026-08-04 (cập nhật theo ADR-0007)

**Cập nhật (CR-055, 2026-09-30)**: Content Plugin Service (gỡ ở CR-020) và Script Processing Service (gỡ ở CR-040) không còn; thêm Authoring Service (CR-040, ADR-0029) và LLM Service (CR-039). Nội dung dưới đây theo hệ đang chạy.

## 1. Components
Web GUI, API Gateway, Orchestrator Service, Authoring Service, LLM Service (+ Ollama), RabbitMQ, Rendering Service, TTS Service, Video Assembly Service, Publisher Service.
→ Chi tiết: `components.md`

## 2. Component Methods (API Contracts & Message Schemas)
GUI↔Gateway và Gateway↔Orchestrator dùng REST; Orchestrator↔Service nghiệp vụ dùng **RabbitMQ command/event message** cho mọi service, kể cả TTS (ADR-0014 — không còn ngoại lệ Rendering↔TTS REST).
→ Chi tiết: `component-methods.md`

## 3. Services (Orchestration)
**Orchestrator Service** là Saga coordinator duy nhất (theo ADR-0007, supersedes ADR-0005), điều phối 2 Saga chính: **Render Pipeline** (Validate Script → Synthesize Speech → Render Scenes → Assemble Video) và **Publish** (Publish Video). Compensating action theo bước, idempotent theo `saga_id`+`project_id`.
→ Chi tiết: `services.md`

## 4. Component Dependencies
Không có circular dependency. **RabbitMQ là dependency hạ tầng chung** của Orchestrator và mọi service nghiệp vụ trong Saga. Shared Docker Volume vẫn là kênh chia sẻ artifact trung gian.
→ Chi tiết: `component-dependency.md`

## 5. Key Decisions (ADR)
| ADR | Quyết định |
|---|---|
| ADR-0006 | *(Superseded — gỡ cùng Content Plugin Service ở CR-020)* Dynamic plugin loading thay vì static registry |
| ADR-0007 | Saga Orchestration qua Orchestrator Service riêng + RabbitMQ, thay thế ADR-0005 (Gateway-as-orchestrator, REST đồng bộ) |

## 6. Traceability to Requirements & Stories
- FR1 (Plugin Architecture) → ban đầu Content Plugin Service + ADR-0006; gỡ ở CR-020
- FR2 (Script Processing) → soạn kịch bản: Authoring Service + LLM Service; tìm Scene/lượt dry: bước `validate_script` của Rendering Service (CR-040)
- FR3, FR4, FR5 (Rendering, TTS, Assembly) → Rendering Service, TTS Service, Video Assembly Service
- FR6 (GUI) → Web GUI + API Gateway (SSE) + Orchestrator Service (state machine)
- FR7 (YouTube Publishing) → Publisher Service
- FR8 (Containerized Runtime) → toàn bộ component đóng gói qua docker-compose, chia sẻ Shared Docker Volume + RabbitMQ

## 7. Open Items chuyển tiếp sang Units Generation / Construction
- Mỗi microservice (bao gồm Orchestrator Service, RabbitMQ setup) sẽ trở thành 1 unit công việc riêng (Units Generation)
- Chi tiết RabbitMQ exchange/queue topology, retry/dead-letter policy → Low-Level Design + Infrastructure Design của Orchestrator Service unit
- Chi tiết quy ước đường dẫn shared volume → Low-Level Design / Infrastructure Design
- Schema database/state store cụ thể cho Saga state tại Orchestrator Service → Functional Design / NFR Design của Orchestrator Service unit
