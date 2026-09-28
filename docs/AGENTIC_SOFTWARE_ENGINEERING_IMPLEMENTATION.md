# Agentic Software Engineering Implementation Specification

> **Project:** ConceptFlow  
> **Purpose:** Upgrade the repository into a production-grade Agentic Software Engineering workflow.
>
> **Primary implementation agent:** Claude Code  
> **Human role:** Architecture approval, risk approval, final PR approval  
> **Core methodology:** AI-DLC  
> **Important:** Do not replace AI-DLC. Extend it with engineering guardrails, automated verification, independent review, evaluation, observability, and risk-based autonomy.

---

# 1. Objective

Implement an Agentic Software Engineering workflow around the existing ConceptFlow repository.

The target is NOT "more AI agents".

The target is:

> An AI coding agent can execute well-defined software tasks with increasing autonomy while the repository mechanically enforces correctness, security, isolation, traceability, and human approval at appropriate risk boundaries.

The implementation must preserve the existing project architecture and AI-DLC workflow unless a change is explicitly justified.

---

# 2. Existing State

The repository currently has or is expected to have the following concepts:

- `CLAUDE.md`
- AI-DLC lifecycle:
  - Inception
  - Design
  - Code
  - Build & Test
- `aidlc-docs/`
- ADRs under `aidlc-docs/decisions/`
- `audit.md`
- `aidlc-state.md`
- Claude Code memory/context
- `.claude/`
- Git branches/worktrees
- Multiple services
- Service-specific tests
- Contract tests
- Existing coding-agent workflow

Before modifying anything:

1. Inspect the repository.
2. Identify what already exists.
3. Do NOT recreate existing functionality.
4. Do NOT assume the examples in this document exactly match the current Claude Code version.
5. Use the currently supported Claude Code syntax/features found in the repository or official documentation available to the agent.
6. Preserve working project conventions.

---

# 3. Target Architecture

The target workflow is:

```text
                         HUMAN
                           │
                           ▼
                    ┌──────────────┐
                    │    INTENT    │
                    │ Issue / CR   │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │     PLAN     │
                    │   AI-DLC     │
                    └──────┬───────┘
                           │
                    HUMAN APPROVAL
                           │
                           ▼
              ┌─────────────────────────┐
              │     AGENT EXECUTION     │
              │                         │
              │ Coding Agent             │
              │ Skills                   │
              │ MCP / Tools              │
              │ Repository Context       │
              │ Project Memory            │
              └────────────┬────────────┘
                           │
                           ▼
              ┌─────────────────────────┐
              │       GUARDRAILS        │
              │                         │
              │ Permissions              │
              │ Sandbox / Isolation      │
              │ Secrets Protection       │
              │ Hooks                   │
              │ Policy                  │
              └────────────┬────────────┘
                           │
                           ▼
              ┌─────────────────────────┐
              │     VERIFICATION        │
              │                         │
              │ Format / Lint            │
              │ Unit Tests               │
              │ Integration Tests        │
              │ Contract Tests           │
              │ Build                   │
              └────────────┬────────────┘
                           │
                           ▼
              ┌─────────────────────────┐
              │   INDEPENDENT REVIEW    │
              │                         │
              │ Code Reviewer            │
              │ Security Reviewer        │
              │ QC / Acceptance          │
              └────────────┬────────────┘
                           │
                           ▼
                         PR
                           │
                           ▼
                    HUMAN REVIEW
                           │
                           ▼
                         MERGE
                           │
                           ▼
              ┌─────────────────────────┐
              │      OBSERVABILITY      │
              │                         │
              │ Agent runs               │
              │ Failures                 │
              │ Cost / latency           │
              │ Human intervention       │
              │ Production regressions   │
              └────────────┬────────────┘
                           │
                           ▼
              ┌─────────────────────────┐
              │       AGENT EVALS       │
              │                         │
              │ Task success             │
              │ Policy compliance        │
              │ Regression rate          │
              │ First-pass CI rate       │
              └────────────┬────────────┘
                           │
                           └──────► Improve
```

---

# 4. Core Design Principles

## 4.1 AI-DLC remains the lifecycle

Do not replace:

```text
Inception → Design → Code → Build & Test
```

Add engineering controls around it.

## 4.2 Mechanical enforcement beats prompting

If a rule can be checked by a machine, implement it as:

- CI
- hook
- test
- lint
- policy
- permission
- script

Do not rely only on:

> "Claude, remember to..."

## 4.3 Agent must not be the only verifier of its own work

The coding agent may implement.

Independent verification must include:

- automated tests
- CI
- independent reviewer agent where appropriate
- human approval at risk boundaries

## 4.4 Least privilege

Agents must receive only the permissions required for the current task.

Never expose:

- personal credentials
- production secrets
- unrelated tokens
- `.env` containing secrets
- OAuth client secrets

