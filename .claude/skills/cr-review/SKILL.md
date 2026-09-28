---
name: cr-review
description: Review the current Change Request branch before merge - code review, security review, and QC of the CR's acceptance criteria against the implementation and tests - and give a PASS/FAIL verdict with actionable findings. Use before /cr-finish, or when the Creator asks for a review of a CR branch.
argument-hint: "[NNN]  (default: CR number from the branch name)"
---

# /cr-review

> **Interim version (Phase 6).** Code and security review use Claude Code's built-in `code-review` and `security-review` skills; QC is done in this session. None of it is independent of the session that wrote the code. Phase 7 replaces these steps with the read-only role agents in `.claude/agents/` (`solution-architect`, `reviewer`, `security-reviewer`, `tester`).

## 0. Scope

- CR number: `$ARGUMENTS`, or parse it from the branch (`feature/cr-<NNN>-<slug>`). On `main` or a branch without a CR number, stop and ask what to review.
- The change under review is `git diff main...HEAD` plus any uncommitted changes. Print `git diff --stat main...HEAD` first.
- CR documents: `aidlc-docs/inception/requirements/cr-<NNN>-*.md` and the `## CR-<NNN>` entries in `aidlc-docs/audit.md`. Relevant ADRs: `aidlc-docs/decisions/`.

## 1. Code review

Invoke the `code-review` skill (Skill tool) on this branch at level `high`. Keep only findings that hold up when you re-read the code.

## 2. Security review

Invoke the `security-review` skill. Known accepted decisions are not findings: ADR-0016 (OAuth credentials stored in plaintext) is accepted.

If either skill is unavailable in this environment, say so and stop. Never report the review as done without it.

## 3. QC: acceptance criteria vs implementation vs tests

From the CR requirement doc, list every functional requirement / acceptance criterion (FR ids, numbered items, "Definition of done"). For each:

| Criterion | Implemented in | Tested by | Status |
|---|---|---|---|
| FR-x … | `path:line` | `test file::test name` | covered / not tested / not implemented |

Cite real paths and test names; open the files to confirm. "Not tested" and "not implemented" are findings.

If the CR has no requirement doc, say so. QC then runs against the audit entry's user input and is marked weaker.

## 4. Verdict

Findings table, most severe first:

| # | Severity | Source (code/security/QC) | File:line | Finding | Fix |
|---|---|---|---|---|---|

Severity: **Blocker** (wrong behaviour, data loss, security hole, acceptance criterion not implemented), **Major** (missing test for a criterion, regression risk, ADR violation), **Minor** (everything else).

- **PASS**: no Blocker or Major findings open.
- **FAIL**: list what must change. Fix in-scope findings, run `/cr-check`, then `/cr-review` again.

Report the verdict and findings in chat; do not write them into files. Inside `/cr-finish` a commit after `/cr-check` would invalidate the merge-gate marker. The Impact Assessment of the audit entry, written before `/cr-finish`, is where review results are recorded.
