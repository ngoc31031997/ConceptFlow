# Project AI Instructions

This repository follows the AWS AI-DLC workflow.

Always start by reading:

1. .ai-dlc/steering/aws-aidlc-rules/core-workflow.md
2. Every markdown under .ai-dlc/aws-aidlc-rule-details/

When implementing:
- Follow the workflow strictly.
- Do not skip architecture or design.
- Keep production-ready quality.
- Explain design decisions before coding.

## No fake code, no silent skipping
- Never write placeholder/stub/mock implementations in production code, hard-coded fake outputs, `TODO: implement` bodies, or code paths that pretend to succeed (e.g. returning canned data, swallowing an error and continuing as if it worked).
- Never silently skip a requirement, a step of an approved plan, a failing test, or an error. If something in the plan is not done, say so explicitly in the report.
- If you cannot continue for any reason (missing information, missing credentials/tools, a design conflict, an ambiguity that changes behaviour, a test you cannot make pass honestly), STOP and ask the user instead of guessing or working around it.
- Report outcomes faithfully: tests that fail are reported as failing with their output; checks that were not run are reported as not run.

## Docker rebuild policy
- Whenever code is changed to fix a bug or implement/patch a Change Request, rebuild the Docker image(s) for every affected service and restart the container(s) via `docker compose build <service>` + `docker compose up -d <service>`, then confirm it reports healthy — do this immediately, without waiting to be asked, so the Creator can check the change live at any time.
- Only rebuild the services whose code actually changed (check `git status`/diff to scope it) — no need to rebuild the whole stack for a one-service fix.

## Git branching policy
- Every Change Request (CR) gets its own feature branch (`feature/cr-<NNN>-<slug>`) — never implement a new CR directly on `main`/`master` or on another CR's still-open branch. See `.ai-dlc/aws-aidlc-rule-details/construction/git-branching.md` for the full rule. This matters because multiple agents may work in this repo concurrently.

## Git commit policy
- Auto-commit (and push, once a remote is configured) is enabled for this project — no need to ask permission before each commit.
- **CRITICAL**: Only commit after the user has given explicit approval for the stage/step whose artifacts are being committed (e.g., after they respond "ok"/"approve"/"go" to a stage's completion message). Do NOT commit immediately after generating artifacts, even if a stage is otherwise complete — generated-but-unapproved work must stay uncommitted until approval lands.
- Each commit should correspond to one approved stage/step, with a message describing what was approved.
## CR completion policy
- When a Change Request is finished (implemented, verified, and its stage approved per the policy above), automatically commit and push its `feature/cr-<NNN>-<slug>` branch, then merge it into `main` and push `main` — no need to ask again.
- Before merging, pull the latest `main` so the merge is fast and conflict-free; if the merge conflicts, stop and report instead of resolving blindly.

## Read the codebase through graphify first (token budget)
- Do NOT explore the repo by listing directories, globbing broadly, or reading many files "to get context". Start every code question, CR analysis or bug hunt from the graph:
  1. Check freshness: `graph.json`'s `built_at_commit` must equal `git rev-parse HEAD`; otherwise run `make graph` (about 10 s, no LLM).
  2. Architecture overview: read `graphify-out/GRAPH_REPORT.md` (once per session, not per task).
  3. Locate: `graphify query "<question>"` / `graphify explain "<symbol>"` to find the files and symbols involved; `graphify path "<A>" "<B>"` for a call chain; `graphify affected "<symbol>"` for callers before changing behaviour.
  4. Read only the files (and, for large files, only the line ranges) the graph points to, plus the code you will change. Do not re-read a file already read in this session unless it changed.
- Fall back to `grep` for exact strings the graph cannot answer (config keys, error messages, SQL, env names), scoped to the service directory, not the whole repo.
- For what the graph cannot see (RabbitMQ messages, HTTP between services, DB access) go straight to the contract index `docs/contracts/README.md` and the relevant ADR instead of scanning services.

## Code knowledge graph (graphify)
- `graphify-out/` (git-ignored) holds a local graph of the code and markdown structure, built with tree-sitter AST: no LLM, nothing leaves the machine. Refresh it with `make graph`; after `make graph-hooks` the graphify git hooks keep it in step with commits and branch switches. Details: `docs/agentic/graphify.md`.
- Use it to orient before reading many files, not instead of reading the code you change: `graphify query "<question>"`, `graphify explain "<symbol>"`, `graphify path "<A>" "<B>"`, `graphify affected "<symbol>"` (who depends on it), `graphify-out/GRAPH_REPORT.md` for the architecture overview.
- It only knows static edges (imports, calls, references). RabbitMQ messages, HTTP calls between services and DB access are not in it; check the contract index `docs/contracts/README.md` and the ADRs for those. If `graph.json`'s `built_at_commit` is not HEAD, run `make graph` first.
- The agentic layer (skills `/cr-*`, `/fix-bug`, `/rebuild`, role agents, hooks, review gate, `make setup/build/check/check-all`, CI workflow) was removed on 2026-09-29/30; do not invoke or recreate it. Rebuild with `docker compose build/up` directly. The exceptions are three skills, added back on 2026-09-30:
  - `/cr` (`.claude/skills/cr/`, Opus 5.5, medium effort): for a new request, opens the `feature/cr-<NNN>-<slug>` branch, analyses the codebase through graphify and writes a proposed solution (`aidlc-docs/construction/plans/cr-<NNN>-<slug>-design.md`); after the Creator approves, commits it and invokes `/code`.
  - `/code` (`.claude/skills/code/`, Sonnet, medium effort): implements the approved design, runs tests, rebuilds the changed services, reports; commits nothing.
  - `/deliver` (`.claude/skills/deliver/`, Sonnet, medium effort, Creator-invoked only): after the Creator approves, it commits, pushes, merges into main, pulls main, refreshes the graphify graph for the new main and rebuilds the changed services.

## Web UI / UX design rules
- Any UI task must follow `docs/ux-ui-design-rules.md` (screen flow order, etc.). Read it before changing or adding a screen.
