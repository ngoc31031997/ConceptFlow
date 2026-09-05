# Observability: Centralized Logging (Grafana + Loki + Promtail)

## Scope
Infra-only unit (no business logic), same pattern as Unit 1 (RabbitMQ Infrastructure) — added directly rather than through a fresh full AI-DLC per-unit loop, per user request during a bug-fix session. Adds centralized log aggregation across all containers so logs from any service can be searched/correlated in one place instead of `docker compose logs <service>` per service.

## Architecture
- **Promtail**: discovers every container via the Docker socket (`docker_sd_configs`), tails their stdout/stderr, and ships log lines to Loki. Filtered via `relabel_configs` to only containers labeled `com.docker.compose.project=conceptflow` — the same Docker host may run unrelated projects' containers, which must not show up here.
- **Loki**: log storage + query backend, single-binary mode with filesystem storage (`loki_data` volume). No host port published — internal only, accessed via Grafana's datasource proxy.
- **Grafana**: UI for querying logs (LogQL) and building dashboards. Loki datasource auto-provisioned (`infra/observability/grafana/provisioning/datasources/loki.yml`) — no manual setup needed after `docker compose up`. Published at `http://localhost:3001` (port 3000 was already taken by `web-gui`). Anonymous access disabled — login with `GRAFANA_USER`/`GRAFANA_PASS` from `.env`.

## Why Promtail reading the Docker socket (not per-service push)
Chosen over having each of the 8 services push logs directly to Loki: zero code changes to any already-built service (Python, Go, Node), works uniformly across all 3 languages, and matches this project's local-single-machine scale — approved by the user over the alternative (each service integrating a Loki HTTP client).

## Files
- `docker-compose.yml`: `loki`, `promtail`, `grafana` services + `loki_data`/`grafana_data` volumes.
- `infra/observability/promtail-config.yml`: scrape + relabel config.
- `infra/observability/grafana/provisioning/datasources/loki.yml`: datasource auto-provisioning.
- `.env.example`: `GRAFANA_USER`/`GRAFANA_PASS`.

## Verified
- `docker compose config` valid with the 3 new services.
- Live log flow confirmed via LogQL query against a real running stack: only `conceptflow`-labeled containers appear in Loki within a live time window; a separate unrelated Docker Compose project running on the same dev machine (`backend-*` containers) does not leak in.
- Note: Loki's `/label/*/values` endpoints returned stale historical values from unrelated projects during verification (a known Loki behavior — those endpoints are not strictly time-scoped against the index) even after the filter was confirmed correctly excluding them at ingestion; a `query_range` with an explicit time window is the reliable way to confirm what is actually flowing.

## Known limitation
No log retention/rotation policy configured (Loki's defaults apply) — acceptable at this project's local/dev scale; revisit if disk usage becomes a concern.
