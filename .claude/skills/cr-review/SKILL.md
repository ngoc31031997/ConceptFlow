---
name: cr-review
description: Independent review of the current Change Request branch before merge - runs the read-only reviewer, security-reviewer and tester agents on the committed tree and gives a PASS/FAIL verdict with actionable findings. Use before /cr-finish, or when the Creator asks for a review of a CR branch.
argument-hint: "[NNN]  (default: CR number from the branch name)"
---

# /cr-review

The review is done by three read-only agents in `.claude/agents/`, not by this session: the session that wrote the code must not be the only one judging it. This skill prepares their input, runs them, and consolidates what they say. It must not soften, drop or re-grade their findings.

## 0. Scope

- Label: `CR-<NNN>` from `$ARGUMENTS` or the branch name (`feature/cr-<NNN>-<slug>`). On `main`, or a branch without a CR number and no argument, stop and ask what to review and against which requirement doc.
- Requirement doc: `aidlc-docs/inception/requirements/cr-<NNN>-*.md` if it exists.
- The working tree must be **clean**: the review is of a committed tree, the same one `make check` marks. Uncommitted changes → stop and ask to commit (or commit them if the Creator's stage approval covers them).
- Run `/cr-check` first if it has not passed on this tree; the brief tells the agents whether it did.

## 1. Prepare input (script, not by hand)

```bash
scripts/review-prep.sh CR-<NNN> [requirement-doc]
```

It writes the diff and the **brief** for the current tree under `.git/conceptflow/review/` and prints the brief. Do not write or edit these files yourself; the directory is blocked for the shell and for Edit/Write anyway.

## 2. Run the agents, in parallel

One message with three Agent calls, `subagent_type`: `reviewer`, `security-reviewer`, `tester`, foreground (`run_in_background: false`). The `prompt` of each is the brief **exactly as printed**: no additions, no summary, no context of your own. The SubagentStop hook compares the agent's prompt with the brief byte for byte and records nothing if they differ. If you believe an agent needs more context, tell the Creator instead.

A review counts only if the hook wrote its marker. Check after the agents return:

```bash
scripts/review-status.sh
```

An agent without a marker has not completed its review (the hook's reason is in the transcript: wrong tree, altered brief, diff changed, no verdict line). Fix the cause and re-run that agent once; if it still has no marker, report that review as **not done**, never as passed.

## 3. Consolidate

| Agent | Verdict | Blocker | Major | Minor |
|---|---|---|---|---|

Then all Blocker and Major findings, with the agent that raised them, file:line and fix. Minor findings in a short list.

- **PASS** only when all three agents returned `VERDICT: PASS` for the current tree **and** `scripts/review-status.sh` shows all three markers.
- **FAIL**: fix the findings that are in the CR's scope, commit, run `/cr-check` and `/cr-review` again (a new commit is a new tree; old verdicts no longer apply). A finding you believe is wrong: say so with evidence and let the Creator decide. Do not re-run an agent hoping for a different verdict.

Report in chat. Do not write the verdict into repository files: inside `/cr-finish` a commit after `/cr-check` would invalidate the merge-gate markers.

## Design review

For the AI-DLC Design stage (before code), use the `solution-architect` agent on the design document instead. It is not part of this skill or the merge gate.
