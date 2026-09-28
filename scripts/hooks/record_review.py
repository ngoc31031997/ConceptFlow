#!/usr/bin/env python3
"""SubagentStop hook: records the verdicts of the review agents for the merge gate.

When a `reviewer`, `security-reviewer` or `tester` agent (.claude/agents/) finishes,
the report it delivered must end with `VERDICT: PASS|FAIL tree=<40-hex tree hash>`.
A PASS writes `<git common dir>/conceptflow/reviewed-trees/<tree>.<agent>`, but only
if all of these hold:

1. <tree> is the tree of the current HEAD (the review is of what will be merged);
2. the agent's prompt is, byte for byte, `conceptflow/review/<tree>.brief` written by
   scripts/review-prep.sh (the reviewing session cannot soften the brief);
3. `conceptflow/review/<tree>.diff` still equals the real diff against the
   merge-base with main (the agent read the whole change).

A FAIL removes the marker, so a later FAIL overrides an earlier PASS. Anything else
records nothing. Every outcome is appended to `conceptflow/review/hook.log`, which
scripts/review-status.sh prints. Always exits 0: /cr-review reports a review
without a marker as not done.

Payload fields used (verified on the installed Claude Code, 2026-09-28): agent_type,
agent_transcript_path, last_assistant_message, cwd. The delivered report is the
last SubagentHandback call in the agent transcript; last_assistant_message is the
fallback when there is none.
"""
import json
import os
import re
import subprocess
import sys
import time

GATED_AGENTS = ("reviewer", "security-reviewer", "tester")
VERDICT = re.compile(r"^VERDICT: (PASS|FAIL) tree=([0-9a-f]{40})$")
BASE = os.environ.get("BASE", "main")


def git(cwd, *args, strip=True):
    result = subprocess.run(["git", *args], cwd=cwd, capture_output=True, text=True)
    if result.returncode != 0:
        return None
    return result.stdout.strip() if strip else result.stdout


LOG = None  # conceptflow/review/hook.log once the repository is known


def log(message):
    print(f"record_review: {message}", file=sys.stderr)
    if LOG:
        os.makedirs(os.path.dirname(LOG), exist_ok=True)
        with open(LOG, "a") as f:
            f.write(f"{time.strftime('%Y-%m-%dT%H:%M:%S')} {message}\n")


def skip(reason):
    log(f"NOT RECORDED {reason}")
    sys.exit(0)


def read_transcript(path):
    entries = []
    try:
        with open(path) as f:
            for line in f:
                try:
                    entries.append(json.loads(line))
                except ValueError:
                    continue
    except OSError:
        pass
    return entries


def first_prompt(entries):
    for entry in entries:
        if entry.get("type") == "user":
            content = (entry.get("message") or {}).get("content")
            if isinstance(content, str):
                return content
            if isinstance(content, list):
                return "".join(part.get("text", "") for part in content
                               if isinstance(part, dict) and part.get("type") == "text")
            return None
    return None


def delivered_report(entries):
    report = None
    for entry in entries:
        if entry.get("type") != "assistant":
            continue
        for part in (entry.get("message") or {}).get("content") or []:
            if (isinstance(part, dict) and part.get("type") == "tool_use"
                    and part.get("name") == "SubagentHandback"):
                report = (part.get("input") or {}).get("message")
    return report


def main():
    global LOG
    payload = json.load(sys.stdin)
    agent = payload.get("agent_type") or ""
    if agent not in GATED_AGENTS:
        return
    cwd = payload.get("cwd") or os.getcwd()
    common = git(cwd, "rev-parse", "--path-format=absolute", "--git-common-dir")
    if not common:
        skip(f"{agent}: not a git repository")
    LOG = os.path.join(common, "conceptflow", "review", "hook.log")
    entries = read_transcript(payload.get("agent_transcript_path") or "")
    report = delivered_report(entries) or payload.get("last_assistant_message") or ""
    lines = [line.strip().strip("`") for line in report.splitlines() if line.strip()]
    match = VERDICT.match(lines[-1]) if lines else None
    if not match:
        skip(f"{agent}: the report does not end with a VERDICT line")
    verdict, tree = match.groups()

    marks = os.path.join(common, "conceptflow", "reviewed-trees")
    marker = os.path.join(marks, f"{tree}.{agent}")

    if verdict == "FAIL":
        if os.path.exists(marker):
            os.remove(marker)
        log(f"FAIL {agent} tree={tree}")
        return

    if git(cwd, "rev-parse", "HEAD^{tree}") != tree:
        skip(f"{agent}: tree {tree[:12]} is not the tree of the current HEAD")
    review = os.path.join(common, "conceptflow", "review")
    try:
        with open(os.path.join(review, f"{tree}.brief")) as f:
            brief = f.read()
        with open(os.path.join(review, f"{tree}.diff")) as f:
            diff_file = f.read()
    except OSError:
        skip(f"{agent}: no brief/diff from scripts/review-prep.sh for tree {tree[:12]}")
    if (first_prompt(entries) or "").strip() != brief.strip():
        skip(f"{agent}: its prompt is not the brief written by scripts/review-prep.sh")
    base_commit = git(cwd, "merge-base", BASE, "HEAD")
    if base_commit is None or git(cwd, "diff", base_commit, "HEAD", strip=False) != diff_file:
        skip(f"{agent}: the diff file does not match the real diff against {BASE}")

    os.makedirs(marks, exist_ok=True)
    open(marker, "w").close()
    log(f"PASS {agent} tree={tree}")


if __name__ == "__main__":
    main()
