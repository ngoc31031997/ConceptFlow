"""LintIssue — one problem a static check found in a script, and its severity.

A blocking issue stops the script before it is rendered; a warning is reported
and the render goes on.
"""

from __future__ import annotations

from dataclasses import dataclass

BLOCKING = "blocking"
WARNING = "warning"


@dataclass(frozen=True)
class LintIssue:
    """A problem at `line` of the script, blocking unless marked a warning."""

    line: int
    message: str
    severity: str = BLOCKING

    @property
    def is_blocking(self) -> bool:
        return self.severity == BLOCKING

    def __str__(self) -> str:
        prefix = "lỗi" if self.is_blocking else "cảnh báo"
        return f"{prefix} dòng {self.line}: {self.message}"


def blocking_issues(issues: list[LintIssue]) -> list[LintIssue]:
    """The issues that stop the script."""
    return [issue for issue in issues if issue.is_blocking]