unless there is an explicit, controlled mechanism.

## 4.5 Risk-based autonomy

Do not make all actions equally autonomous.

Example policy:

| Action | Default autonomy |
|---|---|
| Read source code | Agent |
| Search repository | Agent |
| Run unit tests | Agent |
| Format/lint | Agent |
| Create branch/worktree | Agent |
| Modify application code | Agent |
| Open PR | Agent |
| Run integration tests | Agent |
| Modify CI configuration | Review required |
| Database migration | Human approval |
| Production deployment | Human approval |
| Production data mutation | Human approval |
| Access secrets | Deny by default |
| Change security policy | Human approval |
| Force push | Deny |
| Delete production resources | Deny |

---

# 5. Phase A — Repository Verification Foundation

## Goal

Create one canonical verification entry point.

## Requirement

Implement:

```bash
make check
```

and:

```bash
make check-all
```

### `make check`

Runs only the checks relevant to the current change.

Expected behavior:

```text
detect changed services
        ↓
run formatter/linter
        ↓
run relevant unit tests
        ↓
run relevant integration/contract tests
        ↓
return non-zero on failure
```

### `make check-all`

Runs the complete repository verification suite.

It must be suitable for CI.

## Acceptance criteria

- `make check` exists.
- `make check-all` exists.
- Exit code is non-zero when a check fails.
- Agent can execute the same command locally and in CI.
- Documentation explains what each command does.
- Do not duplicate service-specific test logic unnecessarily.

---

# 6. Phase B — CI

## Goal

CI becomes the authoritative clean-environment verification layer.

Create or update:

```text
.github/workflows/ci.yml
```

## Minimum behavior

On every pull request:

```text
checkout
  ↓
install required runtimes/dependencies
  ↓
make check-all
  ↓
publish result
```

Use matrix jobs only where they actually improve correctness or isolation.

## Requirements

CI must:

- run in a clean environment
- fail on test/build/lint failure
- expose useful logs
- avoid relying on local developer state
- avoid accessing personal credentials
- use repository-managed secrets only where necessary

## Acceptance criteria

A PR cannot be considered merge-ready when required CI checks are red.

---

# 7. Phase C — Branch Protection

Configure repository policy so that normal changes follow:

```text
Agent
  ↓
branch
  ↓
PR
  ↓
CI
  ↓
review
  ↓
merge
```

Do NOT allow normal agent workflows to:

```text
agent → local merge → push main
```

## Requirements

Document the expected GitHub branch protection settings.

At minimum:

- PR required
- required CI checks
- no force push
- no direct push to protected branch where appropriate
- review required according to repository policy

If repository administration cannot be changed automatically, document the exact manual action required.

---

# 8. Phase D — Agent Permissions

Create/update Claude Code permissions configuration.

## Default deny

Block dangerous operations such as:

```text
force push
rm -rf
reading secret directories
reading .env secrets
destructive production operations
```

The exact syntax must match the currently installed Claude Code version.

Do not blindly copy old Claude Code syntax.

## Principle

Permissions should be:

```text
explicit allow
+
explicit deny
+
least privilege
```

---

# 9. Phase E — Hooks

Convert repeatable rules into deterministic hooks/scripts.

## Required hooks

### Post-edit verification

After relevant source edits:

```text
Edit
  ↓
format/lint changed files
```

Do not run expensive full-repository verification after every tiny edit unless necessary.

### Stop verification

Before the coding agent declares completion:

```text
agent attempts to stop
        ↓
make check
        ↓
PASS → allow stop
FAIL → return failures to agent
```

The hook must use the currently supported Claude Code hook mechanism.

If the installed version does not support a desired lifecycle event, implement the closest supported mechanism and document the difference.

## Important

Hooks are enforcement mechanisms.

Do not make the hook itself overly expensive or flaky.

---

# 10. Phase F — Skills / Commands

Create reusable skills for repetitive workflows.

Recommended commands:

```text
/cr-start
/rebuild
/cr-check
/cr-review
/cr-finish
```

## `/cr-start`

Responsibilities:

```text
validate clean starting state
↓
sync main
↓
create CR branch/worktree
↓
initialize AI-DLC task context
```

## `/rebuild`

Responsibilities:

```text
identify changed service
↓
build/restart relevant service
↓
wait for healthy state
↓
report failure clearly
```

## `/cr-check`

Responsibilities:

```text
run make check
↓
summarize failures
```

## `/cr-review`

Responsibilities:

```text
run independent reviewer
↓
run QC / acceptance check
↓
produce actionable findings
```

## `/cr-finish`

Responsibilities:

```text
make check
↓
review
↓
verify git diff
↓
verify CR / AI-DLC state
↓
commit if policy allows
↓
open PR
```

