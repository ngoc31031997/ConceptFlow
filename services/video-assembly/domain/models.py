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
class SubtitleCue:
    """One narration line and the window it stays on screen (CR-001 FR9.3).

    Timings come from the Orchestrator — real synthesized-audio durations when
    narration is on, estimated reading time when it is off — so subtitles stay
    in step with the animation either way.
    """

    scene_index: int
    text: str
    start_time: float
    end_time: float


@dataclass(frozen=True)
class SubtitleStyle:
    """Creator-chosen subtitle appearance (CR-001 FR9.4)."""

    font_size: str = "medium"  # small | medium | large
    text_color: str = "#FFFFFF"
    background_opacity: float = 0.6  # 0.0 = no box behind the text
    position: str = "bottom"  # bottom | top


@dataclass(frozen=True)
class VideoAssemblyRequest:
    """Input to video assembly: one project's rendered (silent) video, its
    ordered narration audio segments, plus optional background music and
    subtitles.

    audio_segments is empty when the Creator disabled narration (CR-001) —
    the result is then a silent video, or one carrying background music alone.
    """

    project_id: str
    video_path: str
    audio_segments: list[str]
    background_music_path: str | None = None
    subtitle_cues: list[SubtitleCue] | None = None
    subtitle_style: SubtitleStyle | None = None


@dataclass(frozen=True)
class VideoAssemblyResult:
    """Output of video assembly, returned as the event payload.

    Deliberately minimal (Functional Design Rule 6) — Publisher Service only
    needs video_path, and GUI preview can read other metadata directly from
    the file.
    """

    video_path: str
