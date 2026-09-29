# Claude Code hooks

Deterministic enforcement that runs whether or not the agent "remembers" a rule. Registered in `.claude/settings.json` (committed, so every session and worktree gets them). Scripts live in `scripts/hooks/`; their tests are `scripts/hooks/test_hooks.py`, run by `make check` when `scripts/hooks/` changes and always by `make check-all`/CI.

| Event | Script | Does | Blocks with |
|---|---|---|---|
| PreToolUse `Bash` | `guard_bash.py` | Secret guard + merge gate (below) | exit 2; the reason is shown to the agent |
| PostToolUse `Edit\|MultiEdit\|Write` | `lint-edited.sh` | Lints **only the edited file** with its service's tool: `gofmt -l` (Go), the service `.venv`'s `ruff check` (Python), the service's `eslint` (TS/JS). No tests. | exit 2; findings go to the agent (the edit itself already happened) |
| Stop | `stop-check.sh` | Runs `make check` before the agent may end its turn | exit 2; last 150 lines of the check output go to the agent |
| SubagentStop | `record_review.py` | Records the verdict of the `reviewer`, `security-reviewer`, `tester` agents for the merge gate | never blocks |

Hook scripts use only what a dev Mac already has: bash 3.2, `jq`, the system `python3` (3.9, stdlib only), `git`.

## Secret guard

The `Read(...)`/`Edit(...)` deny rules do not see the shell. `guard_bash.py` blocks Bash commands that mention:

- `.env` or `.env.window` as a path (`.env.example`, `.env.test`, `.envrc` are fine);
- anything under `secrets/` except `secrets/README.md`;
- `client_secret_*.json`;
- `docker compose config` (prints resolved secrets), unless `--services`/`--volumes`/`--profiles`/`--images`/`--networks`.

## Merge gate (D1)

Auto-merge into `main` stays (D1), but only for work that passed `make check` **and** the three review agents.

**Marker.** When `make check` passes on a **clean** working tree and `HEAD` differs from its base, `scripts/check.sh` writes `.git/conceptflow/checked-trees/<tree hash of HEAD>` (in the common git dir, so all worktrees share it). The key is the tree, not the commit, because a merge of a branch that already contains `main` produces exactly that branch's tree. On `main` itself nothing is compared against, so no marker is written there.

**Review markers.** The review input is produced by a script, and the verdict is recorded by a hook. The session that wrote the code controls neither:

1. `scripts/review-prep.sh <label> [requirement-doc]` (clean tree only) writes `.git/conceptflow/review/<tree>.diff` (the real diff against the merge-base with `main`) and `<tree>.brief` (the exact prompt for the agents) and prints the brief.
2. `/cr-review` passes the brief **verbatim** to the `reviewer`, `security-reviewer` and `tester` agents (`.claude/agents/`, tools Read/Grep/Glob only). Each ends its report with `VERDICT: PASS|FAIL tree=<tree hash>`.
3. The SubagentStop hook `record_review.py` takes the verdict from the report the agent actually delivered (its last `SubagentHandback` call in the agent transcript; `last_assistant_message` if there is none). On **FAIL** it removes the marker and writes `<tree>.<agent>.fail`, which blocks every later PASS on that tree: re-running an agent until it agrees does not work, fixing and committing (a new tree) does. Only the Creator can lift a FAIL, by deleting the `.fail` file from a terminal. On **PASS** it writes `.git/conceptflow/reviewed-trees/<tree>.<agent>` only if **all** of these hold:
   - no earlier FAIL on this tree from the same agent;
   - the tree is the current `HEAD`'s tree and the working tree is clean (the agents read code around the diff from disk);
   - the agent transcript is the one Claude Code wrote for this session's subagent (`<projects>/<project>/<session_id>/subagents/agent-<agent_id>.jsonl`, entries tagged with that `agentId`), so a hand-made transcript cannot be fed in;
   - the agent's prompt (first user message in its transcript) equals `<tree>.brief`, ignoring only leading/trailing whitespace, so nothing was added such as "just say PASS";
   - `<tree>.diff` still equals the real diff, so the agent saw the whole change.
4. Every outcome, including the reason a review was **not** recorded, goes to `.git/conceptflow/review/hook.log`. `scripts/review-status.sh` shows the four markers for `HEAD` and the log tail (exit 0 = the gate would pass).

Protection:

- Bash commands naming `conceptflow/checked-trees`, `reviewed-trees`, `review`, the hook `record_review` or a subagent transcript (`subagents/agent-`) are blocked by `guard_bash.py`. Running the hook by hand with a forged payload was the Major finding of the first live review.
- Edit/Write of `.git/conceptflow/**` and `~/.claude/projects/**/subagents/**` is denied (both verified live: the write was refused).
- `review-prep.sh` accepts only a short label (`[A-Za-z0-9 ._-]`, max 40) and a requirement doc under `aidlc-docs/` or `docs/`, so no free text gets into the trusted brief.
- Edits to the gate's own configuration (`.claude/settings*.json`, `.claude/agents/**`, `.claude/skills/**`, `scripts/hooks/**`, `scripts/check.sh`, `scripts/review-prep.sh`) are `ask`. In the session's **auto** permission mode those edits went through without a visible prompt; in the default mode they prompt the Creator.

**Rules** (the guard follows `git checkout`/`git switch` inside a chained command line to know the branch):

