"""Fixes to AI-written Remotion shots that have exactly one right answer.

They run without a model call, before a compile check or a repair turn would
spend one. Anything that is not certain is left as it is, for the check to
report and the repair agent to fix: a wrong guess would still compile and draw
the wrong colour or call the wrong function without anyone seeing it. Every
change is returned as a `Fix` so the caller can log it.
"""

from __future__ import annotations

import re
from collections.abc import Collection, Iterable
from dataclasses import dataclass

from app.pipeline.checker import Diagnostic
from app.pipeline.merger import Merged

KIND_AUTOFIX = "autofix"
RULE_PALETTE_NAME = "palette_name"
RULE_TSC_SUGGESTION = "tsc_suggestion"

_PALETTE_IDENT = re.compile(r"\bPALETTE_([A-Za-z0-9_]+)\b")
_PLACEHOLDER = "placeholder"
# tsc's "did you mean" for an unknown name (TS2552) or an unknown property (TS2551).
_SUGGESTION = re.compile(
    r"^(TS255[12]): (?:Cannot find name|Property) '([A-Za-z_$][\w$]*)'.*Did you mean '([A-Za-z_$][\w$]*)'\?",
    re.S)
_UNKNOWN_NAME = "TS2552"


@dataclass(frozen=True)
class Fix:
    """One change made to a shot: `line` counts from 1 within the shot's code."""

    shot: str
    line: int
    rule: str
    before: str
    after: str


def _norm(name: str) -> str:
    return name.replace("_", "").lower()


def fix_palette_names(shot: str, code: str, keys: Iterable[str]) -> tuple[str, list[Fix]]:
    """Rewrite each `PALETTE_<X>` constant the model made up as `PALETTE.<key>`
    when `<X>`, without a `PLACEHOLDER` part, matches exactly one palette key
    ignoring case and underscores. A name matching no key or several keys is
    left unchanged. Returns the new code and the changes made."""
    by_norm: dict[str, list[str]] = {}
    for key in keys:
        by_norm.setdefault(_norm(key), []).append(key)
    fixes: list[Fix] = []
    lines = code.split("\n")
    for n, text in enumerate(lines):
        def replace(m: re.Match[str], line: int = n + 1) -> str:
            rest = "_".join(p for p in m.group(1).split("_") if p.lower() != _PLACEHOLDER)
            found = by_norm.get(_norm(rest), []) if rest else []
            if len(found) != 1:
                return m.group(0)
            after = f"PALETTE.{found[0]}"
            fixes.append(Fix(shot, line, RULE_PALETTE_NAME, m.group(0), after))
            return after

        lines[n] = _PALETTE_IDENT.sub(replace, text)
    return "\n".join(lines), fixes


def fix_tsc_suggestions(
    shots: dict[str, str], merged: Merged, diags: list[Diagnostic], names: Collection[str],
    applied: set[tuple[str, str, str]],
) -> tuple[dict[str, str], list[Fix]]:
    """Apply tsc's "Did you mean 'B'?" on the very line it reports, inside a
    shot of `shots`:

    - an unknown name (TS2552) only when `B` is in `names`, the names the frame
      puts in scope;
    - an unknown property (TS2551) only when `A` and `B` differ in case and
      underscores alone: tsc suggests by spelling distance, so another
      suggestion may name a different thing.

    A (shot, A, B) already in `applied` is not applied again, and is added
    when applied. Returns the new code of the shots that changed and the
    changes made."""
    changed: dict[str, str] = {}
    fixes: list[Fix] = []
    for d in diags:
        m = _SUGGESTION.match(d.message)
        if not m or not d.line:
            continue
        code_kind, before, after = m.groups()
        shot = merged.shot_at(d.line)
        if shot is None or shot not in shots or (shot, before, after) in applied:
            continue
        if code_kind == _UNKNOWN_NAME:
            if after not in names:
                continue
            pattern = re.compile(r"(?<![\w$.])" + re.escape(before) + r"(?![\w$])")
            replacement = after
        else:
            if _norm(before) != _norm(after):
                continue
            pattern = re.compile(r"\." + re.escape(before) + r"(?![\w$])")
            replacement = "." + after
        lines = changed.get(shot, shots[shot].strip("\n")).split("\n")
        local = d.line - merged.lines[shot][0]
        if not 0 <= local < len(lines):
            continue
        new_line, count = pattern.subn(replacement, lines[local])
        if not count:
            continue
        lines[local] = new_line
        changed[shot] = "\n".join(lines)
        applied.add((shot, before, after))
        fixes.append(Fix(shot, local + 1, RULE_TSC_SUGGESTION, before, after))
    return changed, fixes
