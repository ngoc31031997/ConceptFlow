"""RenderScriptUseCase — renders one project's whole script into one video."""

from __future__ import annotations

from adapters.storage.artifact_paths import (
    compute_timing_path,
    compute_video_path,
    ensure_parent_dir,
    read_timing,
    video_exists,
    write_timing,
)
from domain.errors import InvalidDurationError, UnsupportedEngineError
from domain.models import ScriptRenderRequest, ScriptRenderResult
from domain.ports import ScriptRendererPort


class RenderScriptUseCase:
    """Validates, reuses a finished render when there is one, and renders
    one project's script."""

    def __init__(self, renderer: ScriptRendererPort) -> None:
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
            # downstream assembly, so a video whose timing is missing
            # is re-rendered instead.
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
        if request.engine != "remotion":
            raise UnsupportedEngineError(request.engine)
        if not request.narration_segments:
            raise ValueError("narration_segments must not be empty")
        for segment in request.narration_segments:
            # audio_path may legitimately be absent (narration disabled) —
            # duration_seconds is the only field Rendering
            # actually needs, to time the shot that reads it.
            if segment.duration_seconds <= 0:
                raise InvalidDurationError(segment.duration_seconds)
