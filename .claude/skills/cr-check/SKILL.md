---
name: cr-check
description: Run the repository verification (make check, or make check-all) and summarize failures as actionable items. Use before declaring a task or Change Request done, after a batch of edits, or when the Creator asks whether the branch is green.
argument-hint: "[all]"
context: fork
agent: ops-runner
---

# /cr-check

Runs in the `ops-runner` agent (haiku), not in the main session: the long lint/test output stays out of the main context. This agent reports; the main session diagnoses and fixes.

## Run

- No argument: `make check` (lint + unit tests for the services changed vs `main`, plus contract/hook tests when those changed).
- `all`: `make check-all` (every service + contract + hook tests, what CI runs).

Argument given: `$ARGUMENTS`

Run it with a timeout of at least 600000 ms. Do not run individual `go test`/`pytest`/`npm test` commands instead: `make check` is the definition of green (see `docs/agentic/verification.md`).

## Report

On success: one line, e.g. `make check: all 23 steps passed (web-gui, tts)`, and whether the working tree was clean (`git status --porcelain` empty). A pass on a clean tree also records the merge-gate marker; a pass with uncommitted changes does not.

On failure, for each failing step (the output ends with the last 25 lines of each; full logs are in `$TMPDIR/conceptflow-check/`):

| Step | Failing test / lint rule | Error (verbatim, short) | File:line | Changed on this branch? |
|---|---|---|---|---|

- Read the full log file when the tail does not show the error itself. Quote, do not paraphrase.
- "Changed on this branch?": run `git diff --stat main...HEAD -- <failing test file> <code file it exercises>` plus `git status --porcelain -- <same paths>`. `yes` / `no` with the command output as evidence. Only `no` for both the test and the code it exercises may later be called pre-existing.
- `run 'make setup' first` means a missing `.venv`/`node_modules`: report `setup missing: make setup SERVICES=<svc>`; it is not a code failure.

End the failure report with this line, verbatim:

`Next (main session): fix failures in the current task's scope and run /cr-check again; report failures outside that scope, or ones that cannot be fixed honestly, and stop. Never skip, delete or weaken a test to get green, and never report green when a step failed (CLAUDE.md).`
