---
name: run-tasks
description: Read a tasks.html plan made by /analyze and have agents carry out its tasks in dependency order, each on the model and effort the Creator chose, recording each task's status back into the HTML. Run only when the Creator types /run-tasks.
disable-model-invocation: true
argument-hint: "<path to tasks.html> [T1=opus/high T3=sonnet/medium ...]"
---

# /run-tasks

Answer the Creator in Vietnamese. You are the coordinator: agents write the code, you schedule them, check their results and keep the plan file up to date.
The helper is `python3 .claude/skills/analyze/tasks.py` (called `tasks.py` below).

## 1. Load the plan

1. The first argument is the path to `tasks.html`. If missing, list `aidlc-docs/plans/*/tasks.html` and ask which one. If the Creator saved the page to `~/Downloads`, use that path, and copy it over the plan's own `tasks.html` first, after confirming with the Creator.
2. Any `T<n>=<model>/<effort>` pairs after the path (the line from **Copy lựa chọn**) override the file: apply each with `tasks.py set <file> <id> model <m>` and `... effort <e>`.
3. `tasks.py show <file>`, then show the Creator a table: id, title, model, effort, depends_on, status. Ask once to confirm before starting (the Creator may change models here too).

## 2. Work in the branch's worktree

The plan's `branch` is where the code goes. Per CLAUDE.md, never work on `main`: if the session is not already in that branch's worktree, run `scripts/worktree.sh add <branch>` and `EnterWorktree` into it. The plan file is used from the worktree's copy from then on.

## 3. Run the tasks

Repeat until `tasks.py next <file>` prints nothing:

1. Take the ready ids it prints. Run tasks in parallel only when their `files` lists do not overlap; otherwise run them one after another. Mark each `running` (`tasks.py set <file> <id> status running`).
2. Start one Agent per task (`subagent_type: general-purpose`, `model` and `effort` from the task). Its prompt must stand alone and contain:
   - the plan `title` and `summary`, and the task's goal, steps, files and acceptance criteria;
   - the absolute worktree path to work in, and that it must not commit, push, merge or touch `main`;
   - the repo rules it must follow: read `CLAUDE.md`, `docs/code-standards-rules.md` (and `docs/ux-ui-design-rules.md` for UI), use graphify before reading files, no fake/stub code, no silent skipping, tests fake the AI and never call a real model, clean up test data;
   - that if it is blocked or a requirement is unclear it must stop and report, not guess;
   - to finish with a report: files changed, tests run with their result, anything not done.
3. When an agent returns, check its report against the acceptance criteria and look at `git diff` for the task's files. Then set `done`, or `failed` with `tasks.py set <file> <id> note "<short reason>"`.
4. If a task fails or the agent reports a blocker, STOP the loop: do not start its dependants, tell the Creator what happened and ask how to go on (retry, retry on a stronger model, change the task, skip). Never mark a task `done` that did not meet its criteria.

Do not edit code yourself to finish an agent's task; send it back to an agent, or ask the Creator.

## 4. Finish

1. When every task is `done`, run the affected services' tests once more and rebuild the changed services with `scripts/worktree.sh rebuild <service>...` (CLAUDE.md Docker policy), confirming they are healthy.
2. Report to the Creator: each task with its status, model used and a one-line result; tests run and results; services rebuilt; test data cleaned up; anything not done. Link `tasks.html` with its absolute path.
3. Do not commit. The Creator reviews and then runs `/deliver`.
