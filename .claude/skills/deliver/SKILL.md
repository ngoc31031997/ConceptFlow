---
name: deliver
description: After the Creator approves the code, commit the current branch, push it, merge it into main, pull main, refresh the graphify code graph for the new main and rebuild the changed Docker services. Run only when the Creator types /deliver.
disable-model-invocation: true
model: sonnet
effort: medium
argument-hint: "[optional commit message hint]"
allowed-tools: Bash(git status*), Bash(git diff*), Bash(git log*), Bash(git branch*), Bash(git rev-parse*), Bash(git add*), Bash(git commit*), Bash(.claude/skills/deliver/deliver.sh*), ExitWorktree
---

# /deliver

The Creator typing `/deliver` is the approval. Do exactly the steps below, in order, and nothing else.
Do not edit code, do not fix anything, and do not work around errors: when a step fails, stop and report.

## 1. Check the branch

Run this in the branch's own worktree (the session is normally already in it, after `/cr`, `/code` or `/fix`). Run `git rev-parse --abbrev-ref HEAD` and `git status -s`.

- If the branch is `main`, STOP. Tell the Creator: work must be on its own branch (`feature/cr-<NNN>-<slug>` or `fix/...` / `chore/...`), and ask which branch to use.
- If there is nothing to commit, go straight to step 3. The branch may hold commits that are not in main yet.

## 2. Commit

1. Read `git status -s` and `git diff HEAD --stat`. If a file looks like a secret or does not belong to this change (`.env*`, `secrets/`, `client_secret_*.json`, large media files, stray logs), STOP and ask the Creator.
2. Read `git diff HEAD` and `git log --oneline -5`, so you can match the repo's commit style.
3. `git add -A`
4. Commit with a message in this shape:

   ```
   <subject: under 72 characters, in the repo's style (e.g. `CR-052: ...`, `Fix: ...`), naming the feature or fix>

   <brief: 1-3 sentences on what the feature/fix does for the Creator and why>

   Changes:
   - <area or service>: <what changed in it>
   - <area or service>: <what changed in it>
   - ...

   Co-Authored-By: <the attribution line the harness asks for>
   ```

   - The brief describes the feature as a whole. The `Changes:` list gives the details, one bullet per concrete change (a screen, an endpoint, a service, a test, a doc), saying what was changed or fixed in it.
   - Write only about changes that are really in the diff. Do not list files one by one unless each file is its own change.
   - If `$ARGUMENTS` is not empty, use it as the basis for the subject and the brief.

## 3. Deliver

Run the script with the branch name. Give the Bash call a 600000 ms timeout, because Docker builds are slow:

```
.claude/skills/deliver/deliver.sh <branch>
```

The script pushes the branch, fast-forwards `main` in the primary checkout, merges the branch into it, pushes `main`, refreshes the graphify graph (`scripts/graph.sh build`, so `graphify-out/graph.json` is built at the new `main` HEAD), rebuilds and restarts only the services whose code changed, waits for them to be healthy, and removes the branch's worktree.

When the script prints `continue from the primary checkout`, call `ExitWorktree` with `action: "keep"` so the session returns to the primary checkout (the worktree directory is already gone; the branch is kept).

## 4. Report

Answer the Creator in Vietnamese, in a few lines:

- the commit hash and subject;
- whether the merge into `main` and the push succeeded;
- whether the graphify graph was refreshed (the commit it was built at);
- which services were rebuilt and whether they are healthy, or "no rebuild needed";
- whether the worktree was removed.

If the script exits non-zero, give its error lines and the likely cause, and STOP. Do not retry, and do not resolve conflicts:

| Exit | Meaning |
|---|---|
| 1 | bad state: the primary checkout is not on `main` or has uncommitted changes (another CR still worked on there, not in a worktree) |
| 2 | push or pull failed |
| 3 | merge conflict (already aborted, `main` unchanged) |
| 4 | `docker compose build/up` failed |
| 5 | a service is not healthy (the last log lines are printed) |
| 6 | merge, push and rebuild succeeded, but the graphify refresh failed: tell the Creator to run `make graph` |

If the output has `WARN: docker-compose.yml changed`, ask the Creator whether to recreate the whole stack.
