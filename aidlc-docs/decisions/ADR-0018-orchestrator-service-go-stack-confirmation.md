# ADR-0018: Orchestrator Service — Go Stack Confirmation (chi, pgx, amqp091-go)

## Status
Accepted

## Date
2026-08-31

## Stage
NFR Requirements (Unit 8: Orchestrator Service)

## Context
`technology-direction.md` (Inception phase, ADR-0009) already designates Go as the deliberate polyglot exception for Orchestrator Service, driven by its role as a concurrency-heavy Saga coordinator. Low-Level Design for Unit 8 (approved, commit `c60a06a`) already implemented the full module structure on Go, using `chi` (HTTP router), `pgx` (Postgres driver), and `amqp091-go` (RabbitMQ client), without a dedicated ADR recording the framework/library-level choices within Go. NFR Requirements stage requires an explicit tech-stack confirmation with trade-offs presented, per workflow rules — this ADR formalizes that record retroactively, aligned with what LLD already built.

## Options Considered
### Option A: Go + chi + pgx + amqp091-go (Chosen)
- What it is: Confirm the stack already used throughout Low-Level Design — `chi` for the small (4-endpoint) REST surface, `pgx` for Postgres access with native pooling, `amqp091-go` for RabbitMQ.
- Strengths: 100% consistent with the approved Low-Level Design; matches system-wide direction (ADR-0009); goroutines are a natural fit for concurrently consuming 12 event types; all libraries mature and actively maintained; open-source, no licensing cost.
- Trade-offs: Go remains the only non-Python service besides the Node.js Gateway — an accepted, deliberate polyglot choice already made at Inception (ADR-0009), not a new trade-off introduced here.

### Option B: Revert to Python/FastAPI (system-wide default) for consistency
- What it is: Discard the Go-based Low-Level Design and rebuild Orchestrator Service in Python/FastAPI, matching the majority of other units.
- Strengths: Single-language system, lower cognitive overhead for anyone maintaining the whole codebase.
- Trade-offs: Directly contradicts ADR-0009's explicit rationale (Saga coordinator benefits from Go's concurrency model); discards a fully-designed and approved Low-Level Design (module structure, interface contracts, sequence flows) for no functional gain; Python/FastAPI's async model can achieve similar concurrency but was already deliberately rejected for this unit at Inception.

## Decision
Chọn **Option A**: xác nhận chính thức Go 1.22+ + chi + pgx + amqp091-go cho Orchestrator Service.

## Rationale
Đây là quyết định đã chốt từ Inception (ADR-0009) dựa trên đặc thù vai trò Saga coordinator; Low-Level Design đã hiện thực hóa đầy đủ trên nền tảng này và được duyệt. NFR Requirements stage xác nhận lại (không đảo ngược) lựa chọn này, đồng thời ghi nhận chính thức các thư viện cụ thể trong Go ecosystem (chi/pgx/amqp091-go) mà LLD đã ngầm định chọn nhưng chưa có ADR riêng.

## Consequences
- **Positive**: Nhất quán toàn bộ chuỗi thiết kế đã duyệt (Inception → LLD → NFR), không có rework.
- **Negative / Accepted Trade-offs**: Go là ngoại lệ polyglot duy nhất ngoài Gateway — team (dù chỉ 1 người) cần duy trì kiến thức Go song song Python cho phần lớn hệ thống. Đã được chấp nhận từ ADR-0009.
- **Follow-ups**: Không có.

## Related
- Design artifact: `aidlc-docs/construction/orchestrator-service/nfr-requirements/tech-stack-decisions.md`
- Related ADRs: ADR-0009 (selective polyglot tech stack, gốc quyết định Go cho Orchestrator), ADR-0007 (Saga orchestration-based)
