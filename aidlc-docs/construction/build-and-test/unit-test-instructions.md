# Unit Test Execution

## Run Unit Tests

### Python services (content-plugin, script-processing, tts, rendering, video-assembly, publisher)
```bash
cd services/<service-name>
source .venv/bin/activate   # venv created per build-instructions.md, Python 3.12
python -m pytest tests/ -q
```

### Go service (orchestrator)
```bash
cd services/orchestrator
go test ./...
```

### Node.js services (api-gateway, web-gui)
```bash
cd services/<service-name>
npm install
npx eslint .
npm test
```

### rabbitmq unit
Infrastructure-only unit (no application code) — no unit tests apply. Its `infra/rabbitmq/rabbitmq.conf` and `infra/rabbitmq/definitions.json` were reviewed manually: `rabbitmq.conf` points `management.load_definitions` at `definitions.json`, both are correctly mounted read-only into the container per `docker-compose.yml`, and `docker compose config` confirms the mount paths resolve. No further "tests" exist for this unit at this stage.

## Actual Results Observed

| Service | Command | Result | Date |
|---|---|---|---|
| content-plugin | `pytest tests/ -q` | **24 passed**, 0 failed | 2026-08-31 |
| script-processing | `pytest tests/ -q` | **28 passed**, 0 failed | 2026-08-31 |
| tts | `pytest tests/ -q` | **22 passed**, 0 failed | 2026-08-31 |
| rendering | `pytest tests/ -q` | **30 passed**, 0 failed | 2026-08-31 |
| video-assembly | `pytest tests/ -q` | **20 passed**, 0 failed | 2026-08-31 |
| publisher | `pytest tests/ -q` | **28 passed**, 0 failed | 2026-08-31 |
| orchestrator | `go build ./... && go vet ./... && go test ./...` | build OK, vet OK, **all packages `ok`** (adapters/amqp, adapters/http, adapters/postgres, application); `cmd/orchestrator`, `adapters/logging`, `config`, `domain` reported `[no test files]` | 2026-08-31 |
| api-gateway | `npx eslint . && npm test` (Jest) | lint clean, **28 passed**, 0 failed (10 suites) | 2026-09-05 |
| web-gui | `npx eslint . && npm test` (Vitest) | lint clean (0 errors, 2 acceptable Fast-Refresh warnings), **18 passed**, 0 failed (9 files) | 2026-09-05 |
| rabbitmq | N/A (infra only) | config reviewed manually, sound | 2026-08-04 |

**Total**: 152 Python tests + 28 Node (api-gateway) tests + 18 Node (web-gui) tests = **198 tests passed, 0 failed**, across 8 application units. All Go packages with tests pass. No test was skipped, xfailed, or errored in the final run for any unit.

- **Test Coverage**: No coverage tool (`pytest-cov`/`--coverage`) is configured in any service, so no coverage percentage was measured in this pass. Recommend adding coverage tooling in a follow-up if code-coverage gating is desired for CI.
- **Test Report Location**: Console output only in this pass (no `--junitxml`/HTML report generated). For CI, wire per-service test reports (see ci-cd-integration-instructions.md).

### 3. Fix Failing Tests
No test failures were observed across any of the 8 application units (10 units total including rabbitmq infra-only and the docker-compose topology check). If a future run does surface failures:
1. Review the pytest/`go test -v`/`jest`/`vitest` output directly in the terminal.
2. Identify the failing test file/case.
3. Fix the underlying code or test per whether the test or the implementation is wrong (do not blindly change tests to pass).
4. Rerun the relevant command until green.
