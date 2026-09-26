"""Pull the per-shot functions out of a model reply."""

from __future__ import annotations

import re
from dataclasses import dataclass

_FENCE = re.compile(r"```[a-zA-Z0-9]*\r?\n(.*?)\r?\n?```", re.DOTALL)


class ExtractError(ValueError):
    pass


def strip_fence(text: str) -> str:
    text = text.strip()
    blocks = _FENCE.findall(text)
    if len(blocks) > 1:
        raise ExtractError("the reply contains more than one code block")
    return blocks[0] if blocks else text


@dataclass(frozen=True)
class ShotBlock:
    shot_id: str  # "1.1"
    code: str  # the function, with the comment lines directly above it


def split_shots(code: str, pattern: re.Pattern[str]) -> list[ShotBlock]:
    """Split top-level shot functions. `pattern` matches a column-0 definition
    line and captures the two numbers of the shot id."""
    lines = code.splitlines()
    starts: list[tuple[int, str]] = []
    for i, line in enumerate(lines):
        m = pattern.match(line)
        if m:
            starts.append((i, f"{int(m.group(1))}.{int(m.group(2))}"))
    if not starts:
        raise ExtractError("no shot function found in the reply")

    blocks: list[ShotBlock] = []
    # A comment line right above a definition belongs to it.
    begins: list[int] = []
    for idx, (i, _) in enumerate(starts):
        b = i
        floor = starts[idx - 1][0] + 1 if idx else 0
        while b - 1 >= floor and lines[b - 1].lstrip().startswith(("//", "#")):
            b -= 1
        begins.append(b)

    leading = [ln for ln in lines[: begins[0]] if ln.strip() and not ln.lstrip().startswith(("//", "#"))]
    if leading:
        raise ExtractError(f"unexpected code before the first shot function: {leading[0].strip()[:80]!r}")

    for n, (_, shot_id) in enumerate(starts):
        end = begins[n + 1] if n + 1 < len(begins) else len(lines)
        blocks.append(ShotBlock(shot_id, "\n".join(lines[begins[n] : end]).rstrip()))
    return blocks


def shot_map(blocks: list[ShotBlock]) -> dict[str, str]:
    out: dict[str, str] = {}
    for b in blocks:
        if b.shot_id in out:
            raise ExtractError(f"shot {b.shot_id} is defined twice")
        out[b.shot_id] = b.code
    return out
