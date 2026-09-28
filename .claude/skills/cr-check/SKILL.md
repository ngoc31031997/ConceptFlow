---
name: cr-check
description: Run the repository verification (make check, or make check-all) and summarize failures as actionable items. Use before declaring a task or Change Request done, after a batch of edits, or when the Creator asks whether the branch is green.
argument-hint: "[all]"
---

# /cr-check

## Run

- No argument: `make check` (lint + unit tests for the services changed vs `main`, plus contract/hook tests when those changed).
- `all`: `make check-all` (every service + contract + hook tests, what CI runs).

Run it with a timeout of at least 600000 ms. Do not run individual `go test`/`pytest`/`npm test` commands instead: `make check` is the definition of green (see `docs/agentic/verification.md`).

## Report

On success: one line, e.g. `make check: all 23 steps passed (web-gui, tts)`. Mention that a pass on a clean working tree also records the merge-gate marker; a pass with uncommitted changes does not.

On failure, for each failing step (the output ends with the last 25 lines of each; full logs are in `$TMPDIR/conceptflow-check/`):

| Step | What failed | Likely cause | Next action |
|---|---|---|---|

- Read the full log file when the tail is not enough to name the cause. Do not guess.
- Separate failures caused by the current branch's changes from pre-existing ones. For a suspected pre-existing failure, show that neither the failing test nor the code it exercises changed on this branch (`git diff --stat main...HEAD -- <paths>`). Call it pre-existing only with that evidence.
- `run 'make setup' first` means a missing `.venv`/`node_modules`. Suggest `make setup SERVICES=<svc>`; it is not a code failure.

## Then

- Failures in the current task's scope: fix them and run `/cr-check` again.
- Failures outside that scope, or ones that cannot be fixed honestly: report them and stop. Never skip, delete or weaken a test to get green, and never report green when a step failed (CLAUDE.md).
