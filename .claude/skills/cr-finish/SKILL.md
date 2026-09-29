---
name: cr-finish
description: Finish an approved Change Request - verify the diff and AI-DLC records, bring the branch up to date with main, pass make check and /cr-review, then merge into main through the merge gate, push, and confirm CI. Use only after the Creator has approved the CR's final stage.
argument-hint: ""
effort: low
---

# /cr-finish

Implements CLAUDE.md's *CR completion policy* under decision D1 (auto-merge kept, gated). The merge gate hook (`scripts/hooks/guard_bash.py`, see `docs/agentic/hooks.md`) enforces the `make check` and review parts; this skill must not look for ways around it. Never `--force`, never `--no-verify`, never edit the gate marker directory by hand.

Stop and report at the first step that fails. Do not continue past it.

Who does what (to save tokens): the main session keeps the judgement steps (approval, diff sanity, AI-DLC records, merge). `/cr-check` runs in the `ops-runner` agent (haiku), `/cr-review` in the read-only role agents (sonnet), and the CI watch in `ops-runner` in the background. This skill cannot itself run as a forked agent: agents cannot start other agents, and `/cr-review` needs to.

## 1. Preconditions

- The Creator explicitly approved this CR's final stage in this conversation ("ok"/"approve"/"go" after the completion message). No approval → stop and ask. A request made earlier in the conversation, or an approval of an earlier stage, is not approval.
- Current branch is `feature/cr-<NNN>-<slug>`, `feature/<slug>` or `fix/<slug>` (from `/fix-bug`), not `main`.

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
2. `/cr-review`. All three agents must return PASS for this tree; the SubagentStop hook records their markers.
   `scripts/review-status.sh` must then exit 0 (all four markers). FAIL or a missing marker → stop, report, and let the Creator decide.

Any commit made after step 5.1 (review fixes, audit updates) is a new tree without markers: go back to 5.1.

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

Do not poll from the main session. Hand the watch to the `ops-runner` agent (haiku) with `run_in_background: true`, and meanwhile write the report. Its prompt, with `<sha>` = `git rev-parse main`:

> Watch GitHub CI for commit `<sha>` of the public repository ngoc31031997/ConceptFlow (no token needed). Run
> `curl -s "https://api.github.com/repos/ngoc31031997/ConceptFlow/actions/runs?head_sha=<sha>" | jq -r '.workflow_runs[0] | "\(.status) \(.conclusion) \(.html_url) \(.id)"'`
> every 30 s (`sleep 30` between runs) until the status is `completed`, at most 40 minutes. Report the conclusion and the link. If the conclusion is not `success`, also fetch `https://api.github.com/repos/ngoc31031997/ConceptFlow/actions/runs/<id>/jobs`, and for each failed job `https://api.github.com/repos/ngoc31031997/ConceptFlow/check-runs/<job id>/annotations`, and quote the failing annotations.

When it returns, add the CI result to the report. A red CI on `main` is reported immediately as the top line, with the failing annotations. No result within 40 minutes: report CI as **not confirmed**, with the run link.

## 8. Report

CR number, merge commit, branch pushed, `make check` result, review verdict, CI result, and anything not done (with the reason).
