---
name: cr-start
description: Start a new Change Request - check the working tree is clean, sync main, pick the next CR number, create the feature/cr-NNN-slug branch, and open the AI-DLC Requirements Analysis for it. Use when the Creator asks for a new change, fix or feature that is not already an open CR.
argument-hint: "<short-slug> [NNN]  e.g. illustration-faces"
---

# /cr-start

Arguments: `$ARGUMENTS`: a kebab-case slug, and optionally an explicit CR number.

## 1. Clean starting state

```bash
git status --porcelain
git rev-parse --abbrev-ref HEAD
```

- Uncommitted changes: **stop**. List them and ask the Creator whether they belong to another open CR (another agent may be mid-work) or should be committed/stashed. Never discard them.
- Already on a `feature/cr-*` branch for a different, still-open CR: do not reuse it (`.ai-dlc/aws-aidlc-rule-details/construction/git-branching.md`).

## 2. Sync main

```bash
git fetch origin
git checkout main
git pull origin main
```

If the pull is not a fast-forward, stop and report. Do not resolve a divergent `main` on your own.

## 3. CR number

Unless given, the next number is one above the highest of:

```bash
grep -oE '^## CR-[0-9]{3}' aidlc-docs/audit.md | grep -oE '[0-9]{3}' | sort -n | tail -1
ls aidlc-docs/inception/requirements/ | grep -oE '^cr-[0-9]{3}' | grep -oE '[0-9]{3}' | sort -n | tail -1
git branch -a --list '*feature/cr-*' | grep -oE 'cr-[0-9]{3}' | grep -oE '[0-9]{3}' | sort -n | tail -1
```

Branches count because another agent may have started a CR not yet in the audit log. Zero-pad to three digits.

## 4. Branch

```bash
git checkout -b feature/cr-<NNN>-<slug>
```

If the branch name already exists locally or on `origin`, stop and ask. It may be someone else's work.

## 5. AI-DLC context

Read `.ai-dlc/steering/aws-aidlc-rules/core-workflow.md`, then the rule details for the stage the CR starts in (normally `inception/` Requirements Analysis). Follow them; this skill does not replace the AI-DLC workflow.

Record the CR the way existing CRs are recorded: look at the most recent `aidlc-docs/inception/requirements/cr-*.md` and the latest `## CR-…` entry in `aidlc-docs/audit.md` for the format, and write the Creator's request verbatim into the audit entry.

## 6. Report and stop

Report: CR number, branch name, base commit of `main`, and the requirement questions or analysis the AI-DLC stage needs. Then wait for the Creator's approval before design or code, and commit nothing until that approval (CLAUDE.md commit policy).
