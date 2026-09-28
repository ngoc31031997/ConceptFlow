#!/usr/bin/env bash
# Canonical verification entry point, called by `make check` / `make check-all`.
#
#   scripts/check.sh changed   lint + unit tests for services changed vs $BASE (default: main)
#   scripts/check.sh all       lint + unit tests for every service, plus contract tests
#
# A service's kind comes from the file at its root: go.mod (Go), pyproject.toml
# (Python), package.json (Node). A directory with none of these is not a service.
# Every step runs even after a failure so one run reports everything; the exit
# code is non-zero if any step failed. Full logs go to $CHECK_LOG_DIR.
#
# Written for bash 3.2 (the macOS default): no associative arrays, no mapfile.
set -uo pipefail

MODE="${1:-changed}"
BASE="${BASE:-main}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOG_DIR="${CHECK_LOG_DIR:-${TMPDIR:-/tmp}/conceptflow-check}"
mkdir -p "$LOG_DIR"
cd "$ROOT"

service_kind() {
  if [ -f "services/$1/go.mod" ]; then echo go
  elif [ -f "services/$1/pyproject.toml" ]; then echo python
  elif [ -f "services/$1/package.json" ]; then echo node
  fi
}

all_services() {
  for dir in services/*/; do
    name="$(basename "$dir")"
    [ -n "$(service_kind "$name")" ] && echo "$name"
  done
}

# Files differing from the merge-base with $BASE: committed on this branch,
# staged, unstaged, and untracked.
changed_files() {
  base_commit="$(git merge-base "$BASE" HEAD 2>/dev/null)" || {
    echo "check: cannot find merge-base with '$BASE'" >&2
    return 1
  }
  {
    git diff --name-only "$base_commit"
    git ls-files --others --exclude-standard
  } | sort -u
}

FAILED=""
PASSED=0

# run_step <label> <dir> <command...>
run_step() {
  label="$1"; dir="$2"; shift 2
  log="$LOG_DIR/$(echo "$label" | tr ' /:' '___').log"
  printf '  %-40s ' "$label"
  start=$(date +%s)
  if (cd "$dir" && "$@") >"$log" 2>&1; then
    echo "ok ($(( $(date +%s) - start ))s)"
    PASSED=$((PASSED + 1))
  else
    echo "FAIL ($(( $(date +%s) - start ))s)"
    FAILED="$FAILED
$label|$log"
  fi
}

gofmt_clean() {
  unformatted="$(gofmt -l .)"
  [ -z "$unformatted" ] || { echo "gofmt needed:"; echo "$unformatted"; return 1; }
}

python_for() {
  echo "$ROOT/services/$1/.venv/bin/python"
}

# Fails the step with a clear pointer instead of falling back to whatever
# interpreter or tooling happens to be on PATH.
missing_env() {
  echo "$1 has no $2 — run 'make setup' first"
  return 1
}

check_service() {
  svc="$1"; dir="services/$svc"
  case "$(service_kind "$svc")" in
    go)
      run_step "$svc: gofmt" "$dir" gofmt_clean
      run_step "$svc: go vet" "$dir" go vet ./...
      run_step "$svc: go test" "$dir" go test ./...
      ;;
    python)
      py="$(python_for "$svc")"
      if [ ! -x "$py" ]; then
        run_step "$svc: environment" "$dir" missing_env "$svc" ".venv"
        return
      fi
      run_step "$svc: ruff" "$dir" "$py" -m ruff check .
      run_step "$svc: pytest" "$dir" "$py" -m pytest -q -p no:cacheprovider
      ;;
    node)
      if [ ! -d "$dir/node_modules" ]; then
        run_step "$svc: environment" "$dir" missing_env "$svc" "node_modules"
        return
      fi
      run_step "$svc: eslint" "$dir" npm run --silent lint
      if [ -f "$dir/tsconfig.json" ]; then
        run_step "$svc: tsc" "$dir" npx --no-install tsc -b
      fi
      run_step "$svc: test" "$dir" npm test --silent
      ;;
  esac
}

check_contracts() {
  # The contract test only needs the stdlib and pytest; any service venv has both.
  py=""
  for svc in $(all_services); do
    if [ "$(service_kind "$svc")" = python ] && [ -x "$(python_for "$svc")" ]; then
      py="$(python_for "$svc")"; break
    fi
  done
  if [ -z "$py" ]; then
    run_step "contracts: environment" "$ROOT" missing_env "contracts" "service .venv to borrow"
    return
  fi
  run_step "contracts: pytest" "$ROOT" "$py" -m pytest -q -p no:cacheprovider tests/contracts
}

case "$MODE" in
  all)
    SERVICES="$(all_services)"
    RUN_CONTRACTS=1
    ;;
  changed)
    files="$(changed_files)" || exit 2
    SERVICES=""
    RUN_CONTRACTS=0
    for svc in $(all_services); do
      grep -q "^services/$svc/" <<<"$files" && SERVICES="$SERVICES $svc"
    done
    grep -qE '^(tests/contracts/|docs/contracts/)' <<<"$files" && RUN_CONTRACTS=1
    # A change to the verification itself re-verifies everything.
    if grep -qE '^(Makefile|scripts/(check|setup)\.sh)$' <<<"$files"; then
      SERVICES="$(all_services)"
      RUN_CONTRACTS=1
    fi
    ;;
  *)
    echo "usage: $0 [changed|all]" >&2
    exit 2
    ;;
esac

if [ -z "$(echo "$SERVICES" | tr -d ' \n')" ] && [ "$RUN_CONTRACTS" = 0 ]; then
  echo "check: nothing to verify (no service changed vs $BASE)"
  exit 0
fi

echo "check ($MODE): $(echo $SERVICES | tr ' ' ',')$( [ "$RUN_CONTRACTS" = 1 ] && echo ' +contracts')"
for svc in $SERVICES; do check_service "$svc"; done
[ "$RUN_CONTRACTS" = 1 ] && check_contracts

if [ -n "$FAILED" ]; then
  echo
  echo "$PASSED passed, $(echo "$FAILED" | grep -c '|') failed. Last lines of each failing step:"
  echo "$FAILED" | while IFS='|' read -r label log; do
    [ -z "$label" ] && continue
    echo
    echo "── $label  (full log: $log)"
    tail -n 25 "$log"
  done
  exit 1
fi
echo "all $PASSED steps passed"
