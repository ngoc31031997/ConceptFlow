#!/usr/bin/env bash
# Stop hook: the agent may only finish a turn when `make check` passes.
#
# Exit 0 lets the agent stop; exit 2 blocks it and stderr (the failing steps)
# goes back to the agent. When the agent is already continuing because of this
# hook (stop_hook_active), it is let through so a failure it cannot fix does
# not loop forever; the failure was already reported once.
#
# A pass is cached per worktree, keyed by HEAD + the diff + untracked files, so
# a turn that changed nothing does not re-run the ~40 s check.
set -uo pipefail

payload="$(cat)"
[ "$(jq -r '.stop_hook_active // false' <<<"$payload")" = true ] && exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT" || exit 0
git rev-parse --git-dir >/dev/null 2>&1 || exit 0

state_key() {
  {
    git rev-parse HEAD
    git diff HEAD --binary
    git ls-files --others --exclude-standard -z | xargs -0 git hash-object -- 2>/dev/null
    git ls-files --others --exclude-standard
  } | git hash-object --stdin
}

cache="$(git rev-parse --path-format=absolute --git-dir)/conceptflow/stop-check-pass"
key="$(state_key)"
[ -f "$cache" ] && [ "$(cat "$cache")" = "$key" ] && exit 0

if out="$("$ROOT/scripts/check.sh" changed 2>&1)"; then
  mkdir -p "$(dirname "$cache")" && echo "$key" >"$cache"
  exit 0
fi

{
  echo "Stop blocked: \`make check\` failed. Fix the failures below (or tell the Creator why they are unrelated) before finishing."
  echo
  echo "$out" | tail -n 150
} >&2
exit 2