Do not allow `/cr-finish` to bypass CI or branch protection.

---

# 11. Phase G — Independent Review Agents

Create reviewer roles under:

```text
.claude/agents/
```

Recommended:

```text
reviewer.md
security-reviewer.md
qc.md
```

## Reviewer

Must be read-only.

Check:

- logic bugs
- edge cases
- regression risk
- architecture violations
- ADR violations
- missing tests
- retry/idempotency issues
- transaction boundaries
- concurrency issues

## Security reviewer

Check:

- secrets
- injection
- authentication/authorization
- unsafe dependencies
- unsafe command execution
- sensitive data exposure
- permission escalation

## QC

Compare:

```text
CR acceptance criteria
        vs
implementation
        vs
tests
```

The QC agent should identify missing acceptance criteria coverage.

## Important

Reviewer agents must not silently modify implementation code.

They report findings.

---

# 12. Phase H — Worktree Isolation

Every independent CR should use its own worktree.

Example:

```text
worktrees/
  cr-041/
  cr-042/
  cr-043/
```

Each worktree must avoid resource collisions.

For Docker-based services:

```text
COMPOSE_PROJECT_NAME
```

must be unique per worktree.

Ports should be allocated deterministically.

Example:

```text
CR 041 → port range A
CR 042 → port range B
CR 043 → port range C
```

Do not hardcode conflicting ports across parallel agents.

---

# 13. Phase I — Background Agent Workflow

Only enable background agents after Phase A–H are stable.

Initial scope should be low-risk tasks:

- bug fixes with existing reproduction tests
- dependency updates
- documentation
- test additions
- lint/format fixes
- small refactors

Workflow:

```text
GitHub Issue
     ↓
agent:dev label
     ↓
agent claims task
     ↓
isolated worktree
     ↓
AI-DLC plan
     ↓
implementation
     ↓
tests
     ↓
review
     ↓
PR
     ↓
human review
```

The agent must NOT directly merge to protected branches.

---

# 14. Phase J — Agent Evaluation

This is a required layer.

Software tests answer:

> "Does the software work?"

Agent evaluations additionally answer:

> "Does the agent perform the task correctly and safely?"

Create an evaluation structure, for example:

```text
evals/
  tasks/
  expected/
  policies/
  reports/
```

## Metrics

Track at least:

### Task success rate

```text
successful tasks / total tasks
```

### First-pass CI rate

```text
PRs passing CI first time / total PRs
```

### Human intervention rate

How often a human must manually correct the agent.

### Review defect rate

How often reviewer/QC finds a meaningful defect.

### Regression rate

Bugs introduced after an agent change.

### Policy violation rate

Attempts to violate permissions or repository policy.

### Rework rate

How often agent implementation must be substantially rewritten.

Do NOT optimize for:

```text
lines of code generated
number of commits
number of agent actions
```

---

# 15. Phase K — Observability

Create an auditable record for important agent runs.

At minimum capture:

```text
task/CR ID
agent
model
start/end time
tools used
verification result
review result
PR
human intervention
final status
```

Do not log:

- secrets
- access tokens
- passwords
- sensitive production data

Prefer structured logs.

Example:

```json
{
  "task_id": "CR-041",
  "agent": "developer",
  "status": "success",
  "ci": "passed",
  "review": "passed",
  "human_intervention": 1
}
```

---

# 16. Phase L — Risk-Based Autonomy Policy

Create a documented policy:

```text
docs/agentic/autonomy-policy.md
```

Define action classes:

## Level 0 — Read-only

Agent can act automatically.

Examples:

- search repository
- read code
- inspect logs
- inspect test results

## Level 1 — Local development

Agent can act automatically.

Examples:

- edit code
- run tests
- create branch
- install non-sensitive local dependencies

## Level 2 — Repository collaboration

Agent can act with verification.

Examples:

- open PR
- modify tests
- modify CI
- dependency updates

## Level 3 — Sensitive operations

Human approval required.

Examples:

- database migration
- infrastructure changes
- production deployment
- security policy changes

## Level 4 — Destructive / prohibited

Agent cannot perform automatically.

Examples:

- production data deletion
- secret extraction
- force push protected branches
- disabling security controls

---

# 17. AI-DLC Integration

The final workflow must preserve:

```text
INCEPTION
    ↓
DESIGN
    ↓
HUMAN APPROVAL
    ↓
CODE
    ↓
BUILD & TEST
    ↓
AI REVIEW
    ↓
QC
    ↓
PR
    ↓
CI
    ↓
HUMAN APPROVAL
    ↓
MERGE
```

AI-DLC documents remain the source of truth for:

- requirements
- design decisions
- architecture decisions
- task state
- acceptance criteria

Agentic tooling must reference these documents rather than creating a second competing planning system.

---

# 18. Repository Context Strategy

