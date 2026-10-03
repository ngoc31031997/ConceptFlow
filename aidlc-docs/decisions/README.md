# Architecture Decision Records

| ADR | Title | Status | Stage | Date |
|-----|-------|--------|-------|------|
| [ADR-0001](ADR-0001-microservices-architecture.md) | Microservices Architecture (Macro Decomposition) | Accepted | High-Level Design | 2026-08-04 |
| [ADR-0002](ADR-0002-hexagonal-architectural-style.md) | Hexagonal / Ports & Adapters as Architectural Style | Accepted | High-Level Design | 2026-08-04 |
| [ADR-0003](ADR-0003-technology-stack-direction.md) | Technology Stack Direction (Python/FastAPI + React) | Accepted | High-Level Design | 2026-08-04 |
| [ADR-0004](ADR-0004-api-gateway-decision.md) | API Gateway Decision | Accepted | High-Level Design | 2026-08-04 |
| [ADR-0005](ADR-0005-orchestration-via-gateway.md) | Orchestration Pattern via API Gateway (No Message Broker) | Superseded by ADR-0007 | High-Level Design | 2026-08-04 |
| [ADR-0006](ADR-0006-dynamic-plugin-loading.md) | Dynamic Plugin Loading for Content Plugin Service | Superseded — Content Plugin Service đã gỡ ở CR-020 (commit 59cb813) | Application Design | 2026-08-04 |
| [ADR-0007](ADR-0007-saga-orchestrator-service-message-queue.md) | Saga Orchestration via Dedicated Orchestrator Service + Message Queue | Accepted | Application Design | 2026-08-04 |
| [ADR-0008](ADR-0008-uri-api-versioning.md) | URI-based API Versioning (System-wide, `/v1/...`) | Accepted | Low-Level Design (Unit 2: Content Plugin Service — decision applies system-wide) | 2026-08-05 |
| [ADR-0009](ADR-0009-selective-polyglot-tech-stack.md) | Selective Polyglot Tech Stack (Go: Orchestrator, Node.js: Gateway, Python: rest) | Accepted | NFR Requirements (Unit 2: Content Plugin Service — decision applies system-wide, refines ADR-0003) | 2026-08-05 |
| [ADR-0010](ADR-0010-tts-engine-selection.md) | TTS Engine Selection — Piper for MVP | Superseded by ADR-0024 | Low-Level Design (Unit 3: TTS Service) | 2026-08-05 |
| [ADR-0011](ADR-0011-script-markdown-syntax.md) | Script Syntax — Markdown with Scene Delimiters | Accepted | Low-Level Design (Unit 4: Script Processing Service) | 2026-08-07 |
| [ADR-0012](ADR-0012-content-plugin-integration-via-orchestrator.md) | Content Plugin Integration via Orchestrator (Not Direct REST) | Superseded — Content Plugin Service đã gỡ ở CR-020 (commit 59cb813) | Low-Level Design (Unit 4: Script Processing Service) | 2026-08-07 |
| [ADR-0013](ADR-0013-postgresql-per-service-inbox-outbox.md) | PostgreSQL Per Service for Inbox/Outbox Pattern | Accepted | Cross-cutting retrofit (initiated during Unit 4 Low-Level Design) | 2026-08-07 |
| [ADR-0014](ADR-0014-tts-service-message-driven.md) | TTS Service Becomes Message-Driven (Own Saga Step) | Accepted | Cross-cutting retrofit (initiated during Unit 4 Low-Level Design); supersedes the TTS interface decisions made in Application Design (`component-methods.md`) and Unit 3's original Low-Level Design/NFR Design. | 2026-08-07 |
| [ADR-0015](ADR-0015-rendering-dynamic-template-loading.md) | Dynamic Plugin Loading for Rendering Service Animation Templates | Accepted | Low-Level Design (Unit 5: Rendering Service) | 2026-08-07 |
| [ADR-0016](ADR-0016-oauth-credential-storage-plaintext.md) | OAuth Credential Storage — Plaintext for MVP | Accepted | Low-Level Design (Unit 7: Publisher Service) | 2026-08-24 |
| [ADR-0017](ADR-0017-progress-delivery-via-rabbitmq-fanout.md) | Saga Progress Delivery to Gateway via RabbitMQ Fanout Exchange | Accepted | Low-Level Design (Unit 8: Orchestrator Service) | 2026-08-24 |
| [ADR-0018](ADR-0018-orchestrator-service-go-stack-confirmation.md) | Orchestrator Service — Go Stack Confirmation (chi, pgx, amqp091-go) | Accepted | NFR Requirements (Unit 8: Orchestrator Service) | 2026-08-31 |
| [ADR-0019](ADR-0019-orchestrator-outbox-for-commands-not-events.md) | Orchestrator's Outbox Publishes Commands, Not Events | Accepted | NFR Design (Unit 8: Orchestrator Service) | 2026-08-31 |
| [ADR-0020](ADR-0020-api-gateway-node-express-stack.md) | API Gateway — Node.js + Express Stack | Accepted | NFR Requirements (Unit 9: API Gateway) | 2026-08-31 |
| [ADR-0021](ADR-0021-web-gui-vite-vitest-toolchain.md) | Web GUI — Vite + Vitest Toolchain | Accepted | NFR Requirements (Unit 10: Web GUI) | 2026-08-31 |
| [ADR-0022](ADR-0022-amqp-connection-recovery.md) | AMQP Connection Recovery via ConnectionManager | Accepted | Operations / Bug Fix (Unit 8: Orchestrator Service) | 2026-09-06 |
| [ADR-0023](ADR-0023-cloud-tts-engine.md) | Dùng Google Cloud Text-to-Speech làm engine giọng đọc chính, giữ Piper làm fallback | Accepted | - | 2026-09-08 |
| [ADR-0024](ADR-0024-edge-tts-replaces-piper.md) | Edge Read Aloud thay Piper làm engine giọng đọc nền | Accepted | Low-Level Design (Unit 3 — TTS Service) | 2026-09-09 |
| [ADR-0025](ADR-0025-azure-and-edge-as-separate-voice-options.md) | Azure và Edge là hai lựa chọn giọng riêng, không thay thế ngầm cho nhau | Accepted | Low-Level Design (Unit 3 — TTS Service) | 2026-09-09 |
| [ADR-0026](ADR-0026-two-tier-oauth-apps-and-accounts.md) | Tách OAuth app và tài khoản YouTube thành hai tầng độc lập | Proposed | Low-Level Design | 2026-09-09 |
| [ADR-0027](ADR-0027-subtitle-delivery-per-surface.md) | Cách giao phụ đề là thuộc tính của bề mặt phát hành, không phải một toggle toàn cục | Proposed | Low-Level Design (CR-015) | 2026-09-09 |
| [ADR-0028](ADR-0028-force-ssl-scope-and-degradation.md) | Nâng scope lên youtube.force-ssl, và suy giảm êm cho kênh chưa nối lại | Proposed | Low-Level Design (CR-015) | 2026-09-09 |
| [ADR-0029](ADR-0029-authoring-service-boundary.md) | Tách authoring-service khỏi orchestrator | Accepted | Requirements Analysis → Construction (CR-040 FR111) | 2026-09-26 |
| [ADR-0030](ADR-0030-code-segments-and-v2-code-contract.md) | Lưu từng đoạn của bước Code trong authoring-service, llm-service chạy không trạng thái (contract `/v2/code/*`) | Accepted | Application Design (CR-050 Unit 2) | 2026-09-30 |
| [ADR-0031](ADR-0031-frame-orientation-by-output-mode.md) | Khung hình theo chế độ đầu ra; short dựng dọc riêng từ đầu | Accepted | Application Design (CR-060) | 2026-10-01 |
| [ADR-0032](ADR-0032-narration-lines-time-remotion-shots.md) | Câu thoại là đơn vị thời gian của shot Remotion | Accepted | Application Design (CR-067) | 2026-10-03 |
