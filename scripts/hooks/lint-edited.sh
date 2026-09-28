#!/usr/bin/env bash
# PostToolUse hook for Edit/Write/MultiEdit: lints only the file just edited,
# with the same tool `make check` uses for its service. No tests: those run
# once at the end of the turn (stop-check.sh).
#
# Exit 0: clean or not a lintable file. Exit 2: the findings go back to the agent.
set -uo pipefail

file="$(jq -r '.tool_input.file_path // empty')"
[ -n "$file" ] && [ -f "$file" ] || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
case "$file" in
  "$ROOT"/services/*) ;;
  *) exit 0 ;;
esac
rel="${file#"$ROOT"/services/}"
svc="${rel%%/*}"
dir="$ROOT/services/$svc"
path_in_svc="${rel#*/}"

report() {
  echo "Lint findings in ${file#"$ROOT"/} ($1):" >&2
  echo "$2" >&2
  exit 2
}

case "$file" in
  *.go)
    out="$(gofmt -l "$file" 2>&1)"
    [ -z "$out" ] || report gofmt "not gofmt-formatted; run: gofmt -w ${file#"$ROOT"/}"
    ;;
  *.py)
    [ -f "$dir/pyproject.toml" ] || exit 0
    py="$dir/.venv/bin/python"
    [ -x "$py" ] || report ruff "services/$svc has no .venv — run 'make setup SERVICES=$svc'"
    out="$(cd "$dir" && "$py" -m ruff check --quiet "$path_in_svc" 2>&1)" || report ruff "$out"
    ;;
  *.ts|*.tsx|*.js|*.jsx|*.mjs|*.cjs)
    [ -f "$dir/package.json" ] || exit 0
    [ -d "$dir/node_modules" ] || report eslint "services/$svc has no node_modules — run 'make setup SERVICES=$svc'"
    out="$(cd "$dir" && npx --no-install eslint "$path_in_svc" 2>&1)" || report eslint "$out"
    ;;
esac
exit 0
