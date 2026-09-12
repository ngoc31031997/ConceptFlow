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

## Docker rebuild policy
- Whenever code is changed to fix a bug or implement/patch a Change Request, rebuild the Docker image(s) for every affected service and restart the container(s) via `docker compose build <service>` + `docker compose up -d <service>`, then confirm it reports healthy — do this immediately, without waiting to be asked, so the Creator can check the change live at any time.
- Only rebuild the services whose code actually changed (check `git status`/diff to scope it) — no need to rebuild the whole stack for a one-service fix.

## Git commit policy
- Auto-commit (and push, once a remote is configured) is enabled for this project — no need to ask permission before each commit.
- **CRITICAL**: Only commit after the user has given explicit approval for the stage/step whose artifacts are being committed (e.g., after they respond "ok"/"approve"/"go" to a stage's completion message). Do NOT commit immediately after generating artifacts, even if a stage is otherwise complete — generated-but-unapproved work must stay uncommitted until approval lands.
- Each commit should correspond to one approved stage/step, with a message describing what was approved.