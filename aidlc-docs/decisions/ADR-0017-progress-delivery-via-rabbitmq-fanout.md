# ADR-0017: Saga Progress Delivery to Gateway via RabbitMQ Fanout Exchange

## Status
Accepted

## Date
2026-08-24

## Stage
Low-Level Design (Unit 8: Orchestrator Service)

## Context
`component-methods.md` specifies that API Gateway's `GET /projects/{id}/events` (SSE) "forwards events received from the Orchestrator Service", but never specified the transport mechanism between Orchestrator and Gateway — Gateway (Unit 9) is not yet built, so this had to be decided now as a cross-unit constraint. Orchestrator needs to push real-time Saga progress (per-step status, per-scene render progress) to whichever Gateway instance(s) are running.

## Options Considered
### Option A: Orchestrator hosts its own SSE endpoint, Gateway reverse-proxies it
- What it is: Orchestrator exposes `GET /v1/projects/{project_id}/events` itself (Go's `net/http` streaming response); Gateway's SSE route is a thin reverse proxy to this endpoint.
- Strengths: Simplest to implement — Gateway needs no pub/sub logic of its own, matching its "routing only" role from `components.md`.
- Trade-offs: Orchestrator now owns long-lived HTTP connections per active GUI session, coupling its process lifecycle to client connection lifecycle; every Gateway instance proxying the same project's stream would each open a separate connection to Orchestrator (no fan-out sharing).

### Option B: Orchestrator publishes progress via a RabbitMQ fanout exchange, Gateway subscribes and manages SSE fan-out (Chosen)
- What it is: New exchange `progress.fanout` (fanout type). Orchestrator publishes a progress message after handling each Saga step event. Gateway declares its own queue bound to this exchange and fans messages out to the correct SSE client connections by `project_id`.
- Strengths: Fully decouples Orchestrator from HTTP connection management — consistent with the message-driven philosophy already used everywhere else (ADR-0007); Orchestrator's goroutines never block on slow/disconnected GUI clients; multiple Gateway instances (if ever needed) can each bind their own queue to the same fanout exchange without Orchestrator knowing or caring how many exist.
- Trade-offs: Gateway (Unit 9) must implement its own AMQP consumer + SSE fan-out logic, adding responsibility beyond pure request routing — a deliberate, accepted deviation from `components.md`'s "Gateway chỉ còn routing" framing, revised by this ADR.

## Decision
Chọn **Option B**: RabbitMQ fanout exchange `progress.fanout`. Orchestrator Service publishes; Gateway Service (Unit 9) will declare and bind its own queue when built.

## Rationale
Toàn hệ thống đã cam kết message-driven cho mọi giao tiếp liên-service kể từ ADR-0007 — giữ Orchestrator hoàn toàn tách khỏi việc quản lý kết nối HTTP dài hạn (SSE) nhất quán với triết lý đó, và tránh Orchestrator's goroutine bị block bởi 1 GUI client kết nối chậm/rớt mạng. Chi phí thêm ở Gateway (Unit 9) là chấp nhận được — Gateway vốn đã cần hiểu SSE contract để phục vụ GUI, chỉ là nguồn dữ liệu chuyển từ "proxy HTTP" sang "consume AMQP".

## Consequences
- **Positive**: Orchestrator không giữ state kết nối GUI nào; hỗ trợ tự nhiên nhiều Gateway instance sau này (mỗi instance tự bind queue riêng) mà không cần thay đổi gì ở Orchestrator; nhất quán triết lý message-driven toàn hệ thống.
- **Negative / Accepted Trade-offs**: Gateway (Unit 9) cần thêm AMQP client + SSE fan-out logic — trách nhiệm lớn hơn so với mô tả "chỉ còn routing" ban đầu ở `components.md` (đã ghi nhận là revision chấp nhận được).
- **Follow-ups**: `components.md`'s mô tả Gateway ("Không còn quản lý state machine hay điều phối tuần tự") vẫn đúng — chỉ bổ sung trách nhiệm consume `progress.fanout` + SSE fan-out, không quay lại quản lý Saga. Khi thiết kế Unit 9 (API Gateway), tham chiếu ADR này để implement consumer cho `progress.fanout`.

## Related
- Design artifact: `aidlc-docs/construction/rabbitmq-infrastructure/nfr-design/logical-components.md` (revised — new `progress.fanout` exchange), `infra/rabbitmq/definitions.json` (revised)
- Related ADRs: Extends ADR-0007 (Saga orchestration via message queue), ADR-0009 (Go for Orchestrator — goroutines benefit from not blocking on SSE connections)
