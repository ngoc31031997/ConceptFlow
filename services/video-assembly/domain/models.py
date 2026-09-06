"""Domain value objects for the Video Assembly Service.

Manim-script input mode: Rendering produces one silent video for the whole
project (not per-scene clips), so assembly's job is simpler than before —
concatenate the ordered narration audio_segments into one track, mux it
onto that single video, and optionally overlay background music. There is
no per-scene clip:audio pairing or format-consistency check anymore (only
one video, no clips to compare).
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class VideoAssemblyRequest:
    """Input to video assembly: one project's rendered (silent) video, its
    ordered narration audio segments, plus optional background music."""

    project_id: str
    video_path: str
    audio_segments: list[str]
    background_music_path: str | None = None


@dataclass(frozen=True)
class VideoAssemblyResult:
    """Output of video assembly, returned as the event payload.

    Deliberately minimal (Functional Design Rule 6) — Publisher Service only
    needs video_path, and GUI preview can read other metadata directly from
    the file.
    """

    video_path: str
