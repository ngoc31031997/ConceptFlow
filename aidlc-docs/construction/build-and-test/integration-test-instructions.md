# Integration Test Instructions

## Status: NOT executed in this pass
This pass validated `docker compose config` (syntax/topology) and built 2 of 8 service images, but did **not** run `docker compose up` for the full stack and did **not** execute any cross-service integration test. Reasons:
1. Full stack startup requires real secrets (`GOOGLE_OAUTH_CLIENT_ID`/`SECRET`/`REDIRECT_URI` for the `publisher` service's YouTube upload flow) that are not available in this environment.
2. Starting the full stack would leave 15 containers (8 app services + rabbitmq + 6 postgres sidecars — content-plugin, script-processing, tts, rendering, video-assembly, publisher each have their own DB) running, which the Build and Test task scope explicitly asked to avoid.
3. API Gateway and Web GUI — which normally originate the sagas below via their REST/SSE calls into the orchestrator — are not yet built, so an end-to-end trigger path doesn't fully exist yet.

The scenarios below are written from the Saga message flow documented in `aidlc-docs/construction/orchestrator-service/low-level-design/sequence-flows.md`, as **instructions for a future integration-test run**, not results of an actual run.

## Prerequisites (for when this is actually run)
- `.env` populated with real `RABBITMQ_USER`/`PASS`, `POSTGRES_USER`/`PASS`, and Google OAuth credentials (`GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URI`) — a real Google Cloud project with the YouTube Data API enabled and an OAuth consent screen configured is needed to actually exercise the publisher's upload step.
- Docker Desktop running with enough resources for 15 containers (8 services + rabbitmq + 6 Postgres instances).
- Since API Gateway doesn't exist yet, sagas must be triggered directly against `orchestrator`'s REST API (e.g. `POST /v1/sagas/render`) with `curl`/Postman rather than through a UI.

## Test Scenarios (derived from sequence-flows.md)

### Scenario 1: Start Render Saga → Parse Script (Flow 1)
- **Description**: Orchestrator accepts a new render saga, persists a `draft` Project + `parse_script` SagaStep, enqueues a `parse_script` command via the transactional outbox, and script-processing picks it up off RabbitMQ.
- **Setup**: Full stack up (`docker compose up -d`); orchestrator and script-processing healthy.
- **Test Steps**:
  1. `POST /v1/sagas/render` on orchestrator with a script payload.
  2. Assert `201 {saga_id, status: started}`.
  3. Poll orchestrator's Postgres (`saga_steps` table) or RabbitMQ management UI to confirm `parse_script` command was published within ~500ms (outbox poll interval).
  4. Confirm script-processing consumed the message (check its logs / inbox table for `message_id`).
- **Expected Results**: Project status transitions to `parsing_script`; script-processing receives exactly one `parse_script` command.
- **Cleanup**: `docker compose down -v` to drop volumes/state between test runs.

### Scenario 2: Classify Scenes → Synthesize Speech (Flow 2, content-plugin → tts)
- **Description**: content-plugin classifies scenes and emits `scenes_classified`; orchestrator advances the saga and enqueues `synthesize_speech` for tts.
- **Setup**: Continue from Scenario 1's saga (or seed a Project already at `classify_scenes` step).
- **Test Steps**: Publish/await `scenes_classified` event, verify orchestrator's inbox dedupes on `message_id`, verify Project status becomes `synthesizing_speech`, verify tts receives `synthesize_speech`.
- **Expected Results**: Exactly-once processing (inbox dedup), correct state transition, progress event published to the `PROG` publisher (would be consumed by API Gateway's SSE fanout, not yet built).
- **Cleanup**: as above.

### Scenario 3: Render Failure → Retry (Flows 4 & 5, rendering → orchestrator)
- **Description**: rendering emits `rendering_failed`; orchestrator marks the step failed WITHOUT rolling back already-rendered artifacts; a subsequent `POST /v1/projects/{id}/retry` re-enqueues `render_scenes` with a new `message_id`, and rendering's own artifact-level idempotency skips scenes already rendered successfully.
- **Test Steps**: Force a rendering failure (e.g. malformed scene payload), confirm Project status `failed_at_render_scenes`, call the retry endpoint, confirm only the failed scene(s) are re-rendered (not all scenes).
- **Expected Results**: No duplicate artifacts for already-succeeded scenes; saga eventually reaches `rendering` → later steps.

### Scenario 4: Out-of-Order / Duplicate Event Safeguard (Flow 6)
- **Description**: A `scenes_classified` event redelivered (or arriving late) for a step already `completed` is logged as unexpected and skipped rather than corrupting state.
- **Test Steps**: Manually republish a `scenes_classified` message with a `saga_id` whose step is already `completed`; confirm orchestrator logs a warning and does not re-advance the state machine; confirm the message is still acked.

### Scenario 5: Full Saga → Publish (Flow 7, video-assembly → publisher → YouTube)
- **Description**: End-to-end saga from render through publish, exercising every service and requiring real YouTube OAuth credentials.
- **Test Steps**: Run a complete render saga to completion, then `POST` a publish saga; confirm publisher calls the YouTube Data API and the Project reaches a terminal `published` status.
- **Expected Results**: Video appears on the configured YouTube channel (private/unlisted recommended for test runs).
- **Note**: This is the only scenario that truly requires external network access and real secrets; the others can run against a fully local Docker Compose stack with RabbitMQ/Postgres only.

## Setup Integration Test Environment
```bash
cp .env.example .env   # fill in real credentials first
docker compose up -d
docker compose ps      # confirm all services report healthy
```

## Run Integration Tests
No automated integration test suite/runner exists yet in the repo (no `tests/integration/` directory at the repo root or per-service). A future pass should add one, e.g. a Python `pytest` suite using `httpx`/`pika`/`aio-pika` against the running compose stack, or a `docker compose -f docker-compose.yml -f docker-compose.test.yml run` harness.

### Verify Service Interactions
- **Test Scenarios**: the 5 above.
- **Expected Results**: see per-scenario notes.
- **Logs Location**: `docker compose logs <service>` per service; RabbitMQ management UI at `http://localhost:15672` (credentials from `.env`) for message flow visibility.

### Cleanup
```bash
docker compose down -v
```
