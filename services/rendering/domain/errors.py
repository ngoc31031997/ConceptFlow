"""Domain-specific exceptions for the Rendering Service."""

from __future__ import annotations


class UnsupportedTemplateError(Exception):
    """Raised when animation_template_id has no registered template
    (Functional Design Business Rule 1 — zero-trust validation)."""

    def __init__(self, template_id: str) -> None:
        self.template_id = template_id
        super().__init__(f"Unsupported animation_template_id: {template_id!r}")


class InvalidDurationError(Exception):
    """Raised when duration_seconds <= 0 (Business Rule 1)."""

    def __init__(self, duration_seconds: float) -> None:
        self.duration_seconds = duration_seconds
        super().__init__(f"duration_seconds must be > 0, got {duration_seconds}")


class AnimationEngineError(Exception):
    """Raised when Manim crashes or times out (Business Rule error
    classification — transient, the Orchestrator may retry the command)."""
