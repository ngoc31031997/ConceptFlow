# Integration Test Instructions

## Status: NOT executed in this pass
> **Cập nhật (CR-055, 2026-09-30)**: service hiện có — Python: `tts`, `rendering`, `video-assembly`, `publisher`, `llm-service`; Go: `orchestrator`, `authoring-service`; Node: `api-gateway`, `web-gui`; cộng `rabbitmq`, 6 Postgres (orchestrator, authoring-service, tts, rendering, video-assembly, publisher), `ollama` (+ `ollama-pull` chạy một lần), `loki`/`promtail`/`grafana` — 22 mục trong `docker compose config --services`, 9 image tự build. `content-plugin` (gỡ ở CR-020) và `script-processing` (gỡ ở CR-040) không còn.

This pass (and the original 2026-08-31 pass) validated `docker compose config` (full 16-container topology, now including api-gateway and web-gui) but did **not** run `docker compose up` for the full stack and did **not** execute any cross-service integration test. Reasons:
1. Full stack startup requires real secrets (`GOOGLE_OAUTH_CLIENT_ID`/`SECRET`/`REDIRECT_URI` for the `publisher` service's YouTube upload flow) that are not available in this environment.
2. Starting the full stack would leave the full container set (xem ghi chú trên) running, which the Build and Test task scope explicitly asked to avoid.
3. Docker Desktop was not running on the verification machine during this pass.

All 10 units are now built, so a true end-to-end browser-driven path (Web GUI → API Gateway → Orchestrator) exists for the first time. The scenarios below are written from the Saga message flow documented in `aidlc-docs/construction/orchestrator-service/low-level-design/sequence-flows.md` and the Web GUI's `sequence-flows.md`, as **instructions for a future integration-test run**, not results of an actual run.

## Prerequisites (for when this is actually run)
- `.env` populated with real `RABBITMQ_USER`/`PASS`, `POSTGRES_USER`/`PASS`, and Google OAuth credentials (`GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URI`) — a real Google Cloud project with the YouTube Data API enabled and an OAuth consent screen configured is needed to actually exercise the publisher's upload step.
- Docker Desktop running with enough resources for the full stack (xem ghi chú trên; ollama cần vài GB RAM khi được dùng).
- Sagas can now be triggered either directly against `orchestrator`'s REST API (via API Gateway's `/v1/sagas/render` proxy route, e.g. `curl`/Postman) or end-to-end through the browser at `http://localhost:3000` (Web GUI) — the latter is the more representative real-user path and should be preferred once available.

## Test Scenarios (derived from sequence-flows.md)

### Scenario 1: Start Render Saga → Validate Script (Flow 1, CR-040)
- **Description**: Orchestrator accepts a new render saga, persists the `validate_script` SagaStep, enqueues a `validate_script` command via the transactional outbox, and rendering picks it up off RabbitMQ.
- **Setup**: Full stack up (`docker compose up -d`); orchestrator and rendering healthy.
- **Test Steps**:
  1. `POST /v1/sagas/render` on orchestrator with a script payload.
  2. Assert `201 {saga_id, status: started}`.
  3. Poll orchestrator's Postgres (`saga_steps` table) or RabbitMQ management UI to confirm `validate_script` command was published on `rendering.commands` within ~500ms (outbox poll interval).
  4. Confirm rendering consumed the message (check its logs / inbox table for `message_id`).
- **Expected Results**: rendering receives exactly one `validate_script` command and replies `script_validated` (or `validation_failed` → Project `failed_at_validate_script`).
- **Cleanup**: `docker compose down -v` to drop volumes/state between test runs.

### Scenario 2: Validate Script → Synthesize Speech (Flow 2, rendering → tts)
- **Description**: rendering emits `script_validated`; orchestrator advances the saga and enqueues `synthesize_speech` for tts.
- **Setup**: Continue from Scenario 1's saga.
- **Test Steps**: Await `script_validated` event, verify orchestrator's inbox dedupes on `message_id`, verify the `synthesize_speech` step starts, verify tts receives `synthesize_speech`.
- **Expected Results**: Exactly-once processing (inbox dedup), correct state transition, progress event on `progress.fanout` reaches API Gateway's SSE.
- **Cleanup**: as above.

### Scenario 3: Render Failure → Retry (Flows 4 & 5, rendering → orchestrator)
- **Description**: rendering emits `rendering_failed`; orchestrator marks the step failed WITHOUT rolling back already-rendered artifacts; a subsequent `POST /v1/projects/{id}/retry` re-enqueues `render_scenes` with a new `message_id`, and rendering's own artifact-level idempotency skips scenes already rendered successfully.
- **Test Steps**: Force a rendering failure (e.g. malformed scene payload), confirm Project status `failed_at_render_scenes`, call the retry endpoint, confirm only the failed scene(s) are re-rendered (not all scenes).
- **Expected Results**: No duplicate artifacts for already-succeeded scenes; saga eventually reaches `rendering` → later steps.

### Scenario 4: Out-of-Order / Duplicate Event Safeguard (Flow 6)
- **Description**: A `script_validated` event redelivered (or arriving late) for a step already `completed` is logged as unexpected and skipped rather than corrupting state.
- **Test Steps**: Manually republish a `script_validated` message with a `saga_id` whose step is already `completed`; confirm orchestrator logs a warning and does not re-advance the state machine; confirm the message is still acked.

### Scenario 5: Full Saga → Publish (Flow 7, video-assembly → publisher → YouTube)
- **Description**: End-to-end saga from render through publish, exercising every service and requiring real YouTube OAuth credentials.
- **Test Steps**: Run a complete render saga to completion, then `POST` a publish saga; confirm publisher calls the YouTube Data API and the Project reaches a terminal `published` status.
- **Expected Results**: Video appears on the configured YouTube channel (private/unlisted recommended for test runs).
- **Note**: This is the only scenario that truly requires external network access and real secrets; the others can run against a fully local Docker Compose stack with RabbitMQ/Postgres only.

### Scenario 6: End-to-End via Web GUI (browser-driven, new — Web GUI's sequence-flows.md Flows 1-4)
- **Description**: A Creator opens `http://localhost:3000`, composes a script through the authoring wizard, submits render, watches live progress via SSE, previews the resulting video, connects YouTube, and publishes — with zero direct API calls from the tester.
- **Setup**: Full stack up (`docker compose up -d`), all services healthy, real Google OAuth credentials configured.
- **Test Steps**:
  1. Open `http://localhost:3000`, tạo project theo wizard (chủ đề, soạn kịch bản), duyệt, bấm render.
  2. Confirm navigation to `/projects/:id/render` and that `ProgressTracker` updates live as steps complete (verifies API Gateway's SSE fanout end-to-end, not just at the orchestrator level).
  3. Confirm auto-navigation to `/projects/:id/result` once `ready_to_publish`.
  4. Play the video via `VideoPlayer`, click "Kết nối YouTube", complete the OAuth consent flow, fill in `PublishForm`, submit.
  5. Confirm the page shows the final `youtube_video_url`.
- **Expected Results**: The full user journey (Epics A-E in `stories.md`) completes without needing curl/Postman — this is the first scenario that validates the **entire system as a real user would experience it**, not just service-to-service messaging.
- **Note**: This scenario was not possible before Unit 10 (Web GUI) completed Code Generation; it is the primary reason to prioritize this pass before proceeding to Operations.

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
