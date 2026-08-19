# ADR-0013: PostgreSQL Per Service for Inbox/Outbox Pattern

## Status
Accepted

## Date
2026-08-07

## Stage
Cross-cutting retrofit (initiated during Unit 4 Low-Level Design)

## Context
Units 1–3 were built without any database — each records "Inbox/Outbox: Không áp dụng" because none of them writes business state that must stay atomically in sync with a published event, and idempotency was handled with an in-memory `set[message_id]` (Unit 2) or a shared-volume file existence check (Unit 3). The user's explicit goal for this project (see `project_conceptflow_goals.md`) is learning microservices/orchestrator patterns properly, not minimizing build effort — they asked to retrofit the Inbox/Outbox pattern system-wide, including the three already-completed units, as a deliberate exercise even where the current scale doesn't strictly require it. This requires picking one database technology and a "database per service" strategy up front, since every business service (Units 2, 3, 4, and future units) will need its own outbox/inbox tables.

## Options Considered
### Option A: SQLite, one file per service
- What it is: Each service owns a local `.db` file (Docker named volume for persistence across restarts), no separate database container.
- Strengths: Simplest possible setup for a local single-machine Docker Compose project — no extra containers, no connection pooling concerns, trivial to inspect (single file).
- Trade-offs: Not representative of how production microservices typically run their database-per-service (client-server DB, not embedded) — weaker as a learning exercise for the orchestrator/microservices pattern the user is optimizing for.

### Option B: PostgreSQL, one container per service (Chosen)
- What it is: Each business service gets its own PostgreSQL container + named volume in `docker-compose.yml` (e.g., `content-plugin-db`, `tts-db`, `script-processing-db`), matching true "database per service" microservices practice.
- Strengths: Much closer to how this pattern is actually implemented in production systems (connection pooling, SQL client library, migrations, transactions) — directly serves the user's stated learning goal; sets a consistent, reusable pattern for all future business-service units (5, 6, 7, 8).
- Trade-offs: More containers running simultaneously on the developer's machine (one Postgres per business service, on top of RabbitMQ) — accepted, since local resource cost is secondary to the pedagogical goal here.

## Decision
PostgreSQL, one container per business service that needs Inbox/Outbox (Units 2, 3, 4, and future message-driven units), each with its own named volume.

## Rationale
The user explicitly confirmed this trade-off (client-server DB per service vs. embedded SQLite) in favor of PostgreSQL, prioritizing fidelity to real microservices practice over minimizing local resource usage — consistent with the project's standing goal of learning the pattern properly (`project_conceptflow_goals.md`).

## Consequences
- **Positive**: Consistent, realistic database-per-service pattern reusable across all future business-service units; real SQL transactions available for atomic outbox writes.
- **Negative / Accepted Trade-offs**: More containers to run locally; each service needs a DB client library, connection config, and a migration/bootstrap step it didn't need before.
- **Follow-ups**: Units 2 and 3 (already built) need their NFR Design, Infrastructure Design, and Code Generation revisited to add this. Unit 4 (in progress) adopts it from the start. Future units (5–8) should default to this pattern when they need Inbox/Outbox.

## Related
- Design artifact: retrofit plan (session-local), applied first in `aidlc-docs/construction/script-processing-service/`
- Related ADRs: None (new infrastructure decision)
