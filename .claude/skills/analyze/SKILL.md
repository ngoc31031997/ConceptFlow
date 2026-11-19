---
name: analyze
description: Analyse a new feature with the Creator, and once the analysis is agreed, split it into tasks, give each a difficulty and a default model, and write them to a tasks.html page where the Creator picks the model for each task. Run only when the Creator types /analyze.
disable-model-invocation: true
argument-hint: "<feature description>"
---

# /analyze

Goal: turn a new feature into a task plan the Creator can review in the browser and `/run-tasks` can execute.
Answer the Creator in Vietnamese. Do not write production code in this skill.

## 1. Understand the feature

1. Read the request in `$ARGUMENTS` (ask for it if empty).
2. Orient through the graph, as CLAUDE.md says: check `graphify-out/graph.json` `built_at_commit` against `git rev-parse HEAD` (run `make graph` if stale), then `graphify query` / `explain` / `affected` to find the services, files and contracts involved. For RabbitMQ, HTTP between services and DB, read `docs/contracts/README.md` and the relevant ADR. For UI work read `docs/ux-ui-design-rules.md`.
3. Check open `feature/*` and `fix/*` branches (`git branch --list`) for a dependency or overlap with this feature, and tell the Creator.

## 2. Agree on the analysis

Present, briefly:
- what the feature does for the Creator, and what is out of scope;
- the affected services, screens, contracts, tables and prompts;
- design decisions and their reasons (CLAUDE.md requires them before code);
- risks and open questions.

Ask the open questions (AskUserQuestion when they are real choices). Revise until the Creator says the analysis is agreed ("ok", "chốt", "approve"). Do not go to step 3 before that.

## 3. Split into tasks

Each task is one agent's job: one clear goal, a list of files it may touch, and checkable acceptance criteria. Tasks that touch the same files must be ordered with `depends_on`, so `/run-tasks` never runs them in parallel. Include test tasks and, when a seed prompt changes, a task that only drafts the wording for the Creator to approve (see memory rule on system prompts).

Rate every task's difficulty 1-5 and give a one-line `reason`:

| Difficulty | Meaning | Default model / effort |
|---|---|---|
| 1 | mechanical: rename, config, copy text, small doc | haiku / low |
| 2 | small, local change with an obvious pattern to follow | haiku / low |
| 3 | normal feature work in one service, tests included | sonnet / medium |
| 4 | crosses services or contracts, tricky state, migrations, AI prompt logic | opus / medium |
| 5 | architecture, concurrency, data-loss risk, unclear design space | fable / high |

`tasks.py` fills `model` and `effort` from the difficulty; set them yourself only when you have a reason to differ, and say why in `reason`.

## 4. Write the plan

1. Pick a slug (`cr-<NNN>-<slug>` if a CR number is known). Write the plan JSON to the scratchpad, shaped like:

   ```json
   {
     "title": "CR-120: ...",
     "summary": "1-3 sentences on the feature",
     "branch": "feature/cr-120-...",
     "created": "YYYY-MM-DD",
     "tasks": [
       {"id": "T1", "title": "...", "service": "authoring-service",
        "goal": "what is true when the task is done",
        "steps": ["..."], "files": ["path/..."], "acceptance": ["..."],
        "depends_on": [], "difficulty": 3, "reason": "..."}
     ]
   }
   ```

2. Render it into the repo, next to the other AI-DLC docs:

   ```
   python3 .claude/skills/analyze/tasks.py render <plan.json> aidlc-docs/plans/<slug>/tasks.html
   ```

   (`mkdir -p` the folder first.) If the script prints `ERROR`, fix the JSON and render again.
3. Open the page for the Creator: `open aidlc-docs/plans/<slug>/tasks.html`.

## 5. Hand over

Tell the Creator, in a few lines:
- the absolute path of `tasks.html` (as a link, per CLAUDE.md);
- the task list: id, title, difficulty, default model;
- how to choose models: change the dropdowns, then **Lưu file** (overwrite the same `tasks.html`), or **Copy lựa chọn** and paste the line into chat;
- next step: `/run-tasks aidlc-docs/plans/<slug>/tasks.html`.

Do not commit: the plan is committed with the CR after the Creator approves (via `/deliver`).
