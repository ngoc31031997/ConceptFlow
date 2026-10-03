---
name: cr
description: Start a new Change Request from the Creator's request - open the feature/cr-NNN-slug branch in its own git worktree, check dependencies on other open CRs, analyse the codebase through graphify, propose a solution (requirement + design) for the Creator to approve, then break it into line-level tasks for a second approval before /code. Use when the Creator types /cr or asks for a new feature or change of behaviour that is not already an open CR. Writes no production code.
argument-hint: "<the Creator's request, in their own words>"
model: claude-opus-5-5
effort: medium
---

# /cr

The Creator's request: `$ARGUMENTS`

This skill analyses and proposes. It writes no production code and commits nothing before the Creator approves. Answer the Creator in Vietnamese.

## 1. Own worktree and branch

Every CR is worked on in its own git worktree, so several CRs and fixes can run side by side (`.ai-dlc/aws-aidlc-rule-details/construction/git-branching.md`). The primary checkout (first line of `git worktree list`) stays on a clean `main`: never check out, stash, commit or edit anything there, even when it has uncommitted changes from another CR.

- CR number: `git fetch origin`, then one above the highest of
  ```bash
  git show origin/main:aidlc-docs/audit.md | grep -oE '^## CR-[0-9]{3}' | grep -oE '[0-9]{3}' | sort -n | tail -1
  git ls-tree --name-only origin/main aidlc-docs/inception/requirements/ | grep -oE 'cr-[0-9]{3}' | grep -oE '[0-9]{3}' | sort -n | tail -1
  git branch -a --list '*feature/cr-*' | grep -oE 'cr-[0-9]{3}' | grep -oE '[0-9]{3}' | sort -n | tail -1
  ```
  Zero-pad to three digits. Pick a short kebab-case slug from the request.
- `scripts/worktree.sh add feature/cr-<NNN>-<slug>` creates the branch from `origin/main` in `.claude/worktrees/`, builds its graphify graph and prints its path. If it reports that the branch exists, STOP and ask.
- Switch the session into it: `EnterWorktree` with `path` = that path. From here on every command, read and edit happens in the worktree; never edit files under the primary checkout.

## 2. Analyse through graphify

Follow CLAUDE.md "Read the codebase through graphify first". Do not list directories or read files to "get context".

1. Freshness: `graph.json`'s `built_at_commit` must equal `git rev-parse HEAD`, else `make graph`. If graphify is missing, say so.
2. `graphify-out/GRAPH_REPORT.md`, once per session.
3. `graphify query "<the requested behaviour>"`, `graphify explain "<symbol>"`, `graphify path "<A>" "<B>"`, and `graphify affected "<symbol>"` for every symbol whose behaviour would change.
4. Read only the files (or line ranges) the graph points to. For RabbitMQ messages, HTTP between services and DB access, read `docs/contracts/` and the relevant ADR in `aidlc-docs/decisions/`.
5. UI change: read `docs/ux-ui-design-rules.md`.
6. Code change: read `docs/code-standards-rules.md`; the plan must respect it.

Describe the current behaviour from the code you read, not from assumption.

## 3. Dependencies on open CRs and fixes

Other CRs and fixes may be open at the same time. Before designing, check whether this CR depends on or collides with any of them.

1. Open work: branches not yet in main, `git branch -a --no-merged origin/main --list '*feature/cr-*' '*fix/*' '*chore/*'`, and every other worktree in `scripts/worktree.sh list` (including the primary checkout if it is not on a clean `main`) with its uncommitted changes (`git -C <path> status --porcelain`).
2. For each: what it changes (`git diff --stat origin/main...<branch>`, plus the uncommitted files) and what it is for (its design doc: `git show <branch>:aidlc-docs/construction/plans/cr-<NNN>-<slug>-design.md`, or its `aidlc-docs/audit.md` entry).
3. Compare with the files, symbols, screens, contracts and tables this CR will touch (step 2, `graphify affected`). Classify each open item:
   - **Phụ thuộc** (depends on): this CR needs code, a contract, a migration or a behaviour that exists only on that branch, so that branch must be delivered first.
   - **Trùng phạm vi** (overlaps): both change the same feature, screen, file, contract or DB table, so they will conflict at merge or their behaviours clash.
   - **Độc lập** (independent).
4. Any dependency or overlap: STOP before writing the design. Tell the Creator which CR/fix, what exactly is shared (files, feature, contract), and numbered options with your recommendation, e.g. wait until it is delivered and branch from the new main / go ahead in parallel and resolve at merge / fold this request into that CR. Record it in the audit (step 6). Never branch from another open CR's branch without the Creator's explicit choice.
5. All independent: say so in one line in the report and in the design doc's **Phụ thuộc** section, listing what was checked.

