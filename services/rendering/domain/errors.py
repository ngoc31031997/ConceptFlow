"""Domain-specific exceptions for the Rendering Service."""

from __future__ import annotations


class InvalidDurationError(Exception):
    """Raised when a narration segment's duration_seconds <= 0 (Business Rule 1)."""

    def __init__(self, duration_seconds: float) -> None:
        self.duration_seconds = duration_seconds
        super().__init__(f"duration_seconds must be > 0, got {duration_seconds}")


class AnimationEngineError(Exception):
    """Raised when Manim crashes, times out, or the script's
    `self.wait(AUTO)` count doesn't match the narration segment count
    (transient/content error — the Orchestrator may retry the command)."""


class InvalidManimApiUsageError(ValueError):
    """Raised when the script's Manim API usage matches a known-bad pattern
    (see domain.script_lint), caught before ever invoking the `manim`
    subprocess. A ValueError subclass so existing `except ValueError`
    handling (consumer.py) needs no change."""

    def __init__(self, issues: list) -> None:
        self.issues = issues
        detail = "; ".join(str(issue) for issue in issues)
        super().__init__(f"invalid Manim API usage in script: {detail}")
