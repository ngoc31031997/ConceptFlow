---
name: fix
description: Fix a bug the Creator reports - open a fix/<slug> branch in its own git worktree, find the root cause through graphify, apply the smallest correct fix with a regression test, rebuild the changed Docker services, and report. Use when the Creator types /fix or reports something that is broken (an error, a crash, wrong output, a screen that misbehaves), not a request for new behaviour. Commits nothing; the Creator runs /deliver after approving.
argument-hint: "<the bug, in the Creator's own words: what happened, where, what was expected>"
model: claude-opus-5-5
effort: medium
---

# /fix

The bug: `$ARGUMENTS`

Answer the Creator in Vietnamese. This skill fixes broken behaviour. A request for new or changed behaviour is a CR: tell the Creator and suggest `/cr` instead.

## 1. Own worktree and branch

Every fix is worked on in its own git worktree, so it can run beside open CRs (`.ai-dlc/aws-aidlc-rule-details/construction/git-branching.md`). The primary checkout (first line of `git worktree list`) stays on a clean `main`: never check out, stash, commit or edit anything there, even when it has uncommitted changes from another CR.

- Pick a short kebab-case slug for the bug. `scripts/worktree.sh add fix/<slug>` creates the branch from `origin/main` in `.claude/worktrees/`, builds its graphify graph and prints its path. If it reports that the branch exists, STOP and ask.
- Switch the session into it: `EnterWorktree` with `path` = that path. From here on every command, read and edit happens in the worktree.
- If the bug is in code that exists only on an open CR branch (not on `main`), STOP and ask whether to fix it on that CR branch instead.
- Once the root cause is known (step 2), check the open CRs and fixes (`git branch -a --no-merged origin/main --list '*feature/cr-*' '*fix/*' '*chore/*'`, `scripts/worktree.sh list` and their uncommitted changes): if one changes the same files or feature, tell the Creator in the report (or STOP and ask, when the fix would clash with it).

## 2. Reproduce and find the root cause

Follow CLAUDE.md "Read the codebase through graphify first". Do not list directories or read files to "get context".

1. Freshness: `graph.json`'s `built_at_commit` must equal `git rev-parse HEAD`, else `make graph`.
2. `graphify-out/GRAPH_REPORT.md`, once per session.
3. Gather the evidence first: the exact error, the logs of the services involved (`docker compose logs --tail=200 <service>`), the request or input that triggers it. Grep the error message in the service directory.
4. `graphify query "<the failing behaviour>"`, `graphify explain "<symbol>"`, `graphify path "<A>" "<B>"` to follow the failing path. For RabbitMQ messages, HTTP between services and DB access, read `docs/contracts/` and the relevant ADR in `aidlc-docs/decisions/`.
5. Read only the files (or line ranges) on that path.
6. Reproduce it when it is safe: a failing test is best; otherwise a live call or the logs. Never delete or overwrite the Creator's real data to reproduce.

Name the root cause with `file:line`, and explain why it produces the symptom. Fix the cause, not the symptom: no retries, guards or fallbacks that hide the error.

If you cannot find or confirm the cause, or the information you need is missing (a log, an input, credentials), STOP and tell the Creator what you found and what you need. Do not fix on a guess.

## 3. Decide whether to ask first

STOP and ask the Creator (numbered options with your recommendation) before changing code when:

- there is more than one reasonable fix and they change behaviour differently;
- the fix changes a contract between services, a DB schema or migration, or stored data;
- the fix is really a change of behaviour or a design gap (then suggest `/cr`);
- `graphify affected "<symbol>"` shows callers whose behaviour the fix would also change in a way the Creator may not want.

Otherwise go straight to step 4.

## 4. Fix

- Before changing a symbol's behaviour, run `graphify affected "<symbol>"` and check its callers.
- Make the smallest change that fixes the root cause. No refactors or unrelated clean-ups.
- Every code change follows `docs/code-standards-rules.md` (comments describe what the code does and why, no bug/CR numbers or change history). UI: `docs/ux-ui-design-rules.md`.
- No stubs, fake outputs, swallowed errors or paths that pretend to succeed (CLAUDE.md "No fake code").
- Add a regression test that fails without the fix and passes with it. If a test is not practical (e.g. pure visual CSS), say why in the report.
- If the same bug pattern exists elsewhere, list those places in the report; fix them only if they are clearly the same bug.

## 5. Verify

- Run the tests of every changed service (`go test ./...`, `pytest`, `npm test`, ...). Report failures with their output; do not make them pass dishonestly.
- Rebuild only the changed services (`git status` / `git diff origin/main --stat` to scope) with `scripts/worktree.sh rebuild <service>...`, run in the worktree: it builds the images from the worktree's code, restarts them in the primary checkout (where `.env`, secrets and volumes live) and waits until they are healthy. Do not run `docker compose up` in a worktree. The live stack is shared: the rebuild replaces whatever another open branch last deployed to that service, so name the branch now live in the report.
- Re-run the reproduction from step 2 live when it is safe, and confirm the symptom is gone.

## 6. Record and report

- Add a `## Fix — <tiêu đề ngắn>` entry at the end of `aidlc-docs/audit.md` in the existing format (Timestamp, User Input verbatim, AI Response: root cause and fix, Impact Assessment, Artifacts Affected), including the real test results and rebuilt services.
- Report to the Creator: the branch and worktree path, the root cause (`file:line`, why), what changed per service, the regression test, test results, rebuilt services and their health, the live check result, and anything not done or not checked and why. Then wait.
- Commit nothing. After the Creator approves, they run `/deliver`. If they ask for changes, apply them, verify again, and report again.