## 4. Questions first, if needed

If the request is ambiguous in a way that changes behaviour, ask the Creator numbered questions (with options and your recommendation) and STOP. Do not propose a solution on a guess. Record the questions in the CR doc and the audit (step 6).

## 5. Proposed solution

Write `aidlc-docs/construction/plans/cr-<NNN>-<slug>-design.md` in Vietnamese, following the shape of recent CR docs (`aidlc-docs/inception/requirements/cr-052-*.md`, `aidlc-docs/construction/plans/cr-052-*-design.md`):

- **Yêu cầu gốc (nguyên văn)**: the Creator's words, verbatim, plus every later answer.
- **Hiện trạng**: what the code does today, with `file:line` references.
- **Yêu cầu**: numbered functional requirements, acceptance criteria, out of scope.
- **Giải pháp đề xuất**: the design, and why. When there is a real choice, give 2-3 options with trade-offs and your recommendation.
- **Phụ thuộc**: the open CRs/fixes checked in step 3 and the result for each (độc lập / phụ thuộc / trùng phạm vi), and what the Creator decided.
- **Phạm vi**: services, files and symbols to change; contract/DB/migration changes; what `graphify affected` showed depends on them.
- **Kế hoạch thực hiện**: ordered steps at file/function level (what changes where, which tests), short enough for the Creator to review the approach. The line-level task list for `/code` is written after this design is approved (see "After the Creator approves the design").
- **Kiểm tra**: tests to run, services to rebuild, what to check live.
- **Rủi ro**: data loss, breaking changes, anything left open.

## 6. Record

Add a `## CR-<NNN> — <tiêu đề>` entry at the end of `aidlc-docs/audit.md` in the existing format (Timestamp, User Input verbatim, AI Response, Impact Assessment, Artifacts Affected).

## 7. Report and stop

Tell the Creator: CR number, branch, worktree path, the dependency check result, a short summary of the solution (options and recommendation if any), the doc path, and that you wait for approval. Commit nothing.

## After the Creator approves the design

When the Creator answers "ok" / "approve" / "go" (or picks an option) to the proposal:

1. Apply any changes they asked for to the design doc and the audit (User Input verbatim).
2. In the CR's worktree, commit the doc and the audit on the CR branch: `CR-<NNN>: design approved — <subject>`, and push it with `git push -u origin feature/cr-<NNN>-<slug>`.
3. Break the plan into tasks (next section), report them and wait. Do not start coding yourself.

If they ask for changes instead, revise the doc, report again, and wait.

## Task breakdown

Write `aidlc-docs/construction/plans/cr-<NNN>-<slug>-tasks.md` in Vietnamese. It is the only plan `/code` follows, so `/code` must be able to implement each task without re-analysing the codebase.

- Read, at line level, only the files and symbols in the design's **Phạm vi** (most are already in this session's context; do not re-read them). Use `graphify affected` for any caller not yet checked.
- One task = one coherent change that can be tested on its own (usually one function or one closely related group of edits in one service), together with the tests that cover it. Order the tasks so each builds on the ones before it.
- Each task, as `### T<n> — <tiêu đề>` followed by a `- [ ] Xong` checkbox line:
  - **Service** and **File**: `file:line` or `file` + function/type name, for every place touched (new files marked as new).
  - **Thay đổi**: exactly what to change; signatures, constants, prompt or message wording, and a short code sketch where the change is not obvious.
  - **Test**: the tests to add or update (file, case names, what each asserts) and the command to run them.
  - **Xong khi**: the observable acceptance condition, traced to the design's FR numbers.
  - **Phụ thuộc**: earlier tasks it needs, or `—`.
- End with a **Sau cùng** section: the services to rebuild and the design's live checks (**Kiểm tra**).
- If the line-level reading shows the approved design is wrong or incomplete, do not patch it silently in the tasks: revise the design doc, tell the Creator what changed and why, and wait for the design to be re-approved before writing the tasks.

Add the task breakdown to the CR's audit entry. Report to the Creator: the task list (number, title, service, files, one line each), anything the breakdown revealed, the doc path, and that you wait for approval of the tasks. Commit nothing.

## After the Creator approves the tasks

1. Apply any changes they asked for to the tasks doc and the audit (User Input verbatim). If a change alters the solution, update the design doc too.
2. Commit the tasks doc (and any design change) with the audit on the CR branch: `CR-<NNN>: tasks approved — <n> tasks`, and push it.
3. Invoke the `code` skill with `CR-<NNN>`.

If they ask for changes instead, revise the tasks, report again, and wait.
