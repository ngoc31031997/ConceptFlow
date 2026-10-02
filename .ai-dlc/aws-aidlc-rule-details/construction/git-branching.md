# Git Branching Strategy

**Purpose**: Keep concurrent work from different agents/sessions isolated, so several Change Requests (CRs) and fixes can be implemented at the same time against this repo.

## Rule: one branch, one worktree

- **Every Change Request gets its own feature branch** `feature/cr-<NNN>-<short-slug>` (e.g. `feature/cr-025-authoring-pipeline`); a bug fix gets `fix/<short-slug>`; other work not tied to a numbered CR gets `chore/<short-slug>` or `feature/<short-slug>`.
- **Every such branch lives in its own git worktree**, created from `origin/main` with `scripts/worktree.sh add <branch>` under `.claude/worktrees/` (git-ignored). A Claude Code session switches into it with `EnterWorktree` (`path`) and does all its reading, editing, testing and committing there.
- **The primary checkout** (the first entry of `git worktree list`) stays on a clean `main`. Nobody works in it: do not check out a branch, stash, commit or edit files there. It owns the running Docker stack (`.env`, `secrets/`, `data/`, volumes) and is where `/deliver` merges into `main`.
- Never implement a CR directly on `main`/`master`, and never on a branch that belongs to a different, still-open CR or fix.
- Never branch a CR from another open CR's branch unless the Creator explicitly chose to.
- Do not merge, delete or remove the worktree of another agent's branch without being asked.
- This applies at every workflow stage (Construction, quick fixes, patches); see the root `CLAUDE.md` Git commit policy for commit timing (commit only after explicit stage approval).

## Dependencies between open CRs

Before designing a new CR, `/cr` compares it with every open branch (`git branch -a --no-merged origin/main`) and every other worktree's uncommitted changes, and classifies each as:

- **depends on**: the new CR needs code, a contract, a migration or behaviour that exists only on the other branch, so that branch must be delivered first;
- **overlaps**: both change the same feature, screen, file, contract or DB table;
- **independent**.

Any dependency or overlap is reported to the Creator, with options, before the design is written. The result is recorded in the design doc's **Phụ thuộc** section.

## Docker from a worktree

There is one live stack. `scripts/worktree.sh rebuild <service>...`, run in a worktree, builds the images from the worktree's code and restarts them in the primary checkout (the compose project name is pinned in `docker-compose.yml`, so the image names match). Do not run `docker compose up` inside a worktree: it has no `.env`, secrets or data. A rebuild replaces whatever another open branch last deployed to the same service, so reports name the branch that is live.

## Lifecycle

1. `/cr` or `/fix`: `scripts/worktree.sh add <branch>`, then `EnterWorktree`.
2. `/code`: implements, tests and rebuilds from that worktree.
3. `/deliver`: commits in the worktree; `deliver.sh` merges the branch into `main` in the primary checkout, rebuilds from `main`, and removes the worktree (the branch is kept); the session leaves it with `ExitWorktree` (`keep`).

`aidlc-docs/audit.md` is append-only and merged with git's union driver (`.gitattributes`), so parallel CRs appending entries do not conflict. Any other conflict stops `/deliver` for the Creator to decide.

## Why

Several agents can be working in this repo concurrently. A shared working directory lets one agent's uncommitted, unapproved work leak into another's commits, tests or Docker rebuilds; separate worktrees keep each CR's files apart while sharing one git history and one live stack.
