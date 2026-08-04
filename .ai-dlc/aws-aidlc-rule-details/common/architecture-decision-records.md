# Architecture Decision Records (ADR)

## Purpose
Capture significant architecture/technology decisions — what was decided, why, what alternatives were considered, and what trade-offs were accepted — in a durable, individually addressable record, separate from the design document that triggered it. Design documents describe the resulting shape of the system; ADRs preserve the reasoning behind a specific decision so it can be revisited later without re-deriving it.

## When to Create an ADR (MANDATORY)
Create one ADR whenever a stage records a decision that meets ANY of these criteria:
- The decision was presented to the user as a multiple-choice trade-off question (per `common/question-format-guide.md`) covering a technology, architectural style, or mechanism choice
- The decision would be costly or disruptive to reverse later (e.g., choice of architectural style, CRUD vs. CQRS, Saga orchestration vs. choreography, database technology, read/write splitting, API Gateway product, messaging broker, caching strategy)
- The user picked "Other" and specified a non-standard approach

Do NOT create an ADR for routine implementation details that don't involve a real trade-off (e.g., a single obvious naming choice, a detail with no viable alternative).

**Stages that typically produce ADRs**:
- High-Level Design: architectural style, distributed system communication pattern, API Gateway decision, technology direction
- Application Design: component boundary decisions with significant alternatives considered
- NFR Requirements: language/runtime selection, framework selection (always — this is inherently a costly-to-reverse decision)
- NFR Design: CRUD vs. CQRS, caching strategy, event-driven design, Saga pattern, Inbox/Outbox pattern
- Infrastructure Design: database read/write splitting, load balancer choice, API Gateway product, cloud provider/service selection
- Low-Level Design: layering/dependency-injection approach when multiple viable structures were considered

## File Location and Naming
- Directory: `aidlc-docs/decisions/`
- Filename: `ADR-{NNNN}-{kebab-case-title}.md` (e.g., `ADR-0001-clean-architecture-for-order-service.md`)
- Numbering: sequential across the whole project, zero-padded to 4 digits, never reused even if an ADR is later superseded

## ADR Template
```markdown
# ADR-{NNNN}: {Decision Title}

## Status
{Proposed | Accepted | Superseded by ADR-{NNNN}}

## Date
{ISO 8601 date}

## Stage
{High-Level Design | Application Design | NFR Design | Infrastructure Design | Low-Level Design}

## Context
{What situation/requirement/constraint led to this decision needing to be made. 1-3 sentences.}

## Options Considered
### Option A: {Name}
- What it is: {brief}
- Strengths: {...}
- Trade-offs: {...}

### Option B: {Name}
- What it is: {brief}
- Strengths: {...}
- Trade-offs: {...}

{...additional options as evaluated in the originating question...}

## Decision
{Which option was chosen, stated plainly}

## Rationale
{Why this option won over the others, referencing the specific context/constraints — not a restatement of its generic strengths}

## Consequences
- **Positive**: {what this decision enables or improves}
- **Negative / Accepted Trade-offs**: {what this decision costs or constrains going forward}
- **Follow-ups**: {any future work this decision implies, if applicable}

## Related
- Design artifact: `{path to the design doc this decision came from}`
- Related ADRs: {links, if any}
```

## Execution Steps (for any stage that produces a decision)
1. After the user answers a MANDATORY trade-off question and the choice is recorded in the stage's design artifact, check whether it meets the "When to Create an ADR" criteria above
2. If yes: create `aidlc-docs/decisions/ADR-{NNNN}-{title}.md` using the template, filling in the actual options/strengths/trade-offs already presented to the user in the question file — do not re-derive them, reuse what was already written
3. Set Status to `Accepted` once the stage itself is approved by the user (not before)
4. If a later stage changes a prior decision, create a NEW ADR with Status `Accepted`, and update the old ADR's Status to `Superseded by ADR-{NNNN}` — never edit/delete a past ADR's original content
5. Reference the ADR number from the stage's own design document (e.g., "See ADR-0003 for the rationale behind this choice") so both directions are linked
6. **MANDATORY**: Log ADR creation in `aidlc-docs/audit.md` with the ADR number and title

## Directory Structure Addition
```text
aidlc-docs/
├── decisions/                      # Architecture Decision Records (ADRs)
│   ├── ADR-0001-{title}.md
│   ├── ADR-0002-{title}.md
│   └── ...
```

## Index File
Maintain `aidlc-docs/decisions/README.md` as a running index:
```markdown
# Architecture Decision Records

| ADR | Title | Status | Stage | Date |
|-----|-------|--------|-------|------|
| [ADR-0001](ADR-0001-title.md) | {title} | Accepted | High-Level Design | {date} |
```
Update this index every time a new ADR is created or an existing ADR's status changes.
