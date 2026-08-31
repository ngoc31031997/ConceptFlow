# API Gateway (Unit 9)

Reverse-proxy and AMQP-to-SSE bridge fronting the ConceptFlow backend services for the Web GUI. Stateless — no database, no business logic. Routes REST requests verbatim to the Content Plugin, Orchestrator, and Publisher services, and bridges RabbitMQ `progress.fanout` messages to Server-Sent Events for real-time progress tracking.

## Overview
- **Language/Stack**: Node.js 20, Express 4.x — see `aidlc-docs/decisions/ADR-0020-api-gateway-node-express-stack.md`.
- **Architecture**: Simple layered (`routes/` → `handlers/` → `clients/`), not Hexagonal — the Gateway has no domain logic to isolate. See `aidlc-docs/construction/api-gateway/low-level-design/module-structure.md`.
- **Persistence**: none — fully stateless except an in-memory SSE connection registry (`Map<projectId, Response[]>`), which is lost (and clients simply reconnect) on restart.
- **Messaging**: consumes `progress.fanout` only (exclusive queue, at-most-once delivery); publishes nothing.

## Prerequisites
- Node.js 20+
- A reachable RabbitMQ instance and the 3 downstream services (Orchestrator, Content Plugin, Publisher), or run everything via the root `docker-compose.yml`.

## Running

### `npm start` (local, against already-running dependencies)
```bash
cd services/api-gateway
npm install
export RABBITMQ_URL=amqp://guest:guest@localhost:5672/
export ORCHESTRATOR_URL=http://localhost:8001
export CONTENT_PLUGIN_URL=http://localhost:8002
export PUBLISHER_URL=http://localhost:8003
npm start
```
The service listens on port 8080 (`GET /health` for the compose healthcheck).

### Docker (via root docker-compose)
```bash
docker compose up api-gateway
```
This is the only application service (besides RabbitMQ's dev-only management UI) whose port is published to the host — it's the system's single entry point.

## Environment Variables
| Variable | Required | Default | Description |
|---|---|---|---|
| `RABBITMQ_URL` | yes | — | AMQP connection string |
| `ORCHESTRATOR_URL` | yes | — | Base URL of the Orchestrator Service |
| `CONTENT_PLUGIN_URL` | yes | — | Base URL of the Content Plugin Service |
| `PUBLISHER_URL` | yes | — | Base URL of the Publisher Service |
| `PORT` | no | `8080` | HTTP listen port |

## REST / SSE API
| Method | Path | Proxied to |
|---|---|---|
| `GET` | `/v1/plugins` | Content Plugin Service |
| `POST` | `/v1/sagas/render` | Orchestrator Service |
| `POST` | `/v1/sagas/publish` | Orchestrator Service |
| `GET` | `/v1/projects/{project_id}` | Orchestrator Service |
| `POST` | `/v1/projects/{project_id}/retry` | Orchestrator Service |
| `GET` | `/v1/auth/youtube/start` | Publisher Service (forwards `302` verbatim) |
| `GET` | `/v1/auth/youtube/callback` | Publisher Service |
| `GET` | `/v1/progress/{project_id}` | Gateway itself (SSE, AMQP-to-SSE bridge — not proxied) |
| `GET` | `/health` | Gateway itself — always `200 {status: "ok"}`, does not check downstream health |

All passthrough endpoints forward request/response verbatim (method, headers including `X-Request-ID`, body, query) — the Gateway does not validate or transform payloads; that is downstream services' responsibility. See `aidlc-docs/construction/api-gateway/low-level-design/interface-contracts.md` for the full contract.

## Testing
```bash
cd services/api-gateway
npm install
npm test
```
Unit tests use hand-written fakes/mocks — `fetch` mocked for `httpClient`, `amqplib` mocked for `amqpClient`, fake client objects for handlers, and `supertest` against minimal Express app instances for routes. No live RabbitMQ or downstream services are required.
