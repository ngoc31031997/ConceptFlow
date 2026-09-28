# Verification entry points shared by developers, Claude Code hooks and CI.
# The logic lives in scripts/{setup,build,check}.sh; see docs/agentic/verification.md.

BASE ?= main
SERVICES ?=

.PHONY: help setup build check check-all

help:
	@echo "make setup      install per-service dev tooling (.venv, node_modules, go modules); SERVICES=\"a b\" to limit"
	@echo "make build      compile every service (go build, npm run build / node --check, compileall)"
	@echo "make check      lint + unit tests for services changed vs $(BASE) (BASE=<ref> to override)"
	@echo "make check-all  lint + unit tests for every service, plus contract tests (used by CI)"

setup:
	@./scripts/setup.sh $(SERVICES)

build:
	@./scripts/build.sh

check:
	@BASE=$(BASE) ./scripts/check.sh changed

check-all:
	@./scripts/check.sh all
