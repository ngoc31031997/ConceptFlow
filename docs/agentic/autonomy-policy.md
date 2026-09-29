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
| `git merge` (including into `main`, D1) | Agent, gated | `allow`, plus the merge gate hook: into `main` only when the branch's tree passed `make check` and the three review agents. See [`hooks.md`](hooks.md) |
| `git push` to `origin` (plain, no flags that rewrite or delete) | Agent | `allow` for `git push`, `git push origin *`, `git push -u origin *`. A push updating `main` passes the merge gate |
| Discard uncommitted work: `git checkout -- …`, `git checkout .`, `git restore`, `git reset --hard`, `git clean`, `git branch -D` | Ask | `ask` |
| `docker exec` (can print container env, i.e. secrets) | Ask | `ask` |
| Data migration (`scripts/migrate-authoring-data.sh`) | Ask | `ask` |
| Edit CI configuration (`.github/**`) | Ask | `ask` on `Edit(/.github/**)` |
| Force push (`--force`, `--force-with-lease`, `-f`, `+refspec`) | Deny | `deny` |
| Delete remote refs (`--delete`, `-d`, `:branch`), `--mirror` | Deny | `deny` |
| `rm -rf` / `rm -fr` / `rm -Rf` | Deny | `deny` |
| Destroy Docker data: `docker compose down -v`/`--volumes`, `docker volume rm`, `docker volume prune`, `docker system prune` | Deny | `deny` |
| Read or write secrets: `.env`, `.env.window`, `secrets/**`, `client_secret_*.json` | Deny | `deny` on `Read(…)` and `Edit(…)`; shell access blocked by the PreToolUse guard hook |
| Finish a turn with `make check` failing | Deny (once) | Stop hook |
| Change the gate's configuration: `.claude/settings*.json`, `.claude/agents/**`, `.claude/skills/**`, `scripts/hooks/**`, `scripts/check.sh`, `scripts/review-prep.sh` | Ask | `ask` on `Edit(…)` for each; changes then go through the reviewed, gated merge |
| Write or forge Claude Code subagent transcripts, or run the review hook by hand | Deny | `deny` on `Edit(~/.claude/projects/**/subagents/**)`; Bash naming them blocked by the guard hook |
| Write merge-gate markers or review input (`.git/conceptflow/**`) | Deny | `deny` on `Edit(/.git/conceptflow/**)`; Bash naming the directories blocked by the guard hook. Only `make check`, `review-prep.sh` and the SubagentStop hook write there |
| Enable/alter the GitHub ruleset `protect-main` | Human | Agents have no admin access. See [`branch-protection.md`](branch-protection.md) |
| Production deployment, production data mutation | Human | Not applicable yet: the repo has no production environment or deploy credentials |

## Scope of the settings files

| File | Committed | Holds |
|---|---|---|
| `.claude/settings.json` | Yes | Everything in the table above. Shared by every agent session and every worktree |
| `.claude/settings.local.json` | No (globally git-ignored) | Personal convenience allows only, e.g. `go test *`, `docker compose *`. Must not contain secrets or one-off commands with payloads. Cannot weaken a project `deny` |

## Known gaps

These are real holes, not oversights. Each has a planned fix.

1. **Shell access to secrets is matched by text.** Since Phase 5 the guard hook blocks commands that name the secret paths or run `docker compose config`, but a command that reads them without naming them (`grep -r KEY .`, a script, `sh -c "$VAR"`) is not caught. The reverse also happens: a Bash command whose *text* mentions `.env` (e.g. a heredoc editing this doc) is blocked; use the Edit/Write tools for such text.
2. **Bash patterns are prefix/glob matches, not a parser.** Wrapped or chained commands (`sh -c "git push -f …"`, `cd x && rm -rf y`, variables) can slip past a pattern. Deny rules are a guard against accidents, not against a determined bypass. The server-side ruleset (`protect-main`) is the backstop for force-push/deletion of `main`.
3. **Gate files can still be rewritten through Bash.** Since Phase 7 the merge gate requires `make check` plus verified PASS verdicts from the `reviewer`, `security-reviewer` and `tester` agents on the same tree, with the brief and diff produced by a script and checked by the hook ([`hooks.md`](hooks.md)). Edit/Write of the gate's files asks the Creator, but a Bash command rewriting them is not matched. Such a change still appears in the reviewed diff and in `git log`.
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
