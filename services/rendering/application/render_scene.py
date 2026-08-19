"""RenderSceneUseCase — business-logic-model.md."""

from __future__ import annotations

from adapters.storage.artifact_paths import (
    animation_exists,
    compute_animation_path,
    ensure_parent_dir,
    read_duration_seconds,
)
from domain.errors import InvalidDurationError
from domain.models import SceneRenderRequest, SceneRenderResult
from domain.ports import AnimationRendererPort


class RenderSceneUseCase:
    """Orchestrates zero-trust validation, idempotency, and rendering for
    one scene (Functional Design Business Rules 1 and 5)."""

    def __init__(self, renderer: AnimationRendererPort) -> None:
        self._renderer = renderer

    def render(self, request: SceneRenderRequest) -> SceneRenderResult:
        self._validate(request)

        animation_path = compute_animation_path(request.project_id, request.scene_index)

        if animation_exists(animation_path):
            # Idempotency (Business Rule 5): reuse the artifact from a prior
            # call instead of re-rendering.
            return SceneRenderResult(
                animation_path=animation_path,
                duration_seconds=read_duration_seconds(animation_path),
            )

        ensure_parent_dir(animation_path)
        duration_seconds = self._renderer.render(request, animation_path)
        return SceneRenderResult(animation_path=animation_path, duration_seconds=duration_seconds)

    @staticmethod
    def _validate(request: SceneRenderRequest) -> None:
        """Zero-trust validation (Business Rule 1) — never trust upstream
        data, even though it already passed through prior Saga steps.

        animation_template_id validity is checked by the renderer adapter
        (which owns the AnimationTemplateRegistry) rather than here, to
        avoid the application layer depending on registry internals.
        """
        if not request.project_id:
            raise ValueError("project_id must not be empty")
        if request.scene_index < 0:
            raise ValueError("scene_index must not be negative")
        if not request.narration_text.strip():
            raise ValueError("narration_text must not be empty")
        if not request.audio_path:
            raise ValueError("audio_path must not be empty")
        if request.duration_seconds <= 0:
            raise InvalidDurationError(request.duration_seconds)
