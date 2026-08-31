# Orchestrator Service (Unit 8)

Saga orchestrator for the ConceptFlow video-generation pipeline. Coordinates the Render Pipeline Saga (5 steps: parse script → classify scenes → synthesize speech → render scenes → assemble video) and the Publish Saga (1 step: publish video), by consuming events from the 6 upstream business services over RabbitMQ, aggregating data onto a `Project` aggregate, and dispatching the next command — without containing any of the business logic those services own.

## Overview
- **Language/Stack**: Go 1.22, `chi` (HTTP router), `pgx/v5` (Postgres driver), `amqp091-go` (RabbitMQ client) — see ADR-0018.
- **Architecture**: Hexagonal (Ports & Adapters) — `internal/domain` has no infrastructure imports; `internal/application` (4 use cases) depends only on 3 domain port interfaces; `internal/adapters/{http,amqp,postgres,logging}` implement those ports / the transport layer.
- **Persistence**: own PostgreSQL database (`orchestrator-db`, database-per-service — ADR-0013), schema bootstrapped at startup (`CREATE TABLE IF NOT EXISTS`).
- **Messaging**: at-least-once delivery — Inbox (`processed_messages`) dedupes incoming events, Outbox (`outbox_events`) guarantees outgoing **command** delivery (ADR-0019 — this Outbox holds commands, not events, unlike every other unit in this system).

## Prerequisites
- Go 1.22+
- A reachable PostgreSQL instance and RabbitMQ instance (or run everything via the root `docker-compose.yml`).

## Running

### `go run` (local, against already-running Postgres/RabbitMQ)
```bash
cd services/orchestrator
export RABBITMQ_URL=amqp://guest:guest@localhost:5672/
export DATABASE_URL=postgresql://postgres:postgres@localhost:5432/orchestrator
go run ./cmd/orchestrator
```

### Docker (via root docker-compose)
```bash
docker compose up orchestrator-db orchestrator
```
The service listens on port 8000 inside the container (`GET /health` for the compose healthcheck).

## Environment Variables
| Variable | Required | Default | Description |
|---|---|---|---|
| `RABBITMQ_URL` | yes | — | AMQP connection string |
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `OUTBOX_POLL_INTERVAL_MS` | no | `500` | How often `OutboxRelay` polls `outbox_events` for unpublished commands |
| `DATABASE_MAX_CONNS` | no | `10` | `pgxpool` max connections |
| `HTTP_PORT` | no | `8000` | HTTP listen port |

## REST API
| Method | Path | Purpose |
|---|---|---|
| `GET` | `/health` | Liveness check |
| `POST` | `/v1/sagas/render` | Start the Render Saga for a new project |
| `POST` | `/v1/sagas/publish` | Start the Publish Saga (requires project status `ready_to_publish`) |
| `GET` | `/v1/projects/{project_id}` | Read current project state |
| `POST` | `/v1/projects/{project_id}/retry` | Retry the failed step of a project in `failed_at_<step>` status |

See `aidlc-docs/construction/orchestrator-service/low-level-design/interface-contracts.md` for full request/response shapes.

## Testing
```bash
cd services/orchestrator
go mod tidy
go build ./...
go vet ./...
go test ./...
```
Unit tests use hand-written in-memory fakes for the 3 domain ports (`ProjectRepositoryPort`, `CommandPublisherPort`, `ProgressPublisherPort`) — no live Postgres/RabbitMQ required. Repository-layer tests cover pure-Go mapping logic only (JSON (de)serialization, enum string round-trips); integration testing against a real database is out of scope for this stage.
