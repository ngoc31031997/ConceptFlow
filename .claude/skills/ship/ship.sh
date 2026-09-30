#!/usr/bin/env bash
# Ship an approved, already committed branch: push it, merge it into main in the
# primary checkout, push main, then rebuild and restart only the Docker services
# whose code changed, and wait until they report healthy.
#
# Usage: .claude/skills/ship/ship.sh <branch>
# Exit codes: 1 bad input/state, 2 push failed, 3 merge conflict (merge aborted),
#             4 docker build/up failed, 5 a service is not healthy.
set -euo pipefail

branch="${1:-}"
[ -n "$branch" ] || { echo "usage: $0 <branch>" >&2; exit 1; }
[ "$branch" != "main" ] || { echo "ERROR: refusing to ship main itself; ship a feature/fix/chore branch" >&2; exit 1; }
git rev-parse --verify --quiet "refs/heads/$branch" >/dev/null || { echo "ERROR: no local branch $branch" >&2; exit 1; }

# The first worktree is the primary checkout; main and the compose project live there.
main_wt=$(git worktree list --porcelain | sed -n 's/^worktree //p' | head -1)
echo "== primary checkout: $main_wt"

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

changed=$(git -C "$main_wt" diff --name-only "$before" HEAD)
if [ -z "$changed" ]; then
  echo "== $branch was already in main; nothing changed, no rebuild"
  exit 0
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
  exit 0
fi

echo "== rebuild: ${svcs[*]}"
(cd "$main_wt" && docker compose build "${svcs[@]}" && docker compose up -d "${svcs[@]}") \
  || { echo "ERROR: docker compose build/up failed" >&2; exit 4; }

echo "== waiting for health (up to 300s)"
deadline=$(( $(date +%s) + 300 ))
while :; do
  pending=0; failed=()
  for s in "${svcs[@]}"; do
    cid=$(cd "$main_wt" && docker compose ps -q "$s")
    st=$(docker inspect -f '{{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$cid" 2>/dev/null || echo "missing none")
    case "$st" in
      "running healthy"|"running none") ;;
      "running starting") pending=1 ;;
      *) failed+=("$s ($st)") ;;
    esac
  done
  [ "${#failed[@]}" -eq 0 ] && [ "$pending" -eq 0 ] && break
  if [ "${#failed[@]}" -gt 0 ] || [ "$(date +%s)" -ge "$deadline" ]; then
    echo "ERROR: not healthy: ${failed[*]:-} $([ "$pending" -eq 1 ] && echo '(some still starting at timeout)')" >&2
    for s in "${svcs[@]}"; do (cd "$main_wt" && docker compose logs --tail 30 "$s") >&2 || true; done
    exit 5
  fi
  sleep 5
done

(cd "$main_wt" && docker compose ps "${svcs[@]}")
echo "== shipped $branch -> main ($(git -C "$main_wt" rev-parse --short HEAD)); healthy: ${svcs[*]}"
