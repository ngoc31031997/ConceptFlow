"""Domain value objects for the Video Assembly Service (module-structure.md).

Revision (Functional Design Question 2): scenes carry an explicit
scene_index instead of relying on parallel-array position, so the
service never has to trust the order Orchestrator happened to send them in.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class SceneAssemblyInput:
    """One scene's animation clip + audio clip, identified by scene_index."""

    scene_index: int
    clip_path: str
    audio_path: str


@dataclass(frozen=True)
class VideoAssemblyRequest:
    """Input to video assembly — one project's scenes plus optional background music."""

    project_id: str
    scenes: list[SceneAssemblyInput]
    background_music_path: str | None = None


@dataclass(frozen=True)
class VideoAssemblyResult:
    """Output of video assembly, returned as the event payload.

    Deliberately minimal (Functional Design Rule 6) — Publisher Service only
    needs video_path, and GUI preview can read other metadata directly from
    the file.
    """

    video_path: str
