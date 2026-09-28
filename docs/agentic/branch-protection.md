# Branch protection for `main`

## Policy this implements

The Creator chose (implementation audit, D1) to **keep agent auto-merge into `main`**, with `make check` and review required to pass first. The spec's default of "every change goes through a PR" therefore does not apply. The layers are:

| Layer | Where | Stops |
|---|---|---|
| `make check` + review before merging | Local, enforced by a Claude Code hook (Phase 5) | A red or unreviewed branch being merged into `main` |
| GitHub ruleset on `main` | GitHub | Force pushes and deletion of `main`, from anyone |
| CI (`.github/workflows/ci.yml`) | GitHub, on every push | Nothing *before* the push. It reports a red commit afterwards in a clean environment, which catches "works on my machine" problems |

**Known limitation:** CI runs after the push to `main`, not before. The pre-merge gate is the local hook. A PR-based flow would move the gate server-side; switching to it later only needs the "Require a pull request" and "Require status checks" rules below turned on.

## Manual setup (GitHub web UI)

Agents do not change repository administration. The Creator does this once:

1. Go to **github.com/ngoc31031997/ConceptFlow → Settings → Rules → Rulesets → New ruleset → New branch ruleset**.
2. **Ruleset name:** `protect-main`. **Enforcement status:** Active.
3. **Bypass list:** leave empty.
4. **Target branches → Add target → Include default branch** (`main`).
5. **Rules**, tick only:
   - **Restrict deletions**
   - **Block force pushes**
6. Leave these **off** (each would block the approved auto-merge flow):
   - *Require a pull request before merging*: auto-merge pushes to `main` directly.
   - *Require status checks to pass*: a local merge commit is a new SHA that CI has not seen yet, so the push would be rejected.
   - *Require linear history*: `CLAUDE.md` merges with merge commits.
7. **Create**.

Also recommended: **Settings → Notifications** (personal), or watch the repo's Actions, so a red run on `main` is noticed.

### Verify

The repository is public, so the rules active on `main` can be read without a token:

```bash
curl -s https://api.github.com/repos/ngoc31031997/ConceptFlow/rules/branches/main | grep '"type"'
# expected: "type": "deletion" and "type": "non_fast_forward"
```

An empty list (`[]`) means no ruleset applies to `main` yet.

## Switching to a PR flow later

Turn on *Require a pull request before merging* and *Require status checks to pass* (check: `check-all`) in the same ruleset. Then update `CLAUDE.md`'s CR completion policy and the `/cr-finish` skill to open a PR instead of merging locally.
