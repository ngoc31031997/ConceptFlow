#!/usr/bin/env bash
# Prepares the input of the review agents (/cr-review) for the committed tree.
#
#   scripts/review-prep.sh <label> [requirement-doc]
#     label            e.g. CR-050, or "Phase 7 agentic" for work without a CR number
#     requirement-doc  path to the requirement doc the tester checks against
#
# Writes, in <git common dir>/conceptflow/review/:
#   <tree>.diff    git diff <merge-base with $BASE>..HEAD
#   <tree>.brief   the exact prompt every review agent must receive
# and prints the brief. record_review.py only records a verdict when the agent's
# prompt equals <tree>.brief byte for byte and <tree>.diff still equals the real
# diff, so the session that wrote the code cannot hand the agents a softened
# brief or a partial diff.
#
# Exit 0 ok; 2 usage error, dirty working tree, or no merge-base with $BASE.
set -uo pipefail

BASE="${BASE:-main}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

label="${1:-}"
reqdoc="${2:-}"
[ -n "$label" ] || { echo "usage: $0 <label> [requirement-doc]" >&2; exit 2; }
if [ -n "$reqdoc" ] && [ ! -f "$reqdoc" ]; then
  echo "review-prep: requirement doc '$reqdoc' does not exist" >&2
  exit 2
fi
if [ -n "$(git status --porcelain)" ]; then
  echo "review-prep: the working tree is not clean; commit first (the review is of a committed tree)" >&2
  exit 2
fi
base_commit="$(git merge-base "$BASE" HEAD 2>/dev/null)" || {
  echo "review-prep: cannot find merge-base with '$BASE'" >&2
  exit 2
}

tree="$(git rev-parse 'HEAD^{tree}')"
common="$(git rev-parse --path-format=absolute --git-common-dir)"
dir="$common/conceptflow/review"
mkdir -p "$dir"
git diff "$base_commit" HEAD >"$dir/$tree.diff"

if [ -f "$common/conceptflow/checked-trees/$tree" ]; then
  check_status="PASS (make check passed on this tree)"
else
  check_status="NOT RUN on this tree: treat every behaviour change as untested until shown otherwise"
fi

{
  echo "Review ${label} on branch $(git rev-parse --abbrev-ref HEAD). Tree under review: ${tree}."
  echo "Diff: ${dir}/${tree}.diff ($(git diff --shortstat "$base_commit" HEAD | sed 's/^ //'); base ${base_commit:0:12} = merge-base with ${BASE})"
  echo "Requirement doc: ${reqdoc:-none}"
  echo "make check: ${check_status}"
  echo "Changed files:"
  git diff --name-only "$base_commit" HEAD | sed 's/^/- /'
  echo
  echo "Read the whole diff and the code around it. End with the exact verdict line for tree ${tree}."
} >"$dir/$tree.brief"

cat "$dir/$tree.brief"
