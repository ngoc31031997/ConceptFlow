#!/usr/bin/env bash
# Shows what the merge gate knows about the current HEAD's tree: the make check
# marker, the three review markers, and the review hook's recent log. Read-only.
# The marker directories are blocked for direct shell access; this is the way to
# inspect them (used by /cr-review and /cr-finish).
#
# Exit 0 when HEAD's tree has all four markers (the gate would let it merge), 1 otherwise.
set -uo pipefail

cd "$(dirname "$0")/.." || exit 2
tree="$(git rev-parse 'HEAD^{tree}')"
base="$(git rev-parse --path-format=absolute --git-common-dir)/conceptflow"

echo "tree $tree ($(git rev-parse --abbrev-ref HEAD) @ $(git rev-parse --short HEAD))"
[ -n "$(git status --porcelain)" ] && echo "note: uncommitted changes; markers apply to the committed tree only"

ok=0
mark() {
  if [ -f "$1" ]; then echo "  [x] $2"; else echo "  [ ] $2"; ok=1; fi
}
mark "$base/checked-trees/$tree" "make check"
for agent in reviewer security-reviewer tester; do
  mark "$base/reviewed-trees/$tree.$agent" "review: $agent"
done

if [ -f "$base/review/hook.log" ]; then
  echo "review hook log (last 6):"
  tail -n 6 "$base/review/hook.log" | sed 's/^/  /'
fi
exit "$ok"