Keep root `CLAUDE.md` short.

It should contain:

- project identity
- architecture summary
- critical commands
- hard rules
- forbidden operations
- pointers to deeper documentation

Avoid loading the entire documentation tree for every task.

Use:

```text
CLAUDE.md
    ↓
service-specific CLAUDE.md
    ↓
skill
    ↓
task-specific AI-DLC documents
```

Load detailed context only when needed.

---

# 19. Definition of Done

A task is NOT complete because the agent says:

```text
Done.
```

A task is complete only when:

```text
[ ] AI-DLC requirements satisfied
[ ] implementation complete
[ ] relevant tests pass
[ ] make check passes
[ ] no forbidden changes
[ ] reviewer passes
[ ] QC passes
[ ] git diff reviewed
[ ] PR created
[ ] CI passes
[ ] required human approval obtained
```

---

# 20. Implementation Order

Implement in this order.

## Phase 1

```text
Repository inspection
```

Do not modify anything yet.

Produce:

```text
docs/agentic/implementation-audit.md
```

containing:

- existing capabilities
- missing capabilities
- conflicting configurations
- risky configurations
- recommended implementation order

## Phase 2

```text
make check
make check-all
```

## Phase 3

```text
CI
branch protection documentation
```

## Phase 4

```text
permissions
deny rules
```

## Phase 5

```text
hooks
```

## Phase 6

```text
skills/commands
```

## Phase 7

```text
reviewer
security-reviewer
QC
```

## Phase 8

```text
worktree isolation
```

## Phase 9

```text
background agent workflow
```

## Phase 10

```text
evaluation
observability
```

## Phase 11

```text
risk-based autonomy policy
```

---

# 21. Implementation Rules for Claude

## Rule 1 — Inspect first

Before changing anything:

```text
read repository structure
read CLAUDE.md
inspect .claude/
inspect AI-DLC documents
inspect package/build/test configuration
inspect GitHub workflows
inspect existing scripts
```

## Rule 2 — Reuse existing infrastructure

If the repository already has:

- test scripts
- Makefile
- CI
- hooks
- skills
- reviewers
- worktree scripts

extend them instead of creating duplicates.

## Rule 3 — Small increments

Implement one phase at a time.

After each phase:

```text
run verification
review diff
update documentation
```

## Rule 4 — Do not weaken existing security

Never solve an agent problem by:

- disabling security checks
- exposing secrets
- granting unrestricted shell access
- bypassing branch protection
- disabling tests
- ignoring failing CI

## Rule 5 — Never fake verification

Do not report:

```text
tests passed
CI passed
review passed
```

unless it was actually executed and verified.

## Rule 6 — Stop on ambiguity

If an implementation decision materially changes:

- architecture
- security
- data model
- production behavior
- deployment behavior

stop and ask the human.

---

# 22. Expected Deliverables

At the end of implementation, the repository should contain the appropriate subset of:

```text
.github/
  workflows/
    ci.yml

.claude/
  agents/
    reviewer.md
    security-reviewer.md
    qc.md
  skills/
    cr-start/
    cr-check/
    cr-review/
    cr-finish/
    rebuild/

scripts/
  hooks/
    check-before-stop.sh
    lint-changed.sh

Makefile

docs/
  agentic/
    implementation-audit.md
    autonomy-policy.md
    workflow.md
    evaluation.md

evals/
  tasks/
  expected/
  policies/
  reports/
```

Do not create a file merely because it appears in this example.

Only create files that fit the actual repository structure.

---

# 23. Final Verification

Before declaring implementation complete, run:

```text
make check-all
```

Then verify:

```text
[ ] CI workflow is valid
[ ] no secrets are exposed
[ ] dangerous permissions are denied
[ ] hooks execute correctly
[ ] reviewer is read-only
[ ] QC validates acceptance criteria
[ ] worktrees can run independently
[ ] background workflow cannot bypass PR/CI
[ ] AI-DLC remains intact
[ ] documentation matches actual implementation
```

Finally produce:

```text
docs/agentic/implementation-report.md
```

with:

1. What was implemented
2. What already existed
3. What was intentionally not changed
4. Verification results
5. Remaining manual GitHub configuration
6. Known limitations
7. Recommended next phase

---

# 24. Success Criteria

The project should reach this operating model:

```text
Human defines intent
        ↓
AI-DLC defines plan
        ↓
Agent implements
        ↓
Machine enforces rules
        ↓
Machine verifies code
        ↓
Independent agent reviews
        ↓
CI verifies clean environment
        ↓
Human approves risk-sensitive changes
        ↓
Merge
        ↓
Telemetry + evaluation
        ↓
Improve agent workflow
```

The goal is not maximum autonomy.

The goal is:

> **Maximum useful autonomy within explicit, testable, auditable boundaries.**
