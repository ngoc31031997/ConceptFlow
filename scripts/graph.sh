#!/usr/bin/env bash
# Local code knowledge graph (graphify) that agents query to orient themselves
# before reading code. See docs/agentic/graphify.md.
#
#   scripts/graph.sh build   build or refresh graphify-out/ for the working tree
#                            (code and markdown structure via local tree-sitter AST;
#                            no LLM call, nothing leaves the machine)
#   scripts/graph.sh hooks   install graphify's post-commit / post-checkout git hooks
#                            in this clone, so the graph follows commits and branch switches
#
# graphify-out/ is git-ignored. Exit 0 ok; 1 graphify missing or failed; 2 usage error.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# PATH first; ~/.local/bin is where `pipx install graphifyy` puts it, and GUI git
# clients / fresh agent shells do not always have it on PATH.
GRAPHIFY="${GRAPHIFY:-$(command -v graphify 2>/dev/null || echo "$HOME/.local/bin/graphify")}"
if [ ! -x "$GRAPHIFY" ]; then
  echo "graph: graphify is not installed. Install it with:" >&2
  echo "  brew install pipx && pipx install --python python3.12 graphifyy && pipx ensurepath" >&2
  exit 1
fi

case "${1:-build}" in
  build)
    # --force: the graph must mirror the tree even after a refactor deletes code
    # (without it graphify keeps the old, larger graph).
    "$GRAPHIFY" update . --force
    ;;
  hooks)
    had_attributes=0
    [ -e .gitattributes ] && had_attributes=1
    "$GRAPHIFY" hook install
    # hook install also registers a union merge driver for a committed graph.json in
    # .gitattributes. Our graph is git-ignored, so the line has no use; drop the file
    # if the install created it, rather than leave an untracked file in the tree.
    if [ "$had_attributes" = 0 ] && [ -e .gitattributes ] && [ -z "$(git ls-files .gitattributes)" ]; then
      rm .gitattributes
      echo "graph: removed .gitattributes created by graphify (graph.json is not committed, no merge driver needed)"
    fi
    ;;
  *)
    echo "usage: $0 [build|hooks]" >&2
    exit 2
    ;;
esac
