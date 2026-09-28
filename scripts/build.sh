#!/usr/bin/env bash
# Compiles every service, called by `make build` and CI (.github/workflows/ci.yml).
#
#   Go      go build ./...
#   Python  compileall over the service's source (needs `make setup` for .venv)
#   Node    `npm run build` when the service defines one (web-gui: tsc + vite),
#           otherwise `node --check` on every file under src/ (api-gateway)
#
# Every service is attempted even after a failure; exit code is non-zero if any failed.
# Written for bash 3.2 (the macOS default).
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

FAILED=""

step() {
  label="$1"; shift
  printf '  %-32s ' "$label"
  start=$(date +%s)
  if out="$("$@" 2>&1)"; then
    echo "ok ($(( $(date +%s) - start ))s)"
  else
    echo "FAIL ($(( $(date +%s) - start ))s)"
    echo "$out" | tail -n 30 | sed 's/^/      /'
    if [ -n "${GITHUB_ACTIONS:-}" ]; then
      echo "::error title=build $label::$(echo "$out" | tail -n 15 | sed ':a;N;$!ba;s/%/%25/g;s/\n/%0A/g')"
    fi
    FAILED="$FAILED $label"
  fi
}

for dir in services/*/; do
  svc="$(basename "$dir")"
  if [ -f "$dir/go.mod" ]; then
    step "$svc: go build" bash -c "cd '$dir' && go build ./..."
  elif [ -f "$dir/pyproject.toml" ]; then
    py="$dir/.venv/bin/python"
    if [ ! -x "$py" ]; then
      step "$svc: environment" bash -c "echo '$svc has no .venv — run make setup first'; exit 1"
      continue
    fi
    step "$svc: compileall" "$py" -m compileall -q \
      -x '/(\.venv|node_modules|remotion_project|__pycache__)/' "$dir"
  elif [ -f "$dir/package.json" ]; then
    if [ ! -d "$dir/node_modules" ]; then
      step "$svc: environment" bash -c "echo '$svc has no node_modules — run make setup first'; exit 1"
      continue
    fi
    if grep -q '"build"' "$dir/package.json"; then
      step "$svc: npm run build" bash -c "cd '$dir' && npm run --silent build"
    else
      step "$svc: node --check" bash -c "cd '$dir' && find src \\( -name '*.js' -o -name '*.mjs' -o -name '*.cjs' \\) -print0 | xargs -0 -n1 node --check"
    fi
  fi
done

if [ -n "$FAILED" ]; then
  echo "build failed:$FAILED"
  exit 1
fi
echo "build ok"
