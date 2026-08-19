# Application Design — Consolidated Overview

**Dự án**: Manim-based Educational Video Generation Tool
**Ngày**: 2026-08-04 (cập nhật theo ADR-0007)

## 1. Components
10 microservice: Web GUI, API Gateway, **Orchestrator Service** (mới), **RabbitMQ** (mới), Content Plugin Service, Script Processing Service, Rendering Service, TTS Service, Video Assembly Service, Publisher Service.
→ Chi tiết: `components.md`

## 2. Component Methods (API Contracts & Message Schemas)
GUI↔Gateway và Gateway↔Orchestrator dùng REST; Orchestrator↔Service nghiệp vụ dùng **RabbitMQ command/event message** cho mọi service, kể cả TTS (ADR-0014 — không còn ngoại lệ Rendering↔TTS REST).
→ Chi tiết: `component-methods.md`

## 3. Services (Orchestration)
**Orchestrator Service** là Saga coordinator duy nhất (theo ADR-0007, supersedes ADR-0005), điều phối 2 Saga chính: **Render Pipeline** (Parse Script → Classify Scenes → Render Scenes → Assemble Video) và **Publish** (Publish Video). Compensating action theo bước, idempotent theo `saga_id`+`project_id`.
→ Chi tiết: `services.md`

## 4. Component Dependencies
Không có circular dependency. **RabbitMQ là dependency hạ tầng chung** của Orchestrator và mọi service nghiệp vụ trong Saga. Shared Docker Volume vẫn là kênh chia sẻ artifact trung gian.
→ Chi tiết: `component-dependency.md`

## 5. Key Decisions (ADR)
| ADR | Quyết định |
|---|---|
| ADR-0006 | Dynamic plugin loading (quét thư mục `plugins/`) thay vì static registry |
| ADR-0007 | Saga Orchestration qua Orchestrator Service riêng + RabbitMQ, thay thế ADR-0005 (Gateway-as-orchestrator, REST đồng bộ) |

## 6. Traceability to Requirements & Stories
- FR1 (Plugin Architecture) → Content Plugin Service + ADR-0006
- FR2 (Script Processing) → Script Processing Service
- FR3, FR4, FR5 (Rendering, TTS, Assembly) → Rendering Service, TTS Service, Video Assembly Service
- FR6 (GUI) → Web GUI + API Gateway (SSE) + Orchestrator Service (state machine)
- FR7 (YouTube Publishing) → Publisher Service
- FR8 (Containerized Runtime) → toàn bộ 10 component đóng gói qua docker-compose, chia sẻ Shared Docker Volume + RabbitMQ

## 7. Open Items chuyển tiếp sang Units Generation / Construction
- Mỗi microservice (bao gồm Orchestrator Service, RabbitMQ setup) sẽ trở thành 1 unit công việc riêng (Units Generation)
- Chi tiết dynamic plugin loading mechanism → Low-Level Design của Content Plugin Service unit
- Chi tiết RabbitMQ exchange/queue topology, retry/dead-letter policy → Low-Level Design + Infrastructure Design của Orchestrator Service unit
- Chi tiết quy ước đường dẫn shared volume → Low-Level Design / Infrastructure Design
- Schema database/state store cụ thể cho Saga state tại Orchestrator Service → Functional Design / NFR Design của Orchestrator Service unit
