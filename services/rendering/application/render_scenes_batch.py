"""RenderScenesBatchUseCase — renders every scene in a project, fail-fast
on the first error (mirrors ClassifyScenesBatchUseCase/SynthesizeSpeechBatchUseCase).

Accepts on_scene_start/on_scene_rendered callbacks invoked immediately
before/after each scene render — the messaging layer (consumer.py) uses
these to enqueue the scene_render_started/scene_rendered Outbox events
with their own immediate commits, so progress is visible in real time
(Low-Level Design Question 9) without the application layer knowing
anything about Outbox/messaging.
"""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass

from application.render_scene import RenderSceneUseCase
from domain.errors import AnimationEngineError, InvalidDurationError, UnsupportedTemplateError
from domain.models import SceneRenderRequest, SceneRenderResult


@dataclass(frozen=True)
class BatchRenderSuccess:
    results: list[SceneRenderResult]


@dataclass(frozen=True)
class BatchRenderFailure:
    scene_index: int
    error_message: str


BatchRenderOutcome = BatchRenderSuccess | BatchRenderFailure


class RenderScenesBatchUseCase:
    def __init__(self, single_scene_use_case: RenderSceneUseCase) -> None:
        self._render_scene = single_scene_use_case

    def execute(
        self,
        requests: list[SceneRenderRequest],
        on_scene_start: Callable[[int], None] | None = None,
        on_scene_rendered: Callable[[int, SceneRenderResult], None] | None = None,
    ) -> BatchRenderOutcome:
        results: list[SceneRenderResult] = []
        for request in requests:
            if on_scene_start is not None:
                on_scene_start(request.scene_index)
            try:
                result = self._render_scene.render(request)
            except (ValueError, InvalidDurationError, UnsupportedTemplateError, AnimationEngineError) as exc:
                return BatchRenderFailure(scene_index=request.scene_index, error_message=str(exc))
            results.append(result)
            if on_scene_rendered is not None:
                on_scene_rendered(request.scene_index, result)
        return BatchRenderSuccess(results=results)
