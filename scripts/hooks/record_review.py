#!/usr/bin/env python3
"""SubagentStop hook: records the verdicts of the review agents for the merge gate.

When a `reviewer`, `security-reviewer` or `tester` agent (.claude/agents/) finishes,
the report it delivered must end with `VERDICT: PASS|FAIL tree=<40-hex tree hash>`.
A PASS writes `<git common dir>/conceptflow/reviewed-trees/<tree>.<agent>`, but only
if all of these hold:

0. the agent transcript is the one Claude Code wrote for this session's subagent
   (see genuine_transcript), so a hand-made transcript cannot be passed in;
1. <tree> is the tree of the current HEAD (the review is of what will be merged);
2. the agent's prompt equals `conceptflow/review/<tree>.brief` written by
   scripts/review-prep.sh, apart from leading/trailing whitespace (the reviewing
   session cannot add to or soften the brief);
3. `conceptflow/review/<tree>.diff` still equals the real diff against the
   merge-base with main (the agent read the whole change).

A PASS also needs a clean working tree (the agents read code around the diff from
disk). A FAIL removes the marker and leaves `<tree>.<agent>.fail`, which blocks any
later PASS on that tree: re-running an agent until it agrees does not work; fixing
and committing (a new tree) does. Anything else records nothing. Every outcome is appended to `conceptflow/review/hook.log`, which
scripts/review-status.sh prints. Always exits 0: /cr-review reports a review
without a marker as not done.

Payload fields used (verified on the installed Claude Code, 2026-09-28): agent_type,
agent_id, session_id, transcript_path, agent_transcript_path, last_assistant_message,
cwd. The delivered report is the
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
BASE = "main"
# Fixed on purpose, not read from the environment: an override would let a hand-made
# transcript pass genuine_transcript(). The tests run a copy with this line rewritten.
TRANSCRIPTS_ROOT = "~/.claude/projects"


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


def genuine_transcript(payload):
    """The agent transcript path, if it is where Claude Code keeps this session's subagents.

    Claude Code stores the session transcript at <root>/<project>/<session_id>.jsonl and
    each subagent's at <root>/<project>/<session_id>/subagents/agent-<agent_id>.jsonl
    (verified 2026-09-28). A payload pointing anywhere else, e.g. a forged transcript in
    the scratchpad fed to this script by hand, is rejected. Writing into <root> with
    Edit/Write is denied in .claude/settings.json and the Bash guard blocks commands
    naming this script or subagent transcripts.
    """
    root = os.path.realpath(os.path.expanduser(TRANSCRIPTS_ROOT))
    session = payload.get("session_id") or ""
    agent_id = payload.get("agent_id") or ""
    session_transcript = os.path.realpath(payload.get("transcript_path") or "")
    agent_transcript = os.path.realpath(payload.get("agent_transcript_path") or "")
    if not (session and agent_id and re.fullmatch(r"[\w-]+", session + agent_id)):
        return None
    if not session_transcript.startswith(root + os.sep):
        return None
    if os.path.basename(session_transcript) != f"{session}.jsonl":
        return None
    expected = os.path.join(session_transcript[:-len(".jsonl")], "subagents",
                            f"agent-{agent_id}.jsonl")
    return agent_transcript if agent_transcript == expected else None


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
    transcript = genuine_transcript(payload)
    if transcript is None:
        skip(f"{agent}: the agent transcript is not this session's own subagent transcript")
    entries = read_transcript(transcript)
    if any(e.get("agentId") not in (None, payload.get("agent_id")) for e in entries):
        skip(f"{agent}: the transcript belongs to another agent")
    report = delivered_report(entries)
    if not isinstance(report, str) or not report:
        report = payload.get("last_assistant_message")
    if not isinstance(report, str):
        report = ""
    # Strip fences before dropping empty lines, so a verdict inside ``` still ends the report.
    lines = [line for line in (raw.strip().strip("`").strip() for raw in report.splitlines())
             if line]
    match = VERDICT.match(lines[-1]) if lines else None
    if not match:
        skip(f"{agent}: the report does not end with a VERDICT line")
    verdict, tree = match.groups()

    marks = os.path.join(common, "conceptflow", "reviewed-trees")
    marker = os.path.join(marks, f"{tree}.{agent}")
    failed = marker + ".fail"

    if verdict == "FAIL":
        # A FAIL sticks to its tree: re-running the agent on the same tree until it says
        # PASS must not work. A new commit (new tree) starts clean; only the Creator can
        # lift it by hand (delete the .fail file from a terminal, outside Claude Code).
        os.makedirs(marks, exist_ok=True)
        open(failed, "w").close()
        if os.path.exists(marker):
            os.remove(marker)
        log(f"FAIL {agent} tree={tree}")
        return

    if os.path.exists(failed):
        skip(f"{agent}: an earlier FAIL on tree {tree[:12]} stands; fix and commit (new tree)")
    if git(cwd, "rev-parse", "HEAD^{tree}") != tree:
        skip(f"{agent}: tree {tree[:12]} is not the tree of the current HEAD")
    if git(cwd, "status", "--porcelain") != "":
        skip(f"{agent}: uncommitted changes; the agent may have read code that is not in the tree")
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
