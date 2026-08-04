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

## Git commit policy
- Auto-commit (and push, once a remote is configured) is enabled for this project — no need to ask permission before each commit.
- **CRITICAL**: Only commit after the user has given explicit approval for the stage/step whose artifacts are being committed (e.g., after they respond "ok"/"approve"/"go" to a stage's completion message). Do NOT commit immediately after generating artifacts, even if a stage is otherwise complete — generated-but-unapproved work must stay uncommitted until approval lands.
- Each commit should correspond to one approved stage/step, with a message describing what was approved.