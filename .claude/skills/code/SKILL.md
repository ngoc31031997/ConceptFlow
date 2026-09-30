---
name: code
description: Implement an approved Change Request design - follow the "Kế hoạch thực hiện" of aidlc-docs/construction/plans/cr-NNN-*-design.md on its feature branch, run the tests, rebuild the changed Docker services, and report. Invoked by the /cr skill after the Creator approves the solution, or by the Creator with /code CR-NNN.
argument-hint: "CR-<NNN>"
model: sonnet
effort: medium
---

# /code

CR: `$ARGUMENTS`. Answer the Creator in Vietnamese.

## 1. Preconditions

- The design `aidlc-docs/construction/plans/cr-<NNN>-*-design.md` exists and the Creator approved it in this conversation (or the audit records the approval). No approval: STOP and ask.
- Current branch is `feature/cr-<NNN>-<slug>`. Otherwise, check it out; if it does not exist, STOP and ask. Never code on `main`.

## 2. Implement the plan

Read the design doc. Then read only the files the plan names (graphify for anything else: `graphify explain`, `graphify affected`; CLAUDE.md "Read the codebase through graphify first").

Follow "Kế hoạch thực hiện" step by step:

- Match the surrounding code's style, naming and comment density. UI: `docs/ux-ui-design-rules.md`.
- No stubs, fake outputs, `TODO: implement`, swallowed errors (CLAUDE.md "No fake code").
- Add or update the tests the plan names.
- If a step turns out wrong or impossible, or the code contradicts the design, STOP and ask. Do not redesign on your own and do not skip the step.

## 3. Verify

- Run the tests of every changed service (the service's own test command: `go test ./...`, `pytest`, `npm test`, ...). Report failures with their output; do not make them pass dishonestly.
- Rebuild only the changed services (`git status` / `git diff main --stat` to scope): `docker compose build <service>` then `docker compose up -d <service>`, and confirm healthy (`docker compose ps`).
- Run the live checks in the design's "Kiểm tra" when they are safe. Never delete or overwrite the Creator's real data to test.

## 4. Record and report

- Add a `## CR-<NNN> — Code xong, chờ duyệt` entry to `aidlc-docs/audit.md`: what was implemented per service, real test results, services rebuilt and their health, anything not done or not checked, and why.
- Report to the Creator: what changed per service, test results, rebuilt services, anything skipped or unverified. Then wait.
- Commit nothing. After the Creator approves, they run `/deliver`.
