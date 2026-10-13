"""Domain-specific exceptions for the Rendering Service."""

from __future__ import annotations


class InvalidDurationError(Exception):
    """Raised when a narration segment's duration_seconds <= 0 (Business Rule 1)."""

    def __init__(self, duration_seconds: float) -> None:
        self.duration_seconds = duration_seconds
        super().__init__(f"duration_seconds must be > 0, got {duration_seconds}")


class AnimationEngineError(Exception):
    """Raised when the render engine crashes, times out, or the script's
    narration marks don't match the narration segment count
    (transient/content error — the Orchestrator may retry the command)."""


class UnsupportedEngineError(ValueError):
    """Raised for a command whose engine is not Remotion, the only engine
    Rendering runs. A ValueError subclass so the consumer's `except ValueError`
    path fails the step with this message."""

    def __init__(self, engine: str) -> None:
        self.engine = engine
        super().__init__(f"engine {engine!r} không còn được hỗ trợ; chỉ có remotion")
