#!/usr/bin/env python3
"""PreToolUse hook for Bash: blocks shell access to secrets and gates merges/pushes to main.

Reads the Claude Code hook payload on stdin. Exit 0 lets the command run; exit 2
blocks it and the stderr message is shown to the agent.

1. Secrets: the Read/Edit deny rules in .claude/settings.json do not cover the
   shell, so `cat .env` or `docker compose config` would print real secrets.
2. Merge gate (implementation-audit D1): while on main, `git merge <ref>` needs
   <ref>'s tree to have passed `make check` (marker written by scripts/check.sh)
   and the reviewer, security-reviewer and tester agents (markers written by
   record_review.py), and <ref> must already contain main so the merge result
   is that same tree. `git push` updating main needs the same for the pushed
   commit's tree.
3. The marker directories themselves may not be touched from the shell.

This matches command text; it is a guard against mistakes, not a sandbox. See
docs/agentic/autonomy-policy.md, "Known gaps".

Runs on the macOS system python3 (3.9): stdlib only.
"""
import json
import os
import re
import shlex
import subprocess
import sys

MAIN = "main"

# `.env` / `.env.window` as a path component; `.env.example`, `.env.test`,
# `.envrc` are not matched because the next char is part of the name.
ENV_FILE = re.compile(r"(?:^|[^\w.-])(?:\S*/)?\.env(?:\.window)?(?![\w.-])")
SECRETS_PATH = re.compile(r"(?:^|[^\w-])secrets/[^\s'\"`;&|)]*")
CLIENT_SECRET = re.compile(r"client_secret_\S*\.json")
COMPOSE_CONFIG = re.compile(r"compose(?:\s+[^\s;&|]+)*?\s+config\b")
COMPOSE_CONFIG_SAFE = re.compile(r"--(?:services|volumes|profiles|images|networks)\b")

SEPARATORS = {"&&", "||", ";", "|", "&", "(", ")", "\n"}

REVIEW_AGENTS = ("reviewer", "security-reviewer", "tester")
MARKER_DIRS = re.compile(r"conceptflow/(?:checked-trees|reviewed-trees|review)\b")
# The review hook runs only from Claude Code's SubagentStop event; invoking it by hand, or
# touching subagent transcripts, would let a session feed it a forged review.
REVIEW_HOOK = re.compile(r"record_review|subagents/agent-")


def block(message):
    print(message, file=sys.stderr)
    sys.exit(2)


def check_markers(command):
    if MARKER_DIRS.search(command):
        block("Blocked: the merge-gate marker directories are written only by `make check` "
              "and the review-agent hook. Run /cr-check or /cr-review instead.")
    if REVIEW_HOOK.search(command):
        block("Blocked: the review hook and subagent transcripts are driven only by Claude "
              "Code's SubagentStop event. Run /cr-review; use scripts/review-status.sh "
              "to see the result.")


def check_secrets(command):
    if ENV_FILE.search(command):
        block("Blocked: the command references .env/.env.window, which hold real secrets. "
              "Use .env.example for variable names; ask the Creator for values.")
    for match in SECRETS_PATH.finditer(command):
        if not match.group(0).strip().endswith("secrets/README.md"):
            block("Blocked: the command references secrets/ (OAuth client secrets). "
                  "Only secrets/README.md may be read.")
    if CLIENT_SECRET.search(command):
        block("Blocked: the command references a client_secret_*.json file.")
    if COMPOSE_CONFIG.search(command) and not COMPOSE_CONFIG_SAFE.search(command):
        block("Blocked: `docker compose config` prints the resolved secrets from .env. "
              "Use `docker compose config --services` (or --volumes/--images) instead.")


def git(cwd, *args):
    result = subprocess.run(["git", *args], cwd=cwd, capture_output=True, text=True)
    return result.stdout.strip() if result.returncode == 0 else None


def gate_missing(cwd, rev):
    """What the gate still needs for rev's tree: 'make check' and/or review agents."""
    tree = git(cwd, "rev-parse", "--verify", "--quiet", rev + "^{tree}")
    common = git(cwd, "rev-parse", "--path-format=absolute", "--git-common-dir")
    if not tree or not common:
        return ["make check"] + ["review by " + a for a in REVIEW_AGENTS]
    base = os.path.join(common, "conceptflow")
    missing = []
    if not os.path.exists(os.path.join(base, "checked-trees", tree)):
        missing.append("make check")
    for agent in REVIEW_AGENTS:
        if not os.path.exists(os.path.join(base, "reviewed-trees", f"{tree}.{agent}")):
            missing.append("review by " + agent)
    return missing


def segments(command):
    """Splits a shell command line into simple commands (lists of words)."""
    lexer = shlex.shlex(command, posix=True, punctuation_chars=";&|()")
    lexer.whitespace_split = True
    lexer.commenters = ""
    try:
        tokens = list(lexer)
    except ValueError:  # unbalanced quotes: let the shell report it
        return []
    current, result = [], []
    for token in tokens:
        if token in SEPARATORS or set(token) <= set(";&|"):
            if current:
                result.append(current)
            current = []
        else:
            current.append(token)
    if current:
        result.append(current)
    return result