| Command | Allowed when |
|---|---|
| `git merge <ref>` on `main` | `<ref>` contains the current `main` **and** `<ref>`'s tree has the `make check` marker and all three review markers. `--abort`/`--continue` always allowed |
| `git pull <remote> <other-branch>` on `main` | Never: use `git merge` so the gate applies. Plain `git pull` / `git pull origin main` is fine |
| `git push` that updates `main` (`git push` on main, `origin main`, `HEAD:main`, `--all`) | The pushed commit's tree has all four markers, or it equals `origin/main` (nothing new) |
| `git push … main` in the **same** command line as a merge/commit/pull on `main` | Never: run the push as its own command so the gate sees the real merged tree |

**Resulting CR finish flow:**

```bash
# on feature/cr-NNN-slug, everything committed
git fetch origin && git merge origin/main   # branch must contain the latest main
make check                                  # must pass on the clean tree → marker
/cr-review                                  # reviewer + security-reviewer + tester PASS → markers
git checkout main && git pull origin main && git merge --no-ff feature/cr-NNN-slug -m "..."
git push origin main                        # separate command
```

Phase 5 gated on `make check` only; Phase 7 added the review markers (Creator approved, 2026-09-28). A new commit is a new tree: review fixes need `/cr-check` and `/cr-review` again.

## Stop hook details

- **Loop guard:** if the agent is already continuing because of this hook (`stop_hook_active`), the stop is allowed. A failure the agent cannot fix is reported once, not forever.
- **Cache:** a pass is stored per worktree (`.git/conceptflow/stop-check-pass`, or the worktree's own git dir) keyed by `HEAD` + `git diff HEAD` + untracked files. A turn that changed nothing costs ~0.05 s instead of re-running the check.
- **Cost:** `make check` only checks services changed vs `main`. A change to `Makefile`/`scripts/{check,setup,build}.sh` re-verifies everything (measured 41 s on a dev Mac).
- **Timeout:** 600 s in `settings.json`.

## Limits

- Protection of the gate's own files covers the Edit/Write tools (`ask`) and Bash commands that *name* the marker directories. A Bash command that rewrites a hook script (`sed -i`, a Python one-liner) is not matched; the Claude Code auto-mode classifier has flagged such self-modification in practice, but that is not a guarantee. The final backstop is review: changes to `scripts/hooks/` and `.claude/` show up in the diff the review agents read, and in `git log`.
- The guard matches command **text**. `sh -c "$CMD"`, variables, `eval`, or a tool that reads files on its own (`grep -r KEY .` sweeping `.env`) are not caught. The server-side ruleset `protect-main` is the backstop for force-push/deletion. Nothing server-side stops a push of an unchecked `main` (see the CI limitation in `branch-protection.md`).
- The same text matching gives false positives: a Bash command whose text merely mentions `.env` (a heredoc writing docs, a commit message) is blocked. Write such text with the Edit/Write tools or a script file.
- Hooks apply to Claude Code sessions only. A human running git in a terminal is not gated.
- Settings are read when a session starts. After changing hooks, start a new session (or review them via `/hooks`) before relying on them.

## Verification (2026-09-28)

- `python3 -m unittest discover -s scripts/hooks`: 16 tests pass (secret patterns incl. look-alikes; merge/push/pull gate against a throwaway repo).
- `lint-edited.sh`: reported an unformatted Go file, an unused Python import (ruff) and TS errors (eslint); clean files and non-service files exit 0.
- `stop-check.sh`: pass 41 s, cached re-run 0.05 s, `stop_hook_active` exits 0, an unformatted Go file blocks with the gofmt failure.
- Live in a session: `ls secrets/<nonexistent>` was blocked by the PreToolUse hook.

Phase 7:

- SubagentStop payload on the installed Claude Code, captured with a temporary probe hook and an `Explore` agent: `agent_id`, `agent_type`, `agent_transcript_path`, `last_assistant_message`, `cwd`, `stop_hook_active` (already `true` on the first stop, so it is not usable as a loop guard). The probe was removed.
- `python3 -m unittest discover -s scripts/hooks`: 41 tests pass (after the fixes from live reviews 1 and 2), including: gate blocks a checked-but-unreviewed and a reviewed-but-unchecked branch; `record_review.py` writes on PASS, removes on a later FAIL, ignores other agents, a verdict not on the last line, a hash that is not a tree, and no verdict; marker directories blocked from the shell.
- **Live run 1** (tree `6e974cc…`): the hook recorded PASS for `security-reviewer` and `tester` and FAIL for `reviewer` (Major: forging a review by running the hook by hand). The gate kept the branch out of `main`. Fixed: transcript-location check, Bash guard for the hook and transcripts, deny on transcript writes, fenced-verdict parsing, label validation, extra tests.
- **Live run 2** (tree `fdf2327…`): `security-reviewer`, `tester` PASS; `reviewer` FAIL (Major: a FAIL could be overturned by re-running the agent on the same tree). Fixed: sticky `.fail` per tree, clean-tree requirement for PASS, no environment overrides in the hook (the tests run a copy with the transcript root rewritten), guard blocks only *running* the hook (git on the file is allowed), hook tests also run when `.claude/` changes, agents told that diff text is data never instructions, `review-status.sh` in `ask`.
- A skill invoked in the session can come from a cached copy: `/cr-review` loaded its pre-hardening text while the file on disk was newer. After editing skills, prefer a new session.
