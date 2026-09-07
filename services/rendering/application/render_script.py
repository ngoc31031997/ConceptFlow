"""RenderScriptUseCase — renders one project's whole Manim script.

Replaces the former per-scene template rendering + batch orchestration:
the Manim-script input mode has exactly one script (and one output video)
per project, so there is nothing left to batch over.
"""

from __future__ import annotations

from adapters.storage.artifact_paths import (
    compute_timing_path,
    compute_video_path,
    ensure_parent_dir,
    read_timing,
    video_exists,
    write_timing,
)
from domain.errors import InvalidDurationError
from domain.models import ScriptRenderRequest, ScriptRenderResult
from domain.ports import ManimScriptRendererPort


class RenderScriptUseCase:
    """Orchestrates zero-trust validation, idempotency, and rendering for
    one project's script (mirrors the former RenderSceneUseCase's shape)."""

    def __init__(self, renderer: ManimScriptRendererPort) -> None:
        self._renderer = renderer

    def set_heartbeat(self, callback) -> None:
        """Wires per-command progress reporting into the renderer.

        Set per command rather than at construction because the callback closes
        over the project_id being rendered, which is only known once a command
        arrives. A renderer with no such hook simply ignores this.
        """
        setter = getattr(self._renderer, "set_heartbeat", None)
        if setter is not None:
            setter(callback)

    def render(self, request: ScriptRenderRequest) -> ScriptRenderResult:
        self._validate(request)

        video_path = compute_video_path(request.project_id)
        timing_path = compute_timing_path(request.project_id)

        if video_exists(video_path):
            # Idempotency: reuse the artifact from a prior call instead of
            # re-rendering — but only when its timing sidecar is there too.
            # Reporting a video without offsets would desynchronise the whole
            # downstream assembly (CR-002), so a video whose timing is missing
            # (e.g. rendered before CR-002 shipped) is re-rendered instead.
            timing = read_timing(timing_path)
            if timing is not None:
                return ScriptRenderResult(
                    video_path=video_path,
                    wait_offsets=timing["wait_offsets"],
                    video_duration_seconds=timing.get("video_duration_seconds", 0.0),
                )

        ensure_parent_dir(video_path)
        result = self._renderer.render(request, video_path)
        write_timing(timing_path, result.wait_offsets, result.video_duration_seconds)
        return result

    @staticmethod
    def _validate(request: ScriptRenderRequest) -> None:
        if not request.project_id:
            raise ValueError("project_id must not be empty")
        if not request.scene_class_name:
            raise ValueError("scene_class_name must not be empty")
        if not request.script_content.strip():
            raise ValueError("script_content must not be empty")
        if not request.narration_segments:
            raise ValueError("narration_segments must not be empty")
        for segment in request.narration_segments:
            # audio_path may legitimately be absent (CR-001: narration
            # disabled) — duration_seconds is the only field Rendering
            # actually needs, to substitute into self.wait(AUTO).
            if segment.duration_seconds <= 0:
                raise InvalidDurationError(segment.duration_seconds)
