#!/usr/bin/env bash
# Deliver an approved, already committed branch: push it, merge it into main in the
# primary checkout, push main, refresh the graphify code graph for the new main,
# then rebuild and restart only the Docker services whose code changed, wait
# until they report healthy, and remove the branch's worktree if it has one.
#
# Usage: .claude/skills/deliver/deliver.sh <branch>
# Exit codes: 1 bad input/state, 2 push failed, 3 merge conflict (merge aborted),
#             4 docker build/up failed, 5 a service is not healthy,
#             6 everything else succeeded but the graphify refresh failed.
set -euo pipefail

branch="${1:-}"
[ -n "$branch" ] || { echo "usage: $0 <branch>" >&2; exit 1; }
[ "$branch" != "main" ] || { echo "ERROR: refusing to deliver main itself; deliver a feature/fix/chore branch" >&2; exit 1; }
git rev-parse --verify --quiet "refs/heads/$branch" >/dev/null || { echo "ERROR: no local branch $branch" >&2; exit 1; }

# The first worktree is the primary checkout; main and the compose project live there.
main_wt=$(git worktree list --porcelain | sed -n 's/^worktree //p' | head -1)
echo "== primary checkout: $main_wt"

# Once merged, the branch's own worktree (scripts/worktree.sh add) has served its
# purpose. A failed removal (e.g. stray untracked files) is only reported.
remove_worktree() {
  local wt
  wt=$(cd "$main_wt" && scripts/worktree.sh path "$branch" 2>/dev/null) || return 0
  [ "$wt" != "$main_wt" ] || return 0
  (cd "$main_wt" && scripts/worktree.sh remove "$branch") \
    || echo "WARN: worktree $wt not removed; remove it with scripts/worktree.sh remove $branch" >&2
  echo "== continue from the primary checkout: $main_wt"
}

git -C "$main_wt" diff --quiet && git -C "$main_wt" diff --cached --quiet \
  || { echo "ERROR: primary checkout has uncommitted changes" >&2; git -C "$main_wt" status -s >&2; exit 1; }

echo "== push $branch"
git push -u origin "$branch" || { echo "ERROR: push of $branch failed" >&2; exit 2; }

# Working on the branch in the primary checkout is the usual case: switch it to main.
current=$(git -C "$main_wt" symbolic-ref --short HEAD)
if [ "$current" = "$branch" ]; then
  echo "== switch primary checkout to main"
  git -C "$main_wt" switch main
elif [ "$current" != "main" ]; then
  echo "ERROR: primary checkout is on $current, not main or $branch" >&2
  exit 1
fi

echo "== pull main"
git -C "$main_wt" pull --ff-only origin main || { echo "ERROR: main cannot fast-forward to origin/main" >&2; exit 2; }

before=$(git -C "$main_wt" rev-parse HEAD)
echo "== merge $branch into main"
if ! git -C "$main_wt" merge --no-edit "$branch"; then
  git -C "$main_wt" merge --abort || true
  echo "ERROR: merge conflict; merge aborted, main unchanged" >&2
  exit 3
fi

echo "== push main"
git -C "$main_wt" push origin main || { echo "ERROR: push of main failed" >&2; exit 2; }

# Keep graphify-out/ in step with the new main. A failure here does not undo the
# merge or skip the rebuild; it is reported through exit code 6 at the end.
graph_status=0
echo "== refresh graphify graph"
"$main_wt/scripts/graph.sh" build || { graph_status=6; echo "ERROR: graphify refresh failed; run 'make graph' in $main_wt" >&2; }
graph_built=$(sed -n 's/.*"built_at_commit": *"\([0-9a-f]*\)".*/\1/p' "$main_wt/graphify-out/graph.json" 2>/dev/null | head -1 || true)
if [ "$graph_status" -eq 0 ] && [ "$graph_built" != "$(git -C "$main_wt" rev-parse HEAD)" ]; then
  graph_status=6
  echo "ERROR: graphify-out/graph.json built_at_commit ($graph_built) is not main HEAD" >&2
fi
[ "$graph_status" -eq 0 ] && echo "== graph built at $(git -C "$main_wt" rev-parse --short HEAD)"

changed=$(git -C "$main_wt" diff --name-only "$before" HEAD)
if [ -z "$changed" ]; then
  echo "== $branch was already in main; nothing changed, no rebuild"
  remove_worktree
  exit "$graph_status"
fi

if printf '%s\n' "$changed" | grep -qx 'docker-compose.yml'; then
  echo "WARN: docker-compose.yml changed; services without code changes were NOT recreated"
fi

# services/<name>/ maps to the compose service <name> when it has a Dockerfile.
compose_services=$(cd "$main_wt" && docker compose config --services)
svcs=()
for name in $(printf '%s\n' "$changed" | sed -n 's#^services/\([^/]*\)/.*#\1#p' | sort -u); do
  if [ -f "$main_wt/services/$name/Dockerfile" ] && printf '%s\n' "$compose_services" | grep -qx "$name"; then
    svcs+=("$name")
  fi
done

if [ "${#svcs[@]}" -eq 0 ]; then
  echo "== no service code changed; no rebuild"
  remove_worktree
  exit "$graph_status"
fi

(cd "$main_wt" && scripts/worktree.sh rebuild "${svcs[@]}") || exit $?

echo "== delivered $branch -> main ($(git -C "$main_wt" rev-parse --short HEAD)); healthy: ${svcs[*]}"
remove_worktree
exit "$graph_status"
