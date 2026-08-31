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

### rabbitmq unit
Infrastructure-only unit (no application code) — no unit tests apply. Its `infra/rabbitmq/rabbitmq.conf` and `infra/rabbitmq/definitions.json` were reviewed manually: `rabbitmq.conf` points `management.load_definitions` at `definitions.json`, both are correctly mounted read-only into the container per `docker-compose.yml`, and `docker compose config` confirms the mount paths resolve. No further "tests" exist for this unit at this stage.

## Actual Results Observed (2026-08-31)

| Service | Command | Result |
|---|---|---|
| content-plugin | `pytest tests/ -q` | **24 passed**, 0 failed |
| script-processing | `pytest tests/ -q` | **28 passed**, 0 failed |
| tts | `pytest tests/ -q` | **22 passed**, 0 failed |
| rendering | `pytest tests/ -q` | **30 passed**, 0 failed |
| video-assembly | `pytest tests/ -q` | **20 passed**, 0 failed |
| publisher | `pytest tests/ -q` | **28 passed**, 0 failed |
| orchestrator | `go build ./... && go vet ./... && go test ./...` | build OK, vet OK, **all packages `ok`** (adapters/amqp, adapters/http, adapters/postgres, application); `cmd/orchestrator`, `adapters/logging`, `config`, `domain` reported `[no test files]` |
| rabbitmq | N/A (infra only) | config reviewed manually, sound |

**Total Python unit tests**: 152 passed, 0 failed, across 6 services. No test was skipped, xfailed, or errored in the final run (an initial run under Python 3.9 failed to even collect tests due to language-version incompatibility — resolved by installing Python 3.12, see build-instructions.md; this was an environment issue, not a code issue).

- **Test Coverage**: No coverage tool (`pytest-cov`) is configured in any service's `requirements-dev.txt`, so no coverage percentage was measured in this pass. Recommend adding `pytest-cov` and a coverage threshold in a follow-up if code-coverage gating is desired for CI.
- **Test Report Location**: Console output only in this pass (no `--junitxml`/HTML report generated). For CI, wire `pytest --junitxml=reports/<service>-junit.xml` per service (see ci-cd-integration-instructions.md).

### 3. Fix Failing Tests
No test failures were observed in this pass (all 152 Python tests + all Go tests passed once the correct toolchain — Python 3.12 — was used). If a future run does surface failures:
1. Review the pytest/`go test -v` output directly in the terminal.
2. Identify the failing test file/case.
3. Fix the underlying code or test per whether the test or the implementation is wrong (do not blindly change tests to pass).
4. Rerun `pytest tests/ -q` (or `go test ./...`) until green.
