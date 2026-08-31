# ADR-0020: API Gateway — Node.js + Express Stack

## Status
Accepted

## Date
2026-08-31

## Stage
NFR Requirements (Unit 9: API Gateway)

## Context
`technology-direction.md` (Inception, ADR-0009) designates Node.js for API Gateway — the deliberate second polyglot exception alongside Orchestrator's Go, chosen for the reverse-proxy/event-loop model's real-world fit for gateway workloads. Low-Level Design for Unit 9 (approved) already assumes a small Express-style route/handler/client layering. NFR Requirements stage needed to confirm the specific web framework within Node.js, since ADR-0009 named the language but not the framework.

## Options Considered
### Option A: Express 4.x (Chosen)
- What it is: The most widely used Node.js web framework, minimal middleware pipeline used only for correlation-ID injection and routing.
- Strengths: Largest ecosystem/community/documentation for the reverse-proxy pattern specifically; predictable, stable API; more than sufficient performance at this project's single-user local scale.
- Trade-offs: Lower raw throughput than newer frameworks under heavy load — irrelevant here since there is no load target beyond one Creator's browser.

### Option B: Fastify
- What it is: A newer, schema-validation-first Node.js framework optimized for high throughput.
- Strengths: Meaningfully faster in benchmarks; built-in JSON schema validation.
- Trade-offs: The throughput advantage has no bearing on this project's actual load; schema validation is not needed since the Gateway deliberately does not re-validate payloads (validation belongs to downstream services per the zero-trust-at-the-edges design already established); smaller ecosystem for simple proxy-pattern examples than Express.

## Decision
Chọn **Option A: Express 4.x**.

## Rationale
Ở quy mô dự án (1 Creator, chạy local), lợi thế throughput của Fastify không mang lại giá trị thực tế, trong khi Express phổ biến hơn cho đúng pattern Gateway đang xây (reverse-proxy + SSE bridge), giúp việc tham khảo/bảo trì dễ hơn cho 1 người phát triển.

## Consequences
- **Positive**: Đơn giản, tài liệu/ví dụ phong phú, đủ nhanh cho use case thực tế.
- **Negative / Accepted Trade-offs**: Nếu sau này scale multi-tenant/production, throughput của Express có thể trở thành điểm nghẽn — ngoài phạm vi hiện tại, cần revisit nếu threat model/scale thay đổi.
- **Follow-ups**: Không có.

## Related
- Design artifact: `aidlc-docs/construction/api-gateway/nfr-requirements/tech-stack-decisions.md`
- Related ADRs: ADR-0009 (selective polyglot, gốc quyết định Node.js cho Gateway)
