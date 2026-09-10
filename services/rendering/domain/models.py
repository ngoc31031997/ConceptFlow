"""Domain value objects for the Rendering Service (Manim-script input mode).

Rendering executes the Creator's own Manim script twice per project (CR-018):
a dry pass that collects the narration lines in the order they actually run,
and — once those lines have been synthesized — a real pass where each
`self.narrate(...)` waits for exactly as long as its audio.
"""

from __future__ import annotations

from dataclasses import dataclass, field


@dataclass(frozen=True)
class NarrationSegment:
    """One narration line's timing, in scene_index (i.e. the order the dry pass
    saw them run) — the i-th segment's duration_seconds is how long the i-th
    `self.narrate(...)` call holds the animation.

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
    # CR-004 FR12.6: the Creator picks this per project (a fast 720p30 draft to
    # check the content, then a 1080p60 pass for upload). None means "use the
    # service default", which covers projects created before the field existed.
    render_quality: str | None = None


@dataclass(frozen=True)
class DryRunResult:
    """What the dry pass learned by running the script without rendering it.

    `narrations` is in **runtime order**, not file order. That distinction is
    the whole point of CR-018: a `self.narrate(...)` inside a loop or a helper
    contributes exactly as many lines as it really produces, which reading
    `# NARRATION:` comments out of the source could never get right.

    `beats` and `chapters` carry (narration_index, value): each marker attaches
    to the narration line that follows it, so its timestamp is the real offset
    the render pass measures rather than an estimate (CR-006 FR15).
    """

    narrations: list[str]
    #: CR-024 FR68.5 — mô tả ngắn khung hình tại mỗi lời thoại, cùng thứ tự với
    #: `narrations`. Dùng cho màn duyệt dàn ý, không ảnh hưởng gì tới render.
    visuals: list[str] = field(default_factory=list)
    beats: list[tuple[int, str]] = field(default_factory=list)
    chapters: list[tuple[int, str]] = field(default_factory=list)


@dataclass(frozen=True)
class ScriptRenderResult:
    """Where the silent video landed, plus the timing Video Assembly needs to
    line narration up with it (CR-002 FR3.5).

    wait_offsets[i] is the second, measured from the start of the video, at
    which the i-th `self.narrate(...)` begins waiting — i.e. where narration
    segment i must start playing. It is NOT the running sum of narration durations: the
    animation between narrations pushes every later segment further out.
    """

    video_path: str
    wait_offsets: list[float] = field(default_factory=list)
    video_duration_seconds: float = 0.0