def git_subcommand(words):
    """Returns (git_dir_override, subcommand, args) or None if not a git command."""
    i = 0
    while i < len(words) and re.match(r"^\w+=", words[i]):  # leading VAR=value
        i += 1
    if i >= len(words) or os.path.basename(words[i]) != "git":
        return None
    i += 1
    cwd = None
    while i < len(words) and words[i].startswith("-"):
        if words[i] in ("-C", "-c", "--git-dir", "--work-tree") and i + 1 < len(words):
            if words[i] == "-C":
                cwd = words[i + 1]
            i += 2
        else:
            i += 1
    if i >= len(words):
        return None
    return cwd, words[i], words[i + 1:]


MERGE_OPTS_WITH_VALUE = {"-m", "-F", "-s", "-X", "--strategy", "--strategy-option",
                         "--file", "--cleanup", "-S", "--into-name"}


def merge_refs(args):
    refs, i = [], 0
    while i < len(args):
        arg = args[i]
        if arg in MERGE_OPTS_WITH_VALUE:
            i += 2
            continue
        if not arg.startswith("-"):
            refs.append(arg)
        i += 1
    return refs


def gate_merge(cwd, refs):
    if not refs:
        block("Blocked: `git merge` on main with no explicit branch. Name the branch to merge.")
    for ref in refs:
        if git(cwd, "rev-parse", "--verify", "--quiet", ref + "^{commit}") is None:
            block(f"Blocked: cannot resolve '{ref}' to verify it before merging into main.")
        if git(cwd, "merge-base", "--is-ancestor", MAIN, ref) is None:
            block(f"Blocked: '{ref}' does not contain the current main, so the merge result "
                  f"would be a tree `make check` never saw. On '{ref}': merge or rebase main, "
                  f"run `make check`, commit, then merge.")
        missing = gate_missing(cwd, ref)
        if missing:
            block(f"Blocked by merge gate (D1): '{ref}' as committed still lacks a pass of: "
                  f"{', '.join(missing)}. Check out '{ref}', commit everything, run /cr-check "
                  f"(make check on a clean tree) and /cr-review, then merge.")


def push_targets_main(args, branch):
    """Returns the local revs whose push would update main."""
    positional, i = [], 0
    while i < len(args):
        arg = args[i]
        if arg in ("--all", "--branches"):
            return [MAIN]
        if arg in ("-o", "--push-option", "--repo", "--receive-pack", "--exec"):
            i += 2
            continue
        if not arg.startswith("-"):
            positional.append(arg)
        i += 1
    refspecs = positional[1:]
    if not refspecs:
        return [MAIN] if branch == MAIN else []
    revs = []
    for spec in refspecs:
        spec = spec.lstrip("+")
        src, _, dst = spec.partition(":")
        dst = dst or src
        if dst in (MAIN, "refs/heads/" + MAIN):
            revs.append(src or MAIN)
    return revs


def gate_push(cwd, revs, changed_main_earlier):
    for rev in revs:
        if changed_main_earlier and rev in (MAIN, "HEAD"):
            block("Blocked: push main as a separate command after the merge/commit, so the "
                  "gate can check the tree that will actually be pushed.")
        if git(cwd, "rev-parse", "--verify", "--quiet", rev + "^{commit}") is None:
            block(f"Blocked: cannot resolve '{rev}' to verify it before pushing to main.")
        if git(cwd, "rev-parse", rev) == git(cwd, "rev-parse", "--verify", "--quiet",
                                              "origin/" + MAIN):
            continue  # nothing new reaches main
        missing = gate_missing(cwd, rev)
        if missing:
            block(f"Blocked by merge gate (D1): the tree of '{rev}' still lacks a pass of: "
                  f"{', '.join(missing)}. Only a merge of a checked and reviewed branch that "
                  f"already contains main can be pushed to main.")


def check_git(command, cwd):
    branch = git(cwd, "rev-parse", "--abbrev-ref", "HEAD")
    changed_main = False  # a merge/commit/pull on main earlier in this command line
    for words in segments(command):
        parsed = git_subcommand(words)
        if parsed is None:
            continue
        override, sub, args = parsed
        where = os.path.join(cwd, override) if override else cwd
        if sub in ("checkout", "switch"):
            # `checkout -b new [start]` also lands on the first name; `checkout -- path`
            # does not switch branches.
            names = [a for a in args if not a.startswith("-")]
            if names and "--" not in args:
                branch = names[0]
        elif sub == "merge" and branch == MAIN:
            if not {"--abort", "--continue", "--quit"} & set(args):
                gate_merge(where, merge_refs(args))
                changed_main = True
        elif sub == "pull" and branch == MAIN:
            positional = [a for a in args if not a.startswith("-")]
            if len(positional) >= 2 and any(r not in (MAIN, "refs/heads/" + MAIN)
                                            for r in positional[1:]):
                block("Blocked: `git pull <remote> <other-branch>` on main merges without "
                      "the gate. Use `git merge <branch>` after `make check` passed on it.")
            changed_main = True
        elif sub in ("commit", "cherry-pick", "revert", "am", "rebase") and branch == MAIN:
            changed_main = True
        elif sub == "push":
            revs = push_targets_main(args, branch)
            if revs:
                gate_push(where, revs, changed_main)


def main():
    payload = json.load(sys.stdin)
    command = (payload.get("tool_input") or {}).get("command") or ""
    cwd = payload.get("cwd") or os.getcwd()
    check_secrets(command)
    check_markers(command)
    check_git(command, cwd)


if __name__ == "__main__":
    main()
