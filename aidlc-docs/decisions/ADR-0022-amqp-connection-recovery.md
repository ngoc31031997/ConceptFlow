# ADR-0022: AMQP Connection Recovery via ConnectionManager

## Status
Accepted

## Date
2026-09-06

## Stage
Operations / Bug Fix (Unit 8: Orchestrator Service)

## Context
Orchestrator opens a single `amqp091-go` connection + channel once at
startup (`main.go`) and hands the raw `*amqp.Channel` pointer directly to
`amqp.Publisher` (used by `postgres.OutboxRelay`) and `amqp.Consumer`.
Neither type had any recovery logic. In production this channel was closed
at the broker side (RabbitMQ container restart), and every subsequent
`PublishCommand` call failed forever with `channel/connection is not
open` — the outbox relay looped on the same dead channel every
`OUTBOX_POLL_INTERVAL_MS` with no way to recover, permanently stalling
command dispatch (observed: new video projects never advanced past their
first Saga step). `Consumer`'s delivery loops also died silently, since
their Go channel closes when the underlying AMQP channel closes — no more
Saga events or DLQ messages were consumed either. The only recovery was a
manual `docker restart orchestrator`.

## Options Considered
### Option A: ConnectionManager owns reconnect-with-backoff (Chosen)
- What it is: a new `amqp.ConnectionManager` owns the `Connection`/`Channel`
  lifecycle behind a mutex. `Publisher` and `Consumer` ask it for the
  current channel on every use instead of caching one. A background
  watchdog goroutine selects on `conn.NotifyClose`/`channel.NotifyClose`
  and redials with capped exponential backoff (`RABBITMQ_RECONNECT_INITIAL_DELAY_MS`
  default 1s → `RABBITMQ_RECONNECT_MAX_DELAY_MS` cap 30s), then runs
  registered `OnReconnect` callbacks so `Consumer` re-issues its `Consume`
  registrations.
- Strengths: fully automatic recovery with no process restart; topology
  (exchanges/queues) doesn't need re-declaring since it's owned by the
  `rabbitmq-infrastructure` unit and is durable; backoff prevents a
  reconnect storm against a broker that's still coming back up; keeps
  `Publisher`/`Consumer` unaware of reconnect mechanics (they just call
  `Channel()`), so their existing unit tests and interfaces
  (`CommandPublisherPort`/`ProgressPublisherPort`) didn't need to change.
- Trade-offs: adds a new stateful component with its own concurrency
  (mutex-guarded channel swap) that needs its own tests; a message
  in-flight at the exact moment of disconnect is lost from the AMQP
  channel's perspective, but this is already covered by existing
  guarantees — the Outbox row stays `published_at IS NULL` and is retried
  by the relay, and undelivered events are redelivered by the broker to a
  new consumer once RabbitMQ's own connection timeout elapses.

### Option B: Per-call redial (open a fresh channel for every publish)
- What it is: `Publisher.PublishCommand` dials a new connection/channel per
  call instead of reusing one.
- Strengths: trivially simple, no watchdog or shared mutable state.
- Trade-offs: a full AMQP handshake (TCP + protocol negotiation) per
  command is far too slow under the relay's 500ms poll cadence and would
  multiply load on RabbitMQ; does nothing for `Consumer`, which fundamentally
  needs a long-lived channel to receive pushed deliveries.

### Option C: Rely on an external supervisor to restart the container
- What it is: keep today's behavior (fail hard on channel death) and let
  Docker's restart policy or an ops runbook restart the `orchestrator`
  container when it wedges.
- Strengths: zero code change.
- Trade-offs: this was the actual incident — restart is manual and slow
  (minutes of stalled sagas before someone notices/intervenes); even with
  an automatic restart policy, a full process restart re-establishes
  in-memory state and reprocesses more than a channel-level reconnect
  needs to, and does not distinguish a transient network blip (which
  self-heals in seconds) from a real crash.

## Decision
Chose **Option A**: `ConnectionManager` in `internal/adapters/amqp/`
owns reconnect-with-backoff; `Publisher` and `Consumer` fetch the live
channel from it rather than holding one directly.

## Rationale
The failure mode was specifically "channel dies, nothing notices" — the
fix belongs at the connection layer, not the caller. Centralizing recovery
in one component keeps `Publisher`/`Consumer` simple and testable, matches
the existing "constructed directly, not a business port" treatment of the
AMQP connection in `dependency-injection.md`, and avoids the latency and
redundant-declaration cost of Option B while fixing the actual incident
faster and more precisely than Option C's container-restart fallback.

## Consequences
- **Positive**: broker restarts/network blips no longer require manual
  intervention; `orchestrator.events` and all 6 DLQ queues automatically
  regain consumers after a reconnect; outbox commands resume publishing
  without operator action.
- **Negative / Accepted Trade-offs**: one more moving part (watchdog
  goroutine, mutex-guarded channel) to reason about and test; a message
  published in the exact window between disconnect and reconnect still
  needs the existing Outbox/Inbox retry paths to catch it (this was already
  true before — no new gap introduced).
- **Follow-ups**: none required immediately; if a future unit acquires a
  similarly long-lived AMQP publisher/consumer (none do today per the
  Explore pass for this ADR), consider extracting `ConnectionManager` into
  a shared module instead of reimplementing it per service.

## Related
- Design artifacts: `aidlc-docs/construction/orchestrator-service/low-level-design/dependency-injection.md`,
  `aidlc-docs/construction/orchestrator-service/nfr-design/messaging-design.md`
- Related ADRs: ADR-0019 (Orchestrator's Outbox/Inbox semantics), ADR-0007
  (Saga orchestration via message queue)
