# High-Level Design — Consolidated Overview

**Dự án**: Manim-based Educational Video Generation Tool
**Ngày**: 2026-08-04

Tài liệu này tổng hợp các artifact High-Level Design chi tiết. Xem từng file riêng để biết chi tiết đầy đủ.

## 1. System Context
1 actor (Creator), 1 hệ thống ngoài (YouTube Data API), hệ thống chạy trong docker-compose trên 1 máy cá nhân.
→ Chi tiết: `system-context.md`

## 2. Architecture Overview
Kiến trúc **Microservices** (ADR-0001) gồm 10 thành phần: Web GUI, API Gateway, **Orchestrator Service** (mới), **Message Queue/RabbitMQ** (mới), Content Plugin Service, Script Processing Service, Rendering Service, TTS Service, Video Assembly Service, Publisher Service.
→ Chi tiết: `architecture-overview.md`

## 3. Technology Direction
Backend: Python/FastAPI (mỗi service) — Frontend: React — Giao tiếp: REST (cấu hình) + SSE (tiến trình render) — Containerization: Docker/docker-compose.
→ Chi tiết: `technology-direction.md`

## 4. Integration Boundaries
GUI↔Gateway: REST (cấu hình) + SSE (tiến trình). Gateway↔Orchestrator: REST (khởi tạo Saga). **Orchestrator↔Service nghiệp vụ: Message Queue/RabbitMQ, Saga orchestration-based** (ADR-0007, supersedes ADR-0005); mỗi data entity thuộc sở hữu đúng 1 service (không shared database); Saga state thuộc sở hữu Orchestrator Service.
→ Chi tiết: `integration-boundaries.md`

## 5. Architectural Style
**Hexagonal / Ports & Adapters** (ADR-0002) áp dụng nội bộ mỗi service, đặc biệt Content Plugin Service, Rendering Service, TTS Service — phục vụ trực tiếp yêu cầu NFR1 (Extensibility). Constructor injection thủ công cho DI.
→ Chi tiết: `architectural-style.md`

## 6. Key Architectural Decisions (ADRs)
| ADR | Quyết định |
|---|---|
| ADR-0001 | Microservices thay vì Modular Monolith |
| ADR-0002 | Hexagonal/Ports & Adapters thay vì Layered hoặc DDD |
| ADR-0003 | Python/FastAPI + React thay vì Desktop app Python thuần |
| ADR-0004 | API Gateway (reverse proxy) thay vì gọi trực tiếp từng service |
| ADR-0005 | *(Superseded by ADR-0007)* Orchestration qua Gateway thay vì Choreography/message broker |
| ADR-0006 | Dynamic plugin loading thay vì static registry |
| ADR-0007 | Saga Orchestration qua Orchestrator Service riêng + Message Queue (RabbitMQ), thay thế ADR-0005 |

## 7. Traceability to Requirements
Mọi Functional Requirement (FR1-FR8) trong `requirements.md` được ánh xạ tới ít nhất 1 microservice trong `architecture-overview.md`; NFR1 (Extensibility) là driver chính cho Architectural Style; NFR3 (Docker local) là driver chính cho Deployment Topology (docker-compose, 1 máy cá nhân).

## 8. Open Items chuyển tiếp sang Application Design
- Định nghĩa cụ thể REST endpoint contract giữa GUI ↔ Gateway ↔ mỗi service
- Định nghĩa port/adapter interface cụ thể cho Content Plugin Service, TTS Service, Rendering Service
- Định nghĩa state machine của "video project" mà Gateway quản lý (draft/processing/failed-at-step/published)
- Xác định cơ chế lưu trữ file trung gian (animation clip, audio clip) giữa các service (shared volume trong docker-compose)
