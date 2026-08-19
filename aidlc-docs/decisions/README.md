# Architecture Decision Records

| ADR | Title | Status | Stage | Date |
|-----|-------|--------|-------|------|
| [ADR-0001](ADR-0001-microservices-architecture.md) | Microservices Architecture (Macro Decomposition) | Accepted | High-Level Design | 2026-08-04 |
| [ADR-0002](ADR-0002-hexagonal-architectural-style.md) | Hexagonal / Ports & Adapters as Architectural Style | Accepted | High-Level Design | 2026-08-04 |
| [ADR-0003](ADR-0003-technology-stack-direction.md) | Technology Stack Direction (Python/FastAPI + React) | Accepted | High-Level Design | 2026-08-04 |
| [ADR-0004](ADR-0004-api-gateway-decision.md) | API Gateway Decision | Accepted | High-Level Design | 2026-08-04 |
| [ADR-0005](ADR-0005-orchestration-via-gateway.md) | Orchestration Pattern via API Gateway (No Message Broker) | Superseded by ADR-0007 | High-Level Design | 2026-08-04 |
| [ADR-0006](ADR-0006-dynamic-plugin-loading.md) | Dynamic Plugin Loading for Content Plugin Service | Accepted | Application Design | 2026-08-04 |
| [ADR-0007](ADR-0007-saga-orchestrator-service-message-queue.md) | Saga Orchestration via Dedicated Orchestrator Service + Message Queue | Accepted | Application Design | 2026-08-04 |
| [ADR-0008](ADR-0008-uri-api-versioning.md) | URI-based API Versioning (System-wide, `/v1/...`) | Accepted | Low-Level Design (Unit 2) | 2026-08-05 |
| [ADR-0009](ADR-0009-selective-polyglot-tech-stack.md) | Selective Polyglot Tech Stack (Go: Orchestrator, Node.js: Gateway, Python: rest) | Accepted | NFR Requirements (Unit 2) | 2026-08-05 |
| [ADR-0010](ADR-0010-tts-engine-selection.md) | TTS Engine Selection — Piper for MVP | Accepted | Low-Level Design (Unit 3) | 2026-08-05 |
| [ADR-0011](ADR-0011-script-markdown-syntax.md) | Script Syntax — Markdown with Scene Delimiters | Accepted | Low-Level Design (Unit 4) | 2026-08-07 |
| [ADR-0012](ADR-0012-content-plugin-integration-via-orchestrator.md) | Content Plugin Integration via Orchestrator (Not Direct REST) | Accepted | Low-Level Design (Unit 4) | 2026-08-07 |
| [ADR-0013](ADR-0013-postgresql-per-service-inbox-outbox.md) | PostgreSQL Per Service for Inbox/Outbox Pattern | Accepted | Cross-cutting retrofit | 2026-08-07 |
| [ADR-0014](ADR-0014-tts-service-message-driven.md) | TTS Service Becomes Message-Driven (Own Saga Step) | Accepted | Cross-cutting retrofit | 2026-08-07 |
