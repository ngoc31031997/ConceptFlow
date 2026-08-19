"""Domain value objects for the Script Processing Service (domain-entities.md)."""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class Scene:
    """narration_text is mandatory (Business Rule 3); illustration_hint and
    code_snippet are optional (Business Rules 4-5)."""

    scene_index: int
    narration_text: str
    illustration_hint: str | None
    code_snippet: str | None


@dataclass(frozen=True)
class ParsedScript:
    """No raw_script retained — the service is stateless (Question 5)."""

    scenes: list[Scene]
