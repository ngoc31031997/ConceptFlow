# Repository Layer Summary — Unit 8: Orchestrator Service

## Location
`services/orchestrator/internal/adapters/postgres/`

## Files
| File | Responsibility |
|---|---|
| `db.go` | `NewPool` — opens `pgxpool.Pool`, runs `CREATE TABLE IF NOT EXISTS` for `projects`, `saga_steps`, `outbox_events`, `processed_messages` at startup (no separate migration tool, matches Units 2-7). |
| `project_repository.go` | `ProjectRepository` implements `domain.ProjectRepositoryPort` — `Get`/`Save` (upsert) on `projects`, `GetStep`/`UpdateStep` (upsert) on `saga_steps`. `Scenes`/`YoutubeTags` stored as `JSONB`, (de)serialized with `encoding/json`. |
| `inbox.go` | `InboxRepository` — `HasProcessed`/`MarkProcessed` against `processed_messages`, dedupes incoming event `message_id` (Rule 9). |
| `outbox.go` | `OutboxRepository` implements `domain.CommandPublisherPort.PublishCommand` by **enqueueing** into `outbox_events` (no network I/O) — see ADR-0019 comment in the file. Also exposes `FetchUnpublished`/`MarkPublished` for the relay. |
| `relay.go` | `OutboxRelay.Run(ctx)` — background goroutine, polls `outbox_events` every `OUTBOX_POLL_INTERVAL_MS` (default 500ms), publishes each pending row via an injected `domain.CommandPublisherPort` (the real `amqp.Publisher`), marks `published_at`. |

## ADR-0019 semantic note
Two distinct implementations of `domain.CommandPublisherPort` exist by design:
1. `postgres.OutboxRepository` — injected into all 4 use cases; `PublishCommand` only writes to `outbox_events` (crash-safe, no AMQP call).
2. `amqp.Publisher` — injected into `OutboxRelay`; `PublishCommand` performs the actual AMQP publish.

This lets a use case's "publish a command" call return only after the state change + queue write are durably committed, while the real network send happens asynchronously from the relay — matching module-structure.md's "gọi CommandPublisherPort.PublishCommand (qua Outbox)".

## Schema
See `db.go`'s `schema` constant — `outbox_events` and `processed_messages` match `messaging-design.md`'s reference SQL exactly; `projects`/`saga_steps` are Orchestrator-specific additions per `domain-entities.md`.

## Testing
`project_repository_test.go` covers only pure-Go mapping logic (JSON round-trip of `[]domain.Scene`, `ProjectStatus`/`SagaStepStatus`/`VoiceLanguage`/`Visibility` string conversions) — no live Postgres connection. Integration testing against a real database is left to the Build & Test stage.
