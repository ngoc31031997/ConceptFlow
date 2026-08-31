# ADR-0021: Web GUI — Vite + Vitest Toolchain

## Status
Accepted

## Date
2026-08-31

## Stage
NFR Requirements (Unit 10: Web GUI)

## Context
`technology-direction.md` (Inception, ADR-0003) already designates React for the Web GUI, but did not choose a specific build tool or testing framework — those decisions were deferred to this unit's own design stages. Low-Level Design (approved) already selected Vite as the build tool (LLD Question 8). NFR Requirements stage needed to confirm the testing toolchain to pair with it.

## Options Considered
### Option A: Vite + Vitest + React Testing Library (Chosen)
- What it is: Vitest is Vite's own test runner, sharing its config/transform pipeline; React Testing Library for component-level assertions.
- Strengths: No separate Babel/ts-jest configuration needed — Vitest reuses Vite's esbuild-based transform directly; faster test runs than Jest in a Vite project; officially recommended pairing by the Vite team; Jest-compatible API keeps the learning curve low.
- Trade-offs: Smaller/younger ecosystem than Jest — some Jest-specific plugins may not have Vitest equivalents, though none are needed for this project's simple 3-page SPA.

### Option B: Jest + React Testing Library
- What it is: The long-established testing framework for React, paired with a separate Babel or ts-jest transform configuration since Jest doesn't natively understand Vite's build pipeline.
- Strengths: Larger ecosystem, most tutorials/examples target Jest, very mature.
- Trade-offs: Requires maintaining a second, separate transform configuration alongside Vite's own — extra setup and a potential source of config drift for a single-developer project with no need for Jest-specific plugins.

## Decision
Chọn **Option A: Vite + Vitest + React Testing Library**.

## Rationale
Vì đã chọn Vite làm build tool (LLD), dùng chung Vitest tránh phải duy trì 2 hệ thống transform/config riêng biệt (Vite cho dev/build, Babel/ts-jest cho test) — giảm bề mặt cấu hình cho 1 người phát triển duy nhất.

## Consequences
- **Positive**: Cấu hình test đơn giản, nhanh, dùng chung pipeline với Vite.
- **Negative / Accepted Trade-offs**: Ecosystem Vitest nhỏ hơn Jest — chấp nhận được vì project không cần plugin đặc thù nào của Jest.
- **Follow-ups**: Không có.

## Related
- Design artifact: `aidlc-docs/construction/web-gui/nfr-requirements/tech-stack-decisions.md`
- Related ADRs: ADR-0003 (chọn React cho Frontend GUI, Inception)
