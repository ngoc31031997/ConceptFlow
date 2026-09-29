---
name: rebuild
description: Rebuild and restart the Docker Compose services whose code changed, then wait until they are healthy. Use right after changing service code for a bug fix or Change Request (CLAUDE.md Docker rebuild policy), or when the Creator asks to rebuild/restart a service.
argument-hint: "[service ...]  (default: services changed vs main)"
context: fork
agent: ops-runner
---

# /rebuild

Runs in the `ops-runner` agent (haiku), not in the main session: it is only a script run, and the build output stays out of the main context.

Run the rebuild script. It picks the services, builds, restarts and waits for health; do not re-implement those steps by hand.

```bash
scripts/rebuild.sh $ARGUMENTS
```

- No arguments: every `services/<name>` changed vs `main` (committed, staged, unstaged, untracked) that is a compose service.
- With arguments: exactly those compose services.
- `HEALTH_TIMEOUT=<seconds>` (default 180) for slow services such as `rendering`.

## Report

- Exit 0: list each service and its final state (`healthy`, or `running` for services without a healthcheck such as `web-gui`). Say the change is live.
- Exit 1: say which service failed (build, start, or health) and quote the relevant log lines the script printed. Do not claim the change is live. End with: `Next: the main session diagnoses the failure; if the cause is in the current task's code it fixes it and runs /rebuild again, otherwise it stops and asks the Creator.`
- Exit 2: usage problem (unknown service, Docker not running). Report it as is.
- "nothing to rebuild": no service code changed. Say so. Documentation or script-only changes need no rebuild.

Never run `docker compose down -v`, `docker volume rm` or prune commands to "fix" a failure: they destroy database volumes and are denied by policy.
