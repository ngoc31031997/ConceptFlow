# Autonomy policy

What a Claude Code agent may do on its own in this repository, what needs a human, and what is refused outright. Every row says **how** it is enforced. A rule that exists only as prose in `CLAUDE.md` is marked as such, because the spec (§4.2) does not count "Claude, remember to…" as enforcement.

Decisions referenced (D1–D10) are recorded in [`implementation-audit.md`](implementation-audit.md).

## Levels

| Level | Meaning | Mechanism |
|---|---|---|
| **Agent** | Runs without a prompt | `permissions.allow` in `.claude/settings.json`, or the session's permission mode |
| **Ask** | Claude Code prompts the human every time, even when a broader allow rule matches | `permissions.ask` |
| **Deny** | Refused. No prompt, and allow rules cannot override it | `permissions.deny` |
| **Human** | The agent does not do it at all. The Creator does it by hand | Documentation plus no granted access |

Precedence in Claude Code is **deny → ask → allow**, across every settings file (user, project, local). A personal `settings.local.json` can widen `allow` but cannot lift a project `deny`.

## Policy

| Action | Level | Enforced by |
|---|---|---|
| Read / search source code | Agent | Default |
| Modify application code | Agent | Default (edits prompt or not according to the session's permission mode) |
| `make setup`, `make build`, `make check`, `make check-all` | Agent | `allow` |
| Create / switch branches, `git add`, `git commit`, `git fetch`, `git pull` | Agent | `allow` |
| `git merge` (including into `main`, D1) | Agent | `allow`. **Gate not enforced yet**: the "`make check` + review passed" hook is Phase 5 |
| `git push` to `origin` (plain, no flags that rewrite or delete) | Agent | `allow` for `git push`, `git push origin *`, `git push -u origin *` |
| Discard uncommitted work: `git checkout -- …`, `git checkout .`, `git restore`, `git reset --hard`, `git clean`, `git branch -D` | Ask | `ask` |
| `docker exec` (can print container env, i.e. secrets) | Ask | `ask` |
| Data migration (`scripts/migrate-authoring-data.sh`) | Ask | `ask` |
| Edit CI configuration (`.github/**`) | Ask | `ask` on `Edit(/.github/**)` |
| Force push (`--force`, `--force-with-lease`, `-f`, `+refspec`) | Deny | `deny` |
| Delete remote refs (`--delete`, `-d`, `:branch`), `--mirror` | Deny | `deny` |
| `rm -rf` / `rm -fr` / `rm -Rf` | Deny | `deny` |
| Destroy Docker data: `docker compose down -v`/`--volumes`, `docker volume rm`, `docker volume prune`, `docker system prune` | Deny | `deny` |
| Read or write secrets: `.env`, `.env.window`, `secrets/**`, `client_secret_*.json` | Deny | `deny` on `Read(…)` and `Edit(…)` |
| Change this policy or `.claude/settings.json` | Human approval | Prose only (`CLAUDE.md` commit policy). Changes go through a reviewed commit |
| Enable/alter the GitHub ruleset `protect-main` | Human | Agents have no admin access. See [`branch-protection.md`](branch-protection.md) |
| Production deployment, production data mutation | Human | Not applicable yet: the repo has no production environment or deploy credentials |

## Scope of the settings files

| File | Committed | Holds |
|---|---|---|
| `.claude/settings.json` | Yes | Everything in the table above. Shared by every agent session and every worktree |
| `.claude/settings.local.json` | No (globally git-ignored) | Personal convenience allows only, e.g. `go test *`, `docker compose *`. Must not contain secrets or one-off commands with payloads. Cannot weaken a project `deny` |

## Known gaps

These are real holes, not oversights. Each has a planned fix.

1. **`Read` deny does not cover the shell.** `Read(**/.env)` stops the Read, Grep and Glob tools, but `cat .env`, `grep KEY .env` or `docker compose config` in Bash are not matched. Fix: a PreToolUse hook on Bash that rejects commands referencing the secret paths (Phase 5).
2. **Bash patterns are prefix/glob matches, not a parser.** Wrapped or chained commands (`sh -c "git push -f …"`, `cd x && rm -rf y`, variables) can slip past a pattern. Deny rules are a guard against accidents, not against a determined bypass. The server-side ruleset (`protect-main`) is the backstop for force-push/deletion of `main`.
3. **Merge gate (D1) is not enforced yet.** An agent can merge a red branch into `main` today. Fix: Phase 5 hook blocking `git merge` into `main` and `git push origin main` unless a "check + review passed" marker matches `HEAD`.
4. **`protect-main` ruleset** must be created by the Creator. Until then nothing server-side stops a force push from a human or another tool.

## Verification

Checked on 2026-09-28 in a Claude Code session (VS Code extension) opened at the repo root, with harmless targets:

| Probe | Result |
|---|---|
| `git push --force no-such-remote no-such-branch` | Denied by rule |
| `rm -rf .permtest/nothing-here` | Denied by rule |
| `docker volume rm no-such-volume-xyz` | Denied by rule |
| Read tool on a dummy `.permtest/.env` | Denied by rule |
| `git push origin :no-such-branch-xyz` | Blocked, but by the session's auto-mode classifier before the rule was reached, so the `:refspec` deny rule itself is **not** confirmed |
| `Edit(…)` secret denies | Added after the probes. **Not probed yet** |

To re-check after changing the settings, open a new session at the repo root and repeat the probes against non-existent remotes, branches, volumes and a dummy `.env`, never against real ones.
