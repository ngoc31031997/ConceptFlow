---
name: cr
description: Start a new Change Request from the Creator's request - open the feature/cr-NNN-slug branch, analyse the codebase through graphify, and propose a solution (requirement + design) for the Creator to approve. Use when the Creator types /cr or asks for a new feature or change of behaviour that is not already an open CR. Writes no production code.
argument-hint: "<the Creator's request, in their own words>"
model: claude-opus-5-5
effort: medium
---

# /cr

The Creator's request: `$ARGUMENTS`

This skill analyses and proposes. It writes no production code and commits nothing before the Creator approves. Answer the Creator in Vietnamese.

## 1. Clean start, branch

```bash
git status --porcelain
git rev-parse --abbrev-ref HEAD
```

- Uncommitted changes: STOP. List them and ask whether they belong to another open CR. Never discard them.
- Sync main: `git fetch origin && git checkout main && git pull --ff-only origin main`. If it is not a fast-forward, STOP and report.
- CR number: one above the highest of
  ```bash
  grep -oE '^## CR-[0-9]{3}' aidlc-docs/audit.md | grep -oE '[0-9]{3}' | sort -n | tail -1
  ls aidlc-docs/inception/requirements/ | grep -oE '^cr-[0-9]{3}' | grep -oE '[0-9]{3}' | sort -n | tail -1
  git branch -a --list '*feature/cr-*' | grep -oE 'cr-[0-9]{3}' | grep -oE '[0-9]{3}' | sort -n | tail -1
  ```
  Zero-pad to three digits. Pick a short kebab-case slug from the request.
- `git checkout -b feature/cr-<NNN>-<slug>`. If the branch already exists locally or on `origin`, STOP and ask.

## 2. Analyse through graphify

Follow CLAUDE.md "Read the codebase through graphify first". Do not list directories or read files to "get context".

1. Freshness: `graph.json`'s `built_at_commit` must equal `git rev-parse HEAD`, else `make graph`. If graphify is missing, say so.
2. `graphify-out/GRAPH_REPORT.md`, once per session.
3. `graphify query "<the requested behaviour>"`, `graphify explain "<symbol>"`, `graphify path "<A>" "<B>"`, and `graphify affected "<symbol>"` for every symbol whose behaviour would change.
4. Read only the files (or line ranges) the graph points to. For RabbitMQ messages, HTTP between services and DB access, read `docs/contracts/` and the relevant ADR in `aidlc-docs/decisions/`.
5. UI change: read `docs/ux-ui-design-rules.md`.
6. Code change: read `docs/code-standards-rules.md`; the plan must respect it.

Describe the current behaviour from the code you read, not from assumption.

## 3. Questions first, if needed

If the request is ambiguous in a way that changes behaviour, ask the Creator numbered questions (with options and your recommendation) and STOP. Do not propose a solution on a guess. Record the questions in the CR doc and the audit (step 5).

## 4. Proposed solution

Write `aidlc-docs/construction/plans/cr-<NNN>-<slug>-design.md` in Vietnamese, following the shape of recent CR docs (`aidlc-docs/inception/requirements/cr-052-*.md`, `aidlc-docs/construction/plans/cr-052-*-design.md`):

- **Yêu cầu gốc (nguyên văn)**: the Creator's words, verbatim, plus every later answer.
- **Hiện trạng**: what the code does today, with `file:line` references.
- **Yêu cầu**: numbered functional requirements, acceptance criteria, out of scope.
- **Giải pháp đề xuất**: the design, and why. When there is a real choice, give 2-3 options with trade-offs and your recommendation.
- **Phạm vi**: services, files and symbols to change; contract/DB/migration changes; what `graphify affected` showed depends on them.
- **Kế hoạch thực hiện**: ordered steps the `/code` skill will follow, each concrete enough to implement without re-analysis (file, function, change), including the tests to add or update.
- **Kiểm tra**: tests to run, services to rebuild, what to check live.
- **Rủi ro**: data loss, breaking changes, anything left open.

## 5. Record

Add a `## CR-<NNN> — <tiêu đề>` entry at the end of `aidlc-docs/audit.md` in the existing format (Timestamp, User Input verbatim, AI Response, Impact Assessment, Artifacts Affected).

## 6. Report and stop

Tell the Creator: CR number, branch, a short summary of the solution (options and recommendation if any), the doc path, and that you wait for approval. Commit nothing.

## After the Creator approves

When the Creator answers "ok" / "approve" / "go" (or picks an option) to the proposal:

1. Apply any changes they asked for to the design doc and the audit (User Input verbatim).
2. Commit the doc and the audit on the CR branch: `CR-<NNN>: design approved — <subject>`, and push the branch.
3. Invoke the `code` skill with `CR-<NNN>` to implement the plan. Do not start coding yourself.

If they ask for changes instead, revise the doc, report again, and wait.
