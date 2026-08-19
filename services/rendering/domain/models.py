"""Domain value objects for the Rendering Service (domain-entities.md)."""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class SceneRenderRequest:
    """Input to scene rendering. Every field is zero-trust validated by
    RenderSceneUseCase (Functional Design Business Rule 1) — the service
    never trusts upstream data, even though it already passed through
    Script Processing / Content Plugin / TTS Service."""

    project_id: str
    scene_index: int
    narration_text: str
    illustration_hint: str | None
    code_snippet: str | None
    code_language: str | None
    animation_template_id: str
    audio_path: str
    duration_seconds: float


@dataclass(frozen=True)
class SceneRenderResult:
    """Output of scene rendering. duration_seconds is the ACTUAL clip
    duration, which may exceed the requested duration_seconds when the
    animation content is naturally longer (Business Rule 2 — never cut
    content short to force an exact match)."""

    animation_path: str
    duration_seconds: float
