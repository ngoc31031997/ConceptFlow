"""RenderScriptUseCase — renders one project's whole Manim script.

Replaces the former per-scene template rendering + batch orchestration:
the Manim-script input mode has exactly one script (and one output video)
per project, so there is nothing left to batch over.
"""

from __future__ import annotations

from adapters.storage.artifact_paths import compute_video_path, ensure_parent_dir, video_exists
from domain.errors import InvalidDurationError
from domain.models import ScriptRenderRequest, ScriptRenderResult
from domain.ports import ManimScriptRendererPort


class RenderScriptUseCase:
    """Orchestrates zero-trust validation, idempotency, and rendering for
    one project's script (mirrors the former RenderSceneUseCase's shape)."""

    def __init__(self, renderer: ManimScriptRendererPort) -> None:
        self._renderer = renderer

    def render(self, request: ScriptRenderRequest) -> ScriptRenderResult:
        self._validate(request)

        video_path = compute_video_path(request.project_id)

        if video_exists(video_path):
            # Idempotency: reuse the artifact from a prior call instead of
            # re-rendering.
            return ScriptRenderResult(video_path=video_path)

        ensure_parent_dir(video_path)
        self._renderer.render(request, video_path)
        return ScriptRenderResult(video_path=video_path)

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
