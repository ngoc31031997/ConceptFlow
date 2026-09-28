#!/usr/bin/env bash
# Rebuilds and restarts Docker Compose services, then waits until each is healthy.
# Implements the Docker rebuild policy in CLAUDE.md; used by the /rebuild skill.
#
#   scripts/rebuild.sh               services whose code changed vs $BASE (default: main)
#   scripts/rebuild.sh tts web-gui   exactly these services
#
# Only services built from ./services/<name> are candidates, and the compose
# service name must equal the directory name (true for every service today).
# Exit 0 when every rebuilt service is healthy (or running, if it has no
# healthcheck); 1 when a build/start fails or a service does not become healthy
# within $HEALTH_TIMEOUT seconds (default 180); 2 for a usage error.
#
# Written for bash 3.2 (the macOS default).
set -uo pipefail

BASE="${BASE:-main}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-180}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

compose_services="$(docker compose config --services)" || {
  echo "rebuild: 'docker compose config --services' failed (is Docker running?)" >&2
  exit 2
}

buildable() {
  [ -d "services/$1" ] && grep -qx "$1" <<<"$compose_services"
}

if [ "$#" -gt 0 ]; then
  SERVICES="$*"
  for svc in $SERVICES; do
    buildable "$svc" || { echo "rebuild: '$svc' is not a compose service built from services/$svc" >&2; exit 2; }
  done
else
  base_commit="$(git merge-base "$BASE" HEAD 2>/dev/null)" || {
    echo "rebuild: cannot find merge-base with '$BASE'" >&2
    exit 2
  }
  files="$( { git diff --name-only "$base_commit"; git ls-files --others --exclude-standard; } | sort -u)"
  SERVICES=""
  for dir in services/*/; do
    svc="$(basename "$dir")"
    grep -q "^services/$svc/" <<<"$files" && buildable "$svc" && SERVICES="$SERVICES $svc"
  done
  if [ -z "$(echo "$SERVICES" | tr -d ' ')" ]; then
    echo "rebuild: no service code changed vs $BASE; nothing to rebuild"
    exit 0
  fi
fi

echo "rebuild:$SERVICES"
docker compose build $SERVICES || { echo "rebuild: build FAILED" >&2; exit 1; }
docker compose up -d $SERVICES || { echo "rebuild: start FAILED" >&2; exit 1; }

# status_of <service>: healthy | unhealthy | starting | running (no healthcheck) | exited | missing
status_of() {
  cid="$(docker compose ps -q "$1" 2>/dev/null | head -n 1)"
  [ -n "$cid" ] || { echo missing; return; }
  docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$cid"
}

failed=0
for svc in $SERVICES; do
  start=$(date +%s)
  while :; do
    status="$(status_of "$svc")"
    case "$status" in
      healthy|running) echo "  $svc: $status ($(( $(date +%s) - start ))s)"; break ;;
      unhealthy|exited|dead|missing)
        echo "  $svc: $status — last log lines:" >&2
        docker compose logs --tail 40 "$svc" >&2
        failed=1; break ;;
    esac
    if [ $(( $(date +%s) - start )) -ge "$HEALTH_TIMEOUT" ]; then
      echo "  $svc: still '$status' after ${HEALTH_TIMEOUT}s — last log lines:" >&2
      docker compose logs --tail 40 "$svc" >&2
      failed=1; break
    fi
    sleep 3
  done
done
exit "$failed"
