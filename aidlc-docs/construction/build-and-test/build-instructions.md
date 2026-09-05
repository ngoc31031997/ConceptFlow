# Build Instructions

## Scope note
This pass covers all 10 units, now that Code Generation is complete for every unit: `rabbitmq` (infra config only), `content-plugin`, `script-processing`, `tts`, `rendering`, `video-assembly`, `publisher` (all Python/FastAPI), `orchestrator` (Go), `api-gateway` (Node.js/Express), and `web-gui` (React/TypeScript/Vite).

## Prerequisites
- **Python**: 3.12 (Dockerfiles pin `python:3.12-slim`). The build/test machine did NOT have 3.12 pre-installed (only system Python 3.9.6); it was installed via `brew install python@3.12`. Python 3.9 fails to import this codebase (`str | None` PEP 604 syntax needs 3.10+, `datetime.UTC` needs 3.11+), so 3.12 is a hard requirement, not just a recommendation.
- **Go**: 1.22+ (orchestrator's `go.mod`/Dockerfile pin `golang:1.22-alpine`). Installed via `brew install go`, binary at `/opt/homebrew/bin/go`.
- **Node.js**: 20+ (api-gateway and web-gui Dockerfiles pin `node:20-alpine`). Verified locally on Node 20.20.2 / npm 10.8.2.
- **Docker / Docker Compose**: Docker Desktop with `docker compose` v2 plugin (verified: Docker 28.0.1, Compose v2.33.1).
- **System libraries for `rendering` service** (Manim dependency `pycairo`): `pkg-config`, `cairo`, `pango` must be present, plus `ffmpeg` for video encoding. On macOS: `brew install cairo pango pkg-config ffmpeg`. Without these, `pip install -r requirements-dev.txt` for `rendering` fails at pycairo's meson build step.
- **Environment Variables**: `docker-compose.yml` reads `RABBITMQ_USER`, `RABBITMQ_PASS`, `POSTGRES_USER`, `POSTGRES_PASS`, `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URI` from a `.env` file. A `.env.example` exists at the repo root; copy it to `.env` and fill in real values before `docker compose up`. Without `.env`, `docker compose config`/`build` still succeed but log "variable not set" warnings and default to blank strings.
- **System Requirements**: standard laptop/dev machine is sufficient at this project's stated single-user local scale (see NFR requirements — no load/perf targets).

## Build Steps

### 1. Install Dependencies (per Python service)
Each of `content-plugin`, `script-processing`, `tts`, `rendering`, `video-assembly`, `publisher` has its own `requirements.txt` (runtime) and `requirements-dev.txt` (runtime + pytest/ruff/httpx, via `-r requirements.txt`).

```bash
cd services/<service-name>
python3.12 -m venv .venv
source .venv/bin/activate
pip install --upgrade pip
pip install -r requirements-dev.txt
```

### 2. Build Go Service (orchestrator)
```bash
cd services/orchestrator
go build ./...
go vet ./...
```

### 3. Install Dependencies (Node.js services: api-gateway, web-gui)
```bash
cd services/<api-gateway|web-gui>
npm install
npx eslint .
npm run build   # web-gui only — api-gateway has no build step (plain Node, no bundler)
```

### 4. Configure Environment
```bash
cp .env.example .env
# edit .env with real RabbitMQ/Postgres credentials and, for full stack up,
# YouTube/Google OAuth client id/secret/redirect URI (needed by publisher)
```

### 5. Build All Units (Docker images)
```bash
docker compose config   # validates docker-compose.yml syntax and full service graph
docker compose build    # builds all 9 buildable service Dockerfiles (rabbitmq uses stock image, no build)
```

## Verify Build Success
- **Expected Output**: `docker compose config` prints the fully resolved compose model (9 app services + rabbitmq + 7 postgres sidecar containers, 1 network, 9 named volumes) with exit code 0. `docker compose build` reports `Built` for each buildable service.
- **Build Artifacts**: Docker images `conceptflow-content-plugin`, `conceptflow-script-processing`, `conceptflow-tts`, `conceptflow-rendering`, `conceptflow-video-assembly`, `conceptflow-publisher`, `conceptflow-orchestrator`, `conceptflow-api-gateway`, `conceptflow-web-gui` in the local Docker image store. `orchestrator` binary at `services/orchestrator/orchestrator` when built with plain `go build`.
- **Observed in this pass**: `docker compose config` succeeded (exit 0, only benign "variable not set" warnings when `.env` absent) and confirms all 9 buildable services' Dockerfile paths/contexts resolve correctly across all 16 containers in the full topology. `docker compose build` for all 9 services was NOT run in this pass — Docker Desktop was not running on the verification machine — but each service's native build/test tooling was run directly and passed (see unit-test-instructions.md): `pytest`/`go build`/`npx tsc -b`/`npx vite build`/`npm test` all green for every unit. A prior pass (2026-08-31) built and verified 2 representative Docker images (`content-plugin`, `orchestrator`) directly; Dockerfiles for `api-gateway` and `web-gui` follow the same reviewed multi-stage pattern (approved at each unit's Infrastructure Design stage) but have not yet had an actual `docker build` run against them.
- **Common Warnings**: `pytest-asyncio`'s `PytestDeprecationWarning` about `asyncio_default_fixture_loop_scope` being unset — cosmetic, does not affect test results, safe to ignore or fix later by setting `asyncio_mode`/`asyncio_default_fixture_loop_scope` in each service's `pytest.ini`/`pyproject.toml`.

## Troubleshooting

### Build Fails with Dependency Errors (pycairo / manim, `rendering` service only)
- **Cause**: `pycairo` requires `pkg-config` and the Cairo/Pango C libraries at build time; without them `meson` fails with `Run-time dependency cairo found: NO`.
- **Solution**: `brew install cairo pango pkg-config ffmpeg` (macOS) or the Linux distro equivalents (`apt-get install pkg-config libcairo2-dev libpango1.0-dev ffmpeg`), then retry `pip install -r requirements-dev.txt`.

### Import Errors on Python < 3.12
- **Cause**: codebase uses PEP 604 union types (`str | None`) and `datetime.UTC`, both unavailable before Python 3.10/3.11 respectively.
- **Solution**: ensure the venv is built with `python3.12`, not the system `python3`.

### `docker compose build` Fails to Reach Docker Daemon
- **Cause**: Docker Desktop not running.
- **Solution**: start Docker Desktop (`open -a Docker` on macOS) and wait for `docker info` to succeed before retrying.
