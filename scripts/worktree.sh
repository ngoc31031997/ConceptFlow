#!/usr/bin/env bash
# One git worktree per Change Request or fix, so several can be worked on side by side.
# The primary checkout (the first worktree) stays on a clean main and owns the running
# Docker stack: .env, secrets/, data/ and the volumes all live there. Each branch gets
# its own directory under .claude/worktrees/ (git-ignored), created from origin/main;
# a Claude Code session switches into it with EnterWorktree (path).
#
#   scripts/worktree.sh add <branch>      create <branch> from origin/main in its own worktree,
#                                         build its graphify graph and print its path
#   scripts/worktree.sh path <branch>     print the worktree path of <branch>
#   scripts/worktree.sh list              list all worktrees and their branches
#   scripts/worktree.sh rebuild <svc>...  build the images of <svc> from the code of the checkout
#                                         this is run in, restart them in the primary checkout,
#                                         and wait until they report healthy
#   scripts/worktree.sh remove <branch>   remove the clean worktree of <branch>; the branch is kept
#   scripts/worktree.sh prune             remove every worktree under .claude/worktrees/ that is delivered
#                                         (clean, pushed, merged into main); the branches are kept
#
# Images are built where the code is and started where the configuration is: the compose
# project name is pinned in docker-compose.yml, so an image built in a worktree carries the
# same name the primary checkout's `docker compose up` uses.
#
# Exit 0 ok; 1 bad state; 2 usage; 4 docker build/up failed; 5 a service is not healthy.
set -euo pipefail

usage() { sed -n '8,17p' "$0" | sed 's/^# \{0,1\}//' >&2; exit 2; }

primary=$(git worktree list --porcelain | sed -n 's/^worktree //p' | head -1)
root="$primary/.claude/worktrees"

worktree_of() {
  git worktree list --porcelain | awk -v ref="refs/heads/$1" '
    /^worktree / { wt = substr($0, 10) }
    $0 == "branch " ref { print wt; exit }'
}

cmd="${1:-}"
[ -n "$cmd" ] || usage
shift

case "$cmd" in
  add)
    branch="${1:-}"
    [ -n "$branch" ] || usage
    if git rev-parse --verify --quiet "refs/heads/$branch" >/dev/null; then
      echo "ERROR: branch $branch already exists locally (worktree: $(worktree_of "$branch"))" >&2
      exit 1
    fi
    git fetch --quiet origin main
    if git ls-remote --exit-code --heads origin "$branch" >/dev/null; then
      echo "ERROR: branch $branch already exists on origin" >&2
      exit 1
    fi
    dir="$root/${branch//\//-}"
    [ ! -e "$dir" ] || { echo "ERROR: $dir already exists" >&2; exit 1; }
    mkdir -p "$root"
    git worktree add --quiet --no-track -b "$branch" "$dir" origin/main
    # Each worktree has its own git-ignored graphify-out/; the graph must match its code.
    "$dir/scripts/graph.sh" build >/dev/null \
      || echo "WARN: graphify build failed; run 'make graph' in $dir" >&2
    echo "$dir"
    ;;

  path)
    branch="${1:-}"
    [ -n "$branch" ] || usage
    dir=$(worktree_of "$branch")
    [ -n "$dir" ] || { echo "ERROR: no worktree has $branch checked out" >&2; exit 1; }
    echo "$dir"
    ;;

  list)
    git worktree list
    ;;

  rebuild)
    [ "$#" -gt 0 ] || usage
    here=$(git rev-parse --show-toplevel)
    env_args=()
    [ -f "$primary/.env" ] && env_args=(--env-file "$primary/.env")
    echo "== build from $here: $*"
    (cd "$here" && docker compose "${env_args[@]}" build "$@") \
      || { echo "ERROR: docker compose build failed" >&2; exit 4; }
    echo "== restart in $primary: $*"
    (cd "$primary" && docker compose up -d --no-build "$@") \
      || { echo "ERROR: docker compose up failed" >&2; exit 4; }

    echo "== waiting for health (up to 300s)"
    deadline=$(( $(date +%s) + 300 ))
    while :; do
      pending=0; failed=()
      for s in "$@"; do
        cid=$(cd "$primary" && docker compose ps -q "$s")
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
        for s in "$@"; do (cd "$primary" && docker compose logs --tail 30 "$s") >&2 || true; done
        exit 5
      fi
      sleep 5
    done
    (cd "$primary" && docker compose ps "$@")
    echo "== healthy: $* (code from $(git -C "$here" rev-parse --abbrev-ref HEAD))"
    ;;

  remove)
    branch="${1:-}"
    [ -n "$branch" ] || usage
    dir=$(worktree_of "$branch")
    [ -n "$dir" ] || { echo "ERROR: no worktree has $branch checked out" >&2; exit 1; }
    [ "$dir" != "$primary" ] || { echo "ERROR: $branch is checked out in the primary checkout, not a worktree" >&2; exit 1; }
    # Without --force git refuses a worktree with uncommitted or untracked files.
    git worktree remove "$dir"
    echo "== removed $dir (branch $branch kept)"
    ;;

  prune)
    # Other CRs and fixes may be in progress in their own worktrees, so only a worktree
    # whose work is finished is removed: no uncommitted or untracked files, the branch
    # tip pushed to origin and contained in main. A fresh branch from `add` has never
    # been pushed, and /code and /fix leave their work uncommitted, so both are kept.
    git worktree prune
    git worktree list --porcelain | awk '
      /^worktree / { wt = substr($0, 10) }
      /^branch refs\/heads\// { print wt "\t" substr($0, 19) }' |
    while IFS=$'\t' read -r dir branch; do
      case "$dir" in "$root"/*) ;; *) continue ;; esac
      tip=$(git rev-parse "refs/heads/$branch")
      pushed=$(git rev-parse --verify --quiet "refs/remotes/origin/$branch" || true)
      if [ -n "$(git -C "$dir" status --porcelain)" ]; then
        echo "== kept $dir ($branch has uncommitted changes)"
      elif [ "$pushed" != "$tip" ]; then
        echo "== kept $dir ($branch is not pushed to origin)"
      elif ! git merge-base --is-ancestor "$tip" refs/heads/main; then
        echo "== kept $dir ($branch is not merged into main)"
      elif git worktree remove "$dir"; then
        echo "== removed $dir (branch $branch kept)"
      else
        echo "WARN: $dir not removed" >&2
      fi
    done
    # A directory git does not know as a worktree may hold someone's files: report it only.
    registered=$(git worktree list --porcelain | sed -n 's/^worktree //p')
    for dir in "$root"/*/; do
      [ -d "$dir" ] || continue
      dir="${dir%/}"
      printf '%s\n' "$registered" | grep -qxF "$dir" \
        || echo "WARN: $dir is not a git worktree; left in place" >&2
    done
    ;;

  *)
    usage
    ;;
esac
