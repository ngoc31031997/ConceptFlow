# Tech Stack Decisions — Unit 8: Orchestrator Service

## Language & Runtime: Go 1.22+
- **Ecosystem/library maturity**: `amqp091-go` (community-standard RabbitMQ client), `pgx` (mature, high-performance Postgres driver with native pooling), `chi` (lightweight idiomatic router) — all mature, actively maintained.
- **Performance**: goroutines give natural lightweight concurrency for consuming 12 AMQP event types in parallel without a thread-pool abstraction.
- **Team familiarity**: N/A new to this project scope, but Low-Level Design (commit `c60a06a`) already implemented the full module structure in Go — reversing this decision now would discard approved, implemented design work.
- **Long-term maintenance**: Go has a large community, stable LTS-style release cadence, wide hiring pool.
- **Licensing/cost**: all libraries are open-source (MIT/Apache-2.0), no cost implications.

## Framework: chi (HTTP router)
- Chosen over heavier frameworks (e.g. Gin, Echo) — chi is a minimal, idiomatic, stdlib-`net/http`-compatible router; Orchestrator's REST surface is small (4 endpoints), no need for a full-featured web framework.

## Postgres Driver: pgx
- Chosen over `database/sql` + `lib/pq` — pgx offers native connection pooling (`pgxpool`) and better performance, already used in LLD's `adapters/postgres/db.go`.

## Messaging Client: amqp091-go
- Chosen over alternatives (e.g. `streadway/amqp`, now unmaintained fork lineage — `amqp091-go` is its maintained successor) — standard choice for RabbitMQ in Go, consistent with Unit 1 (RabbitMQ Infrastructure)'s topology.

## Consistency with System-Wide Direction
`technology-direction.md` designates Go specifically for Orchestrator Service (ADR-0009, the deliberate polyglot exception alongside Node.js/Gateway) — this stage confirms rather than newly decides the language, since Low-Level Design was already built on it. See ADR-0018 for the formal record of this confirmation and the framework/library selections made within Go.

## Related ADR
See `aidlc-docs/decisions/ADR-0018-orchestrator-service-go-stack-confirmation.md`.
