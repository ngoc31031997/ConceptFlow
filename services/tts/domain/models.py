"""Domain value objects for the TTS Service (module-structure.md)."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Literal

Language = Literal["vi", "en"]


@dataclass(frozen=True)
class SpeechRequest:
    """Input to speech synthesis.

    project_id/scene_index identify the shared-volume artifact path
    (Low-Level Design Question 5) — they are not persisted anywhere else.
    """

    project_id: str
    scene_index: int
    text: str
    language: str


@dataclass(frozen=True)
class SpeechResult:
    """Output of speech synthesis, returned as the API response body."""

    audio_path: str
    duration_seconds: float
