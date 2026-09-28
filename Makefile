# Verification entry points shared by developers, Claude Code hooks and CI.
# The logic lives in scripts/check.sh and scripts/setup.sh; see docs/agentic/verification.md.

BASE ?= main
SERVICES ?=

.PHONY: help setup check check-all

help:
	@echo "make setup      install per-service dev tooling (.venv, node_modules, go modules); SERVICES=\"a b\" to limit"
	@echo "make check      lint + unit tests for services changed vs $(BASE) (BASE=<ref> to override)"
	@echo "make check-all  lint + unit tests for every service, plus contract tests (used by CI)"

setup:
	@./scripts/setup.sh $(SERVICES)

check:
	@BASE=$(BASE) ./scripts/check.sh changed

check-all:
	@./scripts/check.sh all
