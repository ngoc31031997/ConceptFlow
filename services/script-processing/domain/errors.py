"""Domain-specific exceptions for the Script Processing Service."""

from __future__ import annotations


class ScriptSyntaxError(Exception):
    """Raised when raw_script violates the Markdown grammar (ADR-0011).

    Carries line_number (may be None when the error isn't tied to a single
    line, e.g. "no scenes found") and a human-readable reason — both are
    surfaced in the parse_failed event so the Creator can locate and fix
    the mistake (Story A2 acceptance criteria).
    """

    def __init__(self, line_number: int | None, reason: str) -> None:
        self.line_number = line_number
        self.reason = reason
        super().__init__(f"line {line_number}: {reason}" if line_number else reason)
