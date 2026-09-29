---
name: fix-bug
description: Fix a bug the Creator found - branch fix/<slug>, log the report, reproduce it with a failing test, find the root cause, apply the smallest fix, verify (make check, rebuild, live check) and report for approval. Use when the Creator reports something broken in existing behaviour (an error, a wrong result, a UI glitch), not for new features or behaviour changes (those are /cr-start).
argument-hint: "<what is wrong, where, how to reproduce>  (screenshots/logs welcome)"
effort: medium
---

# /fix-bug

The bug report: `$ARGUMENTS` (plus anything the Creator attached). AI-DLC treats a bug with a clear, isolated scope as the minimal case: no user stories, no design stage, short requirement (`.ai-dlc/aws-aidlc-rule-details/common/depth-levels.md`, "Simple Scenario"). This skill is that path. Everything in CLAUDE.md still applies: no fake fixes, no skipped or weakened tests, report faithfully.

## 1. Branch

```bash
git branch --show-current
git status --porcelain
git fetch origin && git checkout main && git pull origin main
git checkout -b fix/<short-slug>
```

- Uncommitted changes, or the bug is in a CR branch that is still open (its own work, not merged): **stop and ask**. It may be fixed on that branch instead, and another agent may be working there.
- Repo convention for bugs is `fix/<slug>` (e.g. `fix/confirm-modal-backdrop`). Check `git branch --show-current` again right before every commit: other agents may share this checkout.

## 2. Log the report

Add an entry at the end of `aidlc-docs/audit.md`, in the format of the latest entries: a `## Fix — <short title>` heading, **Timestamp**, and **User Input** with the Creator's words verbatim (describe attached images in one line). AI Response / Impact Assessment / Artifacts are filled in at step 6.

## 3. Reproduce first

Find the code path from the report (logs: `docker compose logs --tail 200 <svc>`; UI: the page/component; API: the handler). Then write a **test that fails because of the bug**, in the service's existing test suite, named after the behaviour ("… does not reset X when Y"). Run only that test to see it fail for the reported reason.

- Cannot reproduce (no failing test, logs show nothing): **stop**. Report what you tried and what you need (steps, data, screenshot, time of the error). Do not fix a guess.
- A bug that only shows in the running stack (browser layout, a real external API): reproduce it there, record how, and cover the logic you change with a unit test where one is possible. Say explicitly what is not covered by an automated test.

## 4. Root cause, then scope check

State in chat, before editing: the root cause at `file:line`, why it produces the reported symptom, and the fix you will make.

Stop and propose `/cr-start` instead when the fix would:
- change an API/message contract (`docs/contracts/`), a DB schema, or a prompt seed's intended behaviour;
- touch more than one service for different reasons, or need a new ADR;
- change behaviour the Creator did not report as wrong (the "bug" is really a requirement change).

## 5. Fix and verify

1. Smallest change that removes the root cause. No drive-by refactors; mention anything else you noticed as a follow-up instead.
2. The new test passes; the rest of the suite still passes: `/cr-check`.
3. Search for the same mistake elsewhere (`Grep` the pattern). Fix identical instances in the same service; list others.
4. `/rebuild` every service whose code changed (CLAUDE.md Docker rebuild policy), and confirm healthy.
5. Check it live where possible: the API call, log line or page that showed the bug now behaves. If you cannot (needs a real account, a browser, paid API), say so and give the Creator exact steps.

## 6. Report and stop

Complete the audit entry (**AI Response**: root cause + fix; **Impact Assessment**: tests run with results, rebuilds, live check; **Artifacts Affected**). Then report in chat:

- Root cause (one or two sentences, `file:line`)
- Fix (what changed, why it is enough)
- Regression test (file :: name), `make check` result, rebuild status
- How the Creator can verify, and anything not verified

Commit nothing until the Creator approves (CLAUDE.md commit policy). After "ok": `/cr-finish` (label `fix-<slug>` when `/cr-review` asks for one).
