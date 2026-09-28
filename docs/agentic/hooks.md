# Claude Code hooks

Deterministic enforcement that runs whether or not the agent "remembers" a rule. Registered in `.claude/settings.json` (committed, so every session and worktree gets them). Scripts live in `scripts/hooks/`; their tests are `scripts/hooks/test_hooks.py`, run by `make check` when `scripts/hooks/` changes and always by `make check-all`/CI.

| Event | Script | Does | Blocks with |
|---|---|---|---|
| PreToolUse `Bash` | `guard_bash.py` | Secret guard + merge gate (below) | exit 2; the reason is shown to the agent |
| PostToolUse `Edit\|MultiEdit\|Write` | `lint-edited.sh` | Lints **only the edited file** with its service's tool: `gofmt -l` (Go), the service `.venv`'s `ruff check` (Python), the service's `eslint` (TS/JS). No tests. | exit 2; findings go to the agent (the edit itself already happened) |
| Stop | `stop-check.sh` | Runs `make check` before the agent may end its turn | exit 2; last 150 lines of the check output go to the agent |

Hook scripts use only what a dev Mac already has: bash 3.2, `jq`, the system `python3` (3.9, stdlib only), `git`.

## Secret guard

The `Read(...)`/`Edit(...)` deny rules do not see the shell. `guard_bash.py` blocks Bash commands that mention:

- `.env` or `.env.window` as a path (`.env.example`, `.env.test`, `.envrc` are fine);
- anything under `secrets/` except `secrets/README.md`;
- `client_secret_*.json`;
- `docker compose config` (prints resolved secrets), unless `--services`/`--volumes`/`--profiles`/`--images`/`--networks`.

## Merge gate (D1)

Auto-merge into `main` stays (D1), but only for work `make check` has passed on.

**Marker.** When `make check` passes on a **clean** working tree and `HEAD` differs from its base, `scripts/check.sh` writes `.git/conceptflow/checked-trees/<tree hash of HEAD>` (in the common git dir, so all worktrees share it). The key is the tree, not the commit, because a merge of a branch that already contains `main` produces exactly that branch's tree. On `main` itself nothing is compared against, so no marker is written there.

**Rules** (the guard follows `git checkout`/`git switch` inside a chained command line to know the branch):

| Command | Allowed when |
|---|---|
| `git merge <ref>` on `main` | `<ref>` contains the current `main` **and** `<ref>`'s tree is marked. `--abort`/`--continue` always allowed |
| `git pull <remote> <other-branch>` on `main` | Never: use `git merge` so the gate applies. Plain `git pull` / `git pull origin main` is fine |
| `git push` that updates `main` (`git push` on main, `origin main`, `HEAD:main`, `--all`) | The pushed commit's tree is marked, or it equals `origin/main` (nothing new) |
| `git push … main` in the **same** command line as a merge/commit/pull on `main` | Never: run the push as its own command so the gate sees the real merged tree |

**Resulting CR finish flow:**

```bash
# on feature/cr-NNN-slug, everything committed
git fetch origin && git merge origin/main   # branch must contain the latest main
make check                                  # must pass on the clean tree → marker
git checkout main && git pull origin main && git merge --no-ff feature/cr-NNN-slug -m "..."
git push origin main                        # separate command
```

Review is **not** part of the gate yet (Creator decision, Phase 5). Phase 7 adds a review marker to the same gate.

## Stop hook details

- **Loop guard:** if the agent is already continuing because of this hook (`stop_hook_active`), the stop is allowed. A failure the agent cannot fix is reported once, not forever.
- **Cache:** a pass is stored per worktree (`.git/conceptflow/stop-check-pass`, or the worktree's own git dir) keyed by `HEAD` + `git diff HEAD` + untracked files. A turn that changed nothing costs ~0.05 s instead of re-running the check.
- **Cost:** `make check` only checks services changed vs `main`. A change to `Makefile`/`scripts/{check,setup,build}.sh` re-verifies everything (measured 41 s on a dev Mac).
- **Timeout:** 600 s in `settings.json`.

## Limits

- The guard matches command **text**. `sh -c "$CMD"`, variables, `eval`, or a tool that reads files on its own (`grep -r KEY .` sweeping `.env`) are not caught. The server-side ruleset `protect-main` is the backstop for force-push/deletion. Nothing server-side stops a push of an unchecked `main` (see the CI limitation in `branch-protection.md`).
- The same text matching gives false positives: a Bash command whose text merely mentions `.env` (a heredoc writing docs, a commit message) is blocked. Write such text with the Edit/Write tools or a script file.
- Hooks apply to Claude Code sessions only. A human running git in a terminal is not gated.
- Settings are read when a session starts. After changing hooks, start a new session (or review them via `/hooks`) before relying on them.

## Verification (2026-09-28)

- `python3 -m unittest discover -s scripts/hooks`: 16 tests pass (secret patterns incl. look-alikes; merge/push/pull gate against a throwaway repo).
- `lint-edited.sh`: reported an unformatted Go file, an unused Python import (ruff) and TS errors (eslint); clean files and non-service files exit 0.
- `stop-check.sh`: pass 41 s, cached re-run 0.05 s, `stop_hook_active` exits 0, an unformatted Go file blocks with the gofmt failure.
- Live in a session: `ls secrets/<nonexistent>` was blocked by the PreToolUse hook.
