"""Static lint for common Manim API mistakes, run before the `manim`
subprocess is ever invoked.

This is not a full type-checker for Manim — it is a small, hand-maintained
list of *known* invalid class/kwarg combinations that Creators (or the LLMs
they paste scripts from) repeatedly get wrong, e.g. passing `corner_radius`
to `Rectangle` instead of `RoundedRectangle`. Catching these via `ast` takes
milliseconds and fails fast with a clear message, instead of burning a whole
`manim` subprocess run (which can take tens of seconds) only to crash deep in
Manim's own `__init__` chain with a bare `TypeError`.

Add new entries to INVALID_KWARGS_BY_CLASS as new mistakes are observed.
"""

from __future__ import annotations

import ast
from dataclasses import dataclass

# class name -> {invalid kwarg -> suggested fix}
INVALID_KWARGS_BY_CLASS: dict[str, dict[str, str]] = {
    "Rectangle": {
        "corner_radius": "Rectangle has no corner_radius; use RoundedRectangle(..., corner_radius=...) instead.",
    },
    "Square": {
        "corner_radius": "Square has no corner_radius; use RoundedRectangle(width=w, height=w, corner_radius=...) instead.",
    },
    "Circle": {
        "corner_radius": "Circle has no corner_radius.",
    },
    "Line": {
        "corner_radius": "Line has no corner_radius.",
    },
    "Text": {
        "text": "Text takes its content as the first positional argument, not text=; use Text(\"...\").",
    },
}


@dataclass(frozen=True)
class LintIssue:
    line: int
    message: str

    def __str__(self) -> str:
        return f"line {self.line}: {self.message}"


def lint_manim_script(script_content: str) -> list[LintIssue]:
    """Returns known-bad Manim API usages found in script_content.

    Only flags calls to classes in INVALID_KWARGS_BY_CLASS using one of
    their listed invalid kwargs — anything else (including genuinely unknown
    Manim APIs) is left for the `manim` subprocess itself to judge, since a
    hand-maintained list can never be exhaustive.
    """
    try:
        tree = ast.parse(script_content)
    except SyntaxError as exc:
        return [LintIssue(line=exc.lineno or 0, message=f"script is not valid Python: {exc.msg}")]

    issues: list[LintIssue] = []
    for node in ast.walk(tree):
        if not isinstance(node, ast.Call):
            continue
        class_name = _called_name(node.func)
        invalid_kwargs = INVALID_KWARGS_BY_CLASS.get(class_name)
        if not invalid_kwargs:
            continue
        for kw in node.keywords:
            if kw.arg in invalid_kwargs:
                issues.append(LintIssue(line=node.lineno, message=invalid_kwargs[kw.arg]))

    return issues


def _called_name(func: ast.expr) -> str | None:
    if isinstance(func, ast.Name):
        return func.id
    if isinstance(func, ast.Attribute):
        return func.attr
    return None
