---
name: code
description: Implement an approved Change Request - work through the approved tasks in aidlc-docs/construction/plans/cr-NNN-*-tasks.md on its feature branch in the CR's own worktree, run the tests, rebuild the changed Docker services, and report. Invoked by the /cr skill after the Creator approves the tasks, or by the Creator with /code CR-NNN.
argument-hint: "CR-<NNN>"
model: sonnet
effort: medium
---

# /code

CR: `$ARGUMENTS`. Answer the Creator in Vietnamese.

## 1. Preconditions

- The tasks doc `aidlc-docs/construction/plans/cr-<NNN>-*-tasks.md` exists and the Creator approved it in this conversation (or the audit records the approval). No tasks doc or no approval: STOP and ask.
- Work in the CR's own worktree (`git-branching.md`): `scripts/worktree.sh path feature/cr-<NNN>-<slug>`; if the session is not already in it, switch with `EnterWorktree` (`path`). If the branch or its worktree does not exist, STOP and ask. Never code on `main` or in the primary checkout, and never check out the CR branch there.
- Re-check the design's **Phụ thuộc**: if a branch it depends on is still not in `origin/main`, STOP and tell the Creator.

## 2. Implement the plan

Read the tasks doc. Read the design doc only for its **Yêu cầu** and **Giải pháp đề xuất** when a task needs that context. Read only the files and line ranges the tasks name. For anything else use graphify (`graphify explain`, `graphify affected`) after checking that `graph.json`'s `built_at_commit` equals `git rev-parse HEAD` (else `make graph`); do not explore the codebase.

Work through the tasks in order, skipping tasks already ticked `- [x] Xong` (a resumed run):

- Match the surrounding code's style and naming. Every code change follows `docs/code-standards-rules.md` (documentation comments only, no CR numbers or change history; basic code standards). UI: `docs/ux-ui-design-rules.md`.
- No stubs, fake outputs, `TODO: implement`, swallowed errors (CLAUDE.md "No fake code").
- Add or update the tests the task names and run them. When they pass and the task's **Xong khi** holds, tick its checkbox (`- [x] Xong`) in the tasks doc before starting the next task.
- If a task turns out wrong or impossible, or the code contradicts it (a line, signature or caller that is not as described), STOP and ask. Do not redesign on your own and do not skip the task.

## 3. Verify

- Run the tests of every changed service (the service's own test command: `go test ./...`, `pytest`, `npm test`, ...) in the worktree. Git-ignored dependencies are per worktree: install them there when missing (e.g. `npm ci`), never link them from the primary checkout. Report failures with their output; do not make them pass dishonestly.
- Rebuild only the changed services (`git status` / `git diff origin/main --stat` to scope) with `scripts/worktree.sh rebuild <service>...`, run in the worktree: it builds the images from the worktree's code, restarts them in the primary checkout (where `.env`, secrets and volumes live) and waits until they are healthy. Do not run `docker compose up` in a worktree. The live stack is shared: the rebuild replaces whatever another open branch last deployed to that service, so name the branch now live in the report.
- Run the live checks in the tasks doc's **Sau cùng** when they are safe. Never delete or overwrite the Creator's real data to test.

## 4. Record and report

- Add a `## CR-<NNN> — Code xong, chờ duyệt` entry to `aidlc-docs/audit.md`: what was implemented per service, real test results, services rebuilt and their health, anything not done or not checked, and why.
- Report to the Creator: each task's status (done / not done and why), what changed per service, test results, rebuilt services, anything skipped or unverified. Then wait.
- Commit nothing. After the Creator approves, they run `/deliver`.
