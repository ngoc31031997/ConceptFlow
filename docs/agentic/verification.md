# Verification — `make setup`, `make build`, `make check`, `make check-all`

One set of commands for developers, Claude Code (hooks, skills) and CI.
The logic lives in `scripts/setup.sh`, `scripts/build.sh` and `scripts/check.sh`; the `Makefile` only forwards to them.

## Commands

| Command | What it runs | When |
|---|---|---|
| `make setup` | Creates/refreshes each Python service's `.venv` from `requirements-dev.txt` (interpreter `$PYTHON`, default `python3.12`), `npm ci` for Node services, `go mod download` for Go services. `SERVICES="tts web-gui"` limits it. | New clone, new worktree, CI, after a dependency change |
| `make build` | Compiles every service: `go build ./...`; `npm run build` where defined (web-gui: `tsc` + `vite build`), otherwise `node --check` on `src/` (api-gateway); `compileall` for Python. | CI; before a Docker rebuild |
| `make check` | Lint + unit tests for the services **changed vs `main`** (`BASE=<ref>` to override). | Before declaring a task done; Stop hook; before merging to `main` |
| `make check-all` | Lint + unit tests for **every** service, plus the contract tests. | CI; after pulling a large change |

Exit code: `0` when every step passed, `1` when any step failed, `2` for a usage error (for example `BASE` has no merge-base with `HEAD`).

## What counts as "changed" (`make check`)

Files that differ from the merge-base of `HEAD` and `$BASE`: committed on the branch, staged, unstaged, and untracked (ignored files excluded).

- A file under `services/<svc>/` selects that service.
- A file under `tests/contracts/` or `docs/contracts/` selects the contract tests.
- A change to `Makefile` or `scripts/{setup,build,check}.sh` selects everything: a change to the verification re-verifies all of it.
- Nothing selected → `check: nothing to verify`, exit `0`.

## Steps per service

A service is a directory under `services/` with one of these at its root. `services/content-plugin/` has none (it is a leftover directory with no tracked files) and is skipped.

| Kind (detected by) | Lint | Unit tests |
|---|---|---|
| Go (`go.mod`) | `gofmt -l` must be empty, `go vet ./...` | `go test ./...` |
| Python (`pyproject.toml`) | `.venv/bin/python -m ruff check .` | `.venv/bin/python -m pytest -q` |
| Node (`package.json`) | `npm run lint`, plus `tsc -b` when `tsconfig.json` exists | `npm test` |

Contract tests (`tests/contracts/`) run with pytest from the first Python service's `.venv`, since they need only the stdlib and pytest.

A missing `.venv` or `node_modules` fails the step with `run 'make setup' first`. The script never falls back to the system interpreter: macOS ships Python 3.9 and the services need 3.12.

## Output

In GitHub Actions, each failing step is also emitted as an `::error` annotation, shown on the run summary and readable through the public check-runs API.


Every step prints one line (`ok`/`FAIL` and its duration). Every step runs even after a failure. At the end, the last 25 lines of each failing step are printed. Full logs are in `$CHECK_LOG_DIR` (default `$TMPDIR/conceptflow-check/`).

## Not covered (by design)

- SonarQube and OWASP Dependency-Check are in the backlog (Creator decision, 2026-09-28).

- Integration tests that need Docker, Postgres or RabbitMQ, and anything calling a real LLM or TTS API. `make check` stays fast (≈40 s for the whole repo on a dev Mac) because the Stop hook runs it.
- Building Docker images. The Docker rebuild policy in `CLAUDE.md` still applies separately.
