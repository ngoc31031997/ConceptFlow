"""Domain value objects for the Video Assembly Service.

Manim-script input mode: Rendering produces one silent video for the whole
project (not per-scene clips), so assembly's job is to lay each narration
segment onto that video at its own offset, and optionally overlay
background music. There is no per-scene clip:audio pairing or
format-consistency check anymore (only one video, no clips to compare).

CR-002 replaced the earlier `audio_segments: list[str]`, which assembly
simply concatenated back to back. That was wrong: the video's timeline is
animation time *plus* narration time, so laying the audio end to end made
every segment after the first play early, by the total animation time
that had run before it. Each segment now carries the offset Rendering
measured for it.
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
class NarrationSegment:
    """One narration clip and where it belongs in the rendered video.

    start_time is the offset Rendering measured for the matching
    `self.wait(AUTO)` (CR-002 FR10.1) — not the running total of the
    preceding narration durations.
    """

    audio_path: str
    start_time: float


@dataclass(frozen=True)
class VideoAssemblyRequest:
    """Input to video assembly: one project's rendered (silent) video, its
    narration segments with their offsets, plus optional background music and
    subtitles.

    narration_segments is empty when the Creator disabled narration (CR-001) —
    the result is then a silent video, or one carrying background music alone.

    video_duration_seconds is Rendering's measured length of video_path. It is
    what assembly pads against so a final narration that runs past the last
    frame is not cut off (CR-002 FR10.6); 0.0 means "unknown", in which case no
    padding is attempted.
    """

    project_id: str
    video_path: str
    narration_segments: list[NarrationSegment]
    video_duration_seconds: float = 0.0
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
