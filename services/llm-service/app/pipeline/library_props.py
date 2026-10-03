"""The prop types of a library drawing, read from its own source.

A drawing's `usage` line only names its props; the Remotion Engineer then has
to guess their values (`open={0.5}` for a door whose `open` is
'closed' | 'ajar' | 'open'). This reads the props type the exported component
declares, `FigureProps & {...}`, so the prompt can show it. It counts
brackets rather than parsing TypeScript: a drawing written another way gets
no signature, never a guessed one.
"""

from __future__ import annotations

import re

# Longer than this and the signature is left out: the prompt stays short and
# the drawing's usage line still names its props.
MAX_SIGNATURE_CHARS = 400

_PAIRS = {"{": "}", "(": ")", "[": "]", "<": ">"}
_COMMENT = re.compile(r"/\*.*?\*/|(?:^|(?<=\s))//[^\n]*", re.S)
_FIGURE_PROPS = re.compile(r"\s*:\s*FigureProps\s*&\s*")
_LITERAL = r"(?:'[^'\n]*'|\"[^\"\n]*\"|-?\d+(?:\.\d+)?)"
_ALIAS = re.compile(r"^\s*(?:export\s+)?type\s+([A-Z]\w*)\s*=\s*(" + _LITERAL + r"(?:\s*\|\s*" + _LITERAL + r")*)\s*;",
                    re.M)


def _close(text: str, start: int, angles: bool) -> int | None:
    """Index just past the bracket that closes the one at `start`, or None.

    String literals are skipped. Angle brackets count only when `angles`
    (a type, where they are generics; not a destructuring pattern, where a
    default value may compare), and `=>` never closes one.
    """
    stack: list[str] = []
    i = start
    while i < len(text):
        c = text[i]
        if c in "'\"`":
            end = text.find(c, i + 1)
            if end < 0:
                return None
            i = end + 1
            continue
        if c in _PAIRS and (c != "<" or angles):
            stack.append(_PAIRS[c])
        elif c in ")]}" or (c == ">" and angles and text[i - 1] != "="):
            if not stack or stack.pop() != c:
                return None
            if not stack:
                return i + 1
        i += 1
    return None


def _top_level_members(body: str) -> list[str]:
    """`body` split at the `;`, `,` and newlines that are not nested in brackets."""
    members, depth, current = [], 0, []
    for i, c in enumerate(body):
        if c in "{([<":
            depth += 1
        elif c in "})]" or (c == ">" and body[i - 1:i] != "="):
            depth = max(depth - 1, 0)
        if depth == 0 and c in ";,\n":
            members.append("".join(current))
            current = []
            continue
        current.append(c)
    members.append("".join(current))
    return [" ".join(m.split()) for m in members if m.strip()]


def prop_signature(code: str, name: str) -> str | None:
    """The props `name` declares beyond FigureProps, as `a?: T; b?: U`, with
    each literal-union type alias of the same file replaced by its values.

    Returns None when the component is not written as
    `export function Name({...}: FigureProps & {...})` (or with a named
    parameter, `Name(props: FigureProps & {...})`), declares no own
    props, or the signature is longer than MAX_SIGNATURE_CHARS.
    """
    code = _COMMENT.sub("", code)
    m = re.search(r"export\s+function\s+" + re.escape(name) + r"\s*\(\s*", code)
    if not m:
        return None
    if code[m.end():m.end() + 1] == "{":
        pattern_end = _close(code, m.end(), angles=False)
    else:
        param = re.compile(r"[A-Za-z_$][\w$]*").match(code, m.end())
        pattern_end = param.end() if param else None
    if pattern_end is None:
        return None
    typed = _FIGURE_PROPS.match(code, pattern_end)
    if not typed or code[typed.end():typed.end() + 1] != "{":
        return None
    body_end = _close(code, typed.end(), angles=True)
    if body_end is None:
        return None
    body = code[typed.end() + 1:body_end - 1]
    for alias, values in _ALIAS.findall(code):
        body = re.sub(r"\b" + re.escape(alias) + r"\b", " ".join(values.split()), body)
    signature = "; ".join(_top_level_members(body))
    if not signature or len(signature) > MAX_SIGNATURE_CHARS:
        return None
    return signature
