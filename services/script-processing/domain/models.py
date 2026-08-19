"""Domain value objects for the Script Processing Service (domain-entities.md)."""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class Scene:
    """narration_text is mandatory (Business Rule 3); illustration_hint and
    code_snippet are optional (Business Rules 4-5). code_language is the
    fence's language annotation (e.g. ```python) — present only when
    code_snippet is present (Story B3, Rendering Service needs it for
    syntax highlight)."""

    scene_index: int
    narration_text: str
    illustration_hint: str | None
    code_snippet: str | None
    code_language: str | None


@dataclass(frozen=True)
class ParsedScript:
    """No raw_script retained — the service is stateless (Question 5)."""

    scenes: list[Scene]
