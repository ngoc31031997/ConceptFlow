"""Domain value objects for the Rendering Service (Manim-script input mode).

Rendering no longer renders one pre-built template per narration scene — it
executes the Creator's own Manim script once for the whole project,
substituting each `self.wait(AUTO)` call (in order) with the real TTS
audio duration for the corresponding "# NARRATION: ..." marker, so the
animation's pacing stays in lockstep with the voiceover.
"""

from __future__ import annotations

from dataclasses import dataclass, field


@dataclass(frozen=True)
class NarrationSegment:
    """One "# NARRATION: ..." marker's timing, in scene_index (i.e. script
    order) — the i-th segment's duration_seconds replaces the i-th
    `self.wait(AUTO)` call in the script.

    audio_path is None when the Creator disabled narration (CR-001):
    duration_seconds is then an estimate from the narration text rather than
    a real audio file's length, but Rendering never reads the audio itself
    either way — only Video Assembly does.
    """

    scene_index: int
    duration_seconds: float
    audio_path: str | None = None


@dataclass(frozen=True)
class ScriptRenderRequest:
    """Input to whole-script rendering. Zero-trust validated by
    RenderScriptUseCase — the service never trusts upstream data."""

    project_id: str
    script_content: str
    scene_class_name: str
    narration_segments: list[NarrationSegment]


@dataclass(frozen=True)
class ScriptRenderResult:
    """Where the silent video landed, plus the timing Video Assembly needs to
    line narration up with it (CR-002 FR3.5).

    wait_offsets[i] is the second, measured from the start of the video, at
    which the i-th `self.wait(AUTO)` begins — i.e. where narration segment i
    must start playing. It is NOT the running sum of narration durations: the
    animation between narrations pushes every later segment further out.
    """

    video_path: str
    wait_offsets: list[float] = field(default_factory=list)
    video_duration_seconds: float = 0.0
