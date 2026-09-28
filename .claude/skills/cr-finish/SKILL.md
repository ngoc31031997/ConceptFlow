---
name: cr-finish
description: Finish an approved Change Request - verify the diff and AI-DLC records, bring the branch up to date with main, pass make check and /cr-review, then merge into main through the merge gate, push, and confirm CI. Use only after the Creator has approved the CR's final stage.
argument-hint: ""
---

# /cr-finish

Implements CLAUDE.md's *CR completion policy* under decision D1 (auto-merge kept, gated). The merge gate hook (`scripts/hooks/guard_bash.py`, see `docs/agentic/hooks.md`) enforces the `make check` part; this skill must not look for ways around it. Never `--force`, never `--no-verify`, never edit the gate marker directory by hand.

Stop and report at the first step that fails. Do not continue past it.

## 1. Preconditions

- The Creator explicitly approved this CR's final stage in this conversation ("ok"/"approve"/"go" after the completion message). No approval → stop and ask. A request made earlier in the conversation, or an approval of an earlier stage, is not approval.
- Current branch is `feature/cr-<NNN>-<slug>` (or `feature/<slug>`), not `main`.

## 2. Diff sanity

```bash
git status --porcelain
git diff --stat main...HEAD
```

- Every change belongs to this CR. Unrelated files, debug output, generated artifacts or anything secret-like: stop and ask.
- Uncommitted changes that belong to the CR: commit them now (the approval covers them), with a message describing what was approved.

## 3. AI-DLC records

- `aidlc-docs/audit.md` has this CR's entry with User Input, AI Response, Impact Assessment (real test results, rebuilds done) and Artifacts Affected matching the diff.
- The CR requirement doc and `aidlc-docs/aidlc-state.md` are updated if the CR changed them. Look at how previous CRs did it.
- Missing or stale: update, commit, and say so in the report.

## 4. Up to date with main

```bash
git fetch origin
git merge origin/main
```

A conflict: `git merge --abort`, stop and report (CLAUDE.md: never resolve blindly). New commits from main affecting running services → `/rebuild`.

## 5. Verify

1. `/cr-check`. It must pass on a **clean** working tree; this records the gate marker for the tree.
2. `/cr-review`. Verdict must be PASS. FAIL → stop, report the findings, and let the Creator decide.

Any commit made after step 5.1 (review fixes, audit updates) invalidates the marker: go back to 5.1.

## 6. Merge and push

```bash
git push -u origin <branch>
git checkout main
git pull origin main
git merge --no-ff <branch> -m "Merge <branch>: CR-<NNN> <title>"
```

Then, as a **separate** command (the gate checks the tree that is actually pushed):

```bash
git push origin main
```

If the gate blocks: report its message verbatim. `main` moved during the pull → return to the branch, go back to step 4.

## 7. CI

The repository is public; watch the run for the pushed commit in the background (no token needed):

```bash
curl -s "https://api.github.com/repos/ngoc31031997/ConceptFlow/actions/runs?head_sha=$(git rev-parse main)" \
  | jq -r '.workflow_runs[0] | "\(.status) \(.conclusion) \(.html_url)"'
```

Poll every 30 s until `completed`. Report the conclusion with the link. A red CI on `main` is reported immediately as the top line, with the failing annotations (`/repos/…/check-runs/<id>/annotations`).

## 8. Report

CR number, merge commit, branch pushed, `make check` result, review verdict, CI result, and anything not done (with the reason).
