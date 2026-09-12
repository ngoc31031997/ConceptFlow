# Git Branching Strategy

**Purpose**: Keep concurrent work from different agents/sessions isolated, since multiple Change Requests (CRs) may be implemented at the same time by different agents against this repo.

## Rule

- **Every Change Request gets its own feature branch.** Before writing any implementation code for a CR, check the current branch:
  - If already on a feature branch scoped to this CR, continue on it.
  - If on `main`/`master` (or any other CR's branch), create a new branch first: `feature/cr-<NNN>-<short-slug>` (e.g. `feature/cr-025-authoring-pipeline`). If the work isn't tied to a numbered CR, use `feature/<short-slug>`.
- Never implement a new CR's code directly on `main`/`master`.
- Never implement a new CR's code on a branch that belongs to a different, still-open CR — branches must not mix unrelated CRs, since another agent may be mid-work on that branch.
- Do not merge or delete another agent's feature branch without being asked.
- This applies regardless of workflow stage (Construction, quick fixes, patches) — see also the root `CLAUDE.md` Git commit policy for commit-timing rules (commit only after explicit stage approval) once the branch exists.

## Why

Several agents can be working in this repo concurrently. Sharing a branch across unrelated CRs risks one agent's uncommitted/unapproved work colliding with another's, or a rebuild/restart triggered by one CR picking up another's half-finished changes.
