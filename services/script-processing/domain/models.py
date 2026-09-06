"""Domain value objects for the Script Processing Service (domain-entities.md)."""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class Scene:
    """narration_text is mandatory. illustration_hint/code_snippet/code_language
    are retained only for backward compatibility with the event schema and
    the Orchestrator's Project.Scenes shape — the Manim-script input mode
    (ADR pending) never populates them, since there is no separate template
    to feed a code panel to."""

    scene_index: int
    narration_text: str
    illustration_hint: str | None
    code_snippet: str | None
    code_language: str | None


@dataclass(frozen=True)
class ParsedScript:
    """No raw_script retained — the service is stateless (Question 5).

    scene_class_name is the Manim `Scene` subclass the Rendering Service
    must execute (the first one found in the script)."""

    scenes: list[Scene]
    scene_class_name: str
