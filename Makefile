# Verification entry points shared by developers, agents and CI.
# The logic lives in scripts/{setup,build,check}.sh; see docs/agentic/verification.md.

BASE ?= main
SERVICES ?=

.PHONY: help setup build check check-all graph graph-hooks

help:
	@echo "make setup      install per-service dev tooling (.venv, node_modules, go modules); SERVICES=\"a b\" to limit"
	@echo "make build      compile every service (go build, npm run build / node --check, compileall)"
	@echo "make check      lint + unit tests for services changed vs $(BASE) (BASE=<ref> to override)"
	@echo "make check-all  lint + unit tests for every service, plus contract tests (used by CI)"
	@echo "make graph      build/refresh the local graphify code graph in graphify-out/ (no LLM; docs/agentic/graphify.md)"
	@echo "make graph-hooks  install graphify git hooks so the graph follows commits and checkouts"

setup:
	@./scripts/setup.sh $(SERVICES)

build:
	@./scripts/build.sh

check:
	@BASE=$(BASE) ./scripts/check.sh changed

check-all:
	@./scripts/check.sh all

graph:
	@./scripts/graph.sh build

graph-hooks:
	@./scripts/graph.sh hooks
