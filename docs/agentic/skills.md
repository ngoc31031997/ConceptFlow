# Claude Code skills (CR workflow)

Project skills live in `.claude/skills/<name>/SKILL.md` and are invoked as `/<name>` (by the Creator, or by the agent when the description matches). They are named after **workflow steps** of a Change Request. **Roles** (solution architect, reviewer, security reviewer, tester) are separate read-only agents in `.claude/agents/` (Phase 7), which the skills call. Creator decision, 2026-09-28.

| Skill | When | Does | Backed by |
|---|---|---|---|
| `/cr-start <slug> [NNN]` | A new change request | Clean-tree check, sync `main`, next CR number (audit log, requirement docs, existing branches), `feature/cr-NNN-slug`, open AI-DLC Requirements Analysis, then wait for approval | git; `.ai-dlc/` rules |
| `/cr-check [all]` | Before saying "done" | `make check` (or `make check-all`), failures as a table with cause and next action; separates branch-caused from pre-existing failures with evidence | `scripts/check.sh` |
| `/rebuild [svc…]` | After service code changes (CLAUDE.md Docker rebuild policy) | Build + restart the changed (or named) compose services, wait for healthy, show logs on failure | `scripts/rebuild.sh` |
| `/cr-review [NNN]` | Before merge | Runs the read-only `reviewer`, `security-reviewer`, `tester` agents in parallel on the committed tree; PASS only if all three pass | `.claude/agents/`; verdicts recorded by the SubagentStop hook |
| `/fix-bug <report>` | The Creator reports something broken | `fix/<slug>` branch, audit entry, failing regression test first, root cause at `file:line`, minimal fix, `/cr-check`, `/rebuild`, live check, report and wait for approval; escalates to `/cr-start` when it is really a requirement/contract change | AI-DLC "simple bug fix" depth |
| `/cr-finish` | After the Creator approves the final stage | Approval check → diff sanity → AI-DLC records → merge `origin/main` → `/cr-check` → `/cr-review` → merge `--no-ff` → separate push of `main` → watch CI | merge gate hook (`hooks.md`) |

## Deviations from the spec (§10)

- `/cr-finish` merges locally and pushes `main` instead of opening a PR (decision D1). It cannot bypass the gate: the gate is a hook, not a step the skill chooses to run.
- Since Phase 7, `/cr-review` delegates to independent read-only agents; the Phase 6 interim version (built-in `code-review`/`security-review` in the same session) is gone.

## Role agents (`.claude/agents/`)

| Agent | Tools | model / effort | Used by | In merge gate |
|---|---|---|---|---|
| `solution-architect` | Read, Grep, Glob | `opus` / `high` | AI-DLC Design stage, before code | No |
| `reviewer` | Read, Grep, Glob | `sonnet` / `high` | `/cr-review` | Yes |
| `security-reviewer` | Read, Grep, Glob | `sonnet` / `high` | `/cr-review` | Yes |
| `tester` | Read, Grep, Glob | `sonnet` / `medium` | `/cr-review` | Yes |
| `ops-runner` | Bash, Read, Grep, Glob (no edit) | `haiku` / – | `/rebuild` and `/cr-check` (`context: fork`), CI watch in `/cr-finish` (background), logs and live check in `/fix-bug` | No |

Skills set `effort` only on entry points (`/cr-start` medium, `/fix-bug` medium, `/cr-finish` low). A skill's model/effort in the main session lasts for the rest of the turn, so cheap models are never set on a skill: `/rebuild` and `/cr-check` instead run whole in the `ops-runner` agent (`context: fork`, `agent: ops-runner`), and the orchestrating skills (`/cr-review`, `/cr-finish`, `/fix-bug`) stay in the main session because agents cannot start agents, delegating individual steps. `SettingsTest.test_forked_skills_use_existing_agents` checks the fork targets. Rationale and how to change: README §2.6.

The four role agents have no shell and no edit tools: they cannot change the code they judge. Each ends with a machine-readable `VERDICT:` line (see `hooks.md`). Claude Code loads agent definitions at session start; a session started before an agent file changed does not see the change.

## `scripts/review-prep.sh`, `scripts/review-status.sh`

```bash
scripts/review-prep.sh CR-050 aidlc-docs/inception/requirements/cr-050-foo.md   # writes diff + brief for HEAD's tree, prints the brief
scripts/review-status.sh                                                        # markers for HEAD's tree + review hook log; exit 0 = gate would pass
```

See `hooks.md` (Review markers) for what the hook verifies.

## `scripts/rebuild.sh`

```bash
scripts/rebuild.sh               # services changed vs main (BASE=<ref> to override)
scripts/rebuild.sh tts web-gui   # exactly these
HEALTH_TIMEOUT=300 scripts/rebuild.sh rendering
```

A candidate is a compose service built from `services/<name>` with the same name. It waits for `healthy`, or `running` for services without a healthcheck (`web-gui`). Exit 0 all healthy, 1 build/start/health failure (prints the last 40 log lines), 2 usage error or Docker not running.

## Verification (2026-09-28)

- `rebuild.sh` with nothing changed: "nothing to rebuild", exit 0. Unknown service: exit 2. `rebuild.sh api-gateway`: built, started, `healthy`, exit 0 (build was cached, so the container was kept).
- `/cr-start`'s CR-number commands return 049 (audit), 047 (requirement docs), 049 (branches) → next CR is 050.
- All five skills were discovered by the running session as soon as the files were written (they appeared in its skill list with their descriptions).
- `/cr-check` invoked on the Phase 6 branch: ran `make check`, which reported "nothing to verify" (only skills, docs and `scripts/rebuild.sh` changed; `rebuild.sh` has no automated test, it was verified by the runs above). `/cr-start`, `/cr-review` and `/cr-finish` change branches or merge, so they are first exercised on the next real CR.
