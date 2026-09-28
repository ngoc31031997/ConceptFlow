#!/usr/bin/env bash
# Installs the per-service dev tooling that scripts/check.sh runs against:
#   Python  services/<svc>/.venv from requirements-dev.txt (interpreter: $PYTHON, default python3.12)
#   Node    npm ci
#   Go      go mod download
# Idempotent: an existing .venv is reused and only re-synced. Pass service
# names to set up only those (e.g. `make setup SERVICES="tts web-gui"`).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PYTHON="${PYTHON:-python3.12}"
cd "$ROOT"

services="${*:-}"
if [ -z "$services" ]; then
  for dir in services/*/; do services="$services $(basename "$dir")"; done
fi

for svc in $services; do
  dir="services/$svc"
  if [ -f "$dir/go.mod" ]; then
    echo "setup $svc (go)"
    (cd "$dir" && go mod download)
  elif [ -f "$dir/pyproject.toml" ]; then
    echo "setup $svc (python)"
    [ -x "$dir/.venv/bin/python" ] || "$PYTHON" -m venv "$dir/.venv"
    "$dir/.venv/bin/python" -m pip install -q --upgrade pip
    "$dir/.venv/bin/python" -m pip install -q -r "$dir/requirements-dev.txt"
  elif [ -f "$dir/package.json" ]; then
    echo "setup $svc (node)"
    (cd "$dir" && npm ci --no-audit --no-fund)
  fi
done
