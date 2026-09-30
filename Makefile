# graphify code graph (docs/agentic/graphify.md). The verification layer
# (make setup/build/check/check-all, CI) was removed on 2026-09-30.

.PHONY: help graph graph-hooks

help:
	@echo "make graph      build/refresh the local graphify code graph in graphify-out/ (no LLM; docs/agentic/graphify.md)"
	@echo "make graph-hooks  install graphify git hooks so the graph follows commits and checkouts"

graph:
	@./scripts/graph.sh build

graph-hooks:
	@./scripts/graph.sh hooks
