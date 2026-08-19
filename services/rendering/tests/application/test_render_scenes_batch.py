"""Unit tests for RenderScenesBatchUseCase (fail-fast batch + callbacks)."""

from __future__ import annotations

import os

import pytest

from application.render_scene import RenderSceneUseCase
from application.render_scenes_batch import (
    BatchRenderFailure,
    BatchRenderSuccess,
    RenderScenesBatchUseCase,
)
from domain.errors import AnimationEngineError
from domain.models import SceneRenderRequest
from domain.ports import AnimationRendererPort


def _write_stub_mp4(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "wb") as f:
        f.write(b"stub-mp4-bytes")


class FakeAnimationRenderer(AnimationRendererPort):
    def __init__(self, fail_at_scene: int | None = None) -> None:
        self.calls: list[int] = []
        self._fail_at_scene = fail_at_scene

    def render(self, request: SceneRenderRequest, output_path: str) -> float:
        if request.scene_index == self._fail_at_scene:
            raise AnimationEngineError("engine crashed")
        self.calls.append(request.scene_index)
        _write_stub_mp4(output_path)
        return 5.0


def make_request(scene_index: int) -> SceneRenderRequest:
    return SceneRenderRequest(
        project_id="proj-1",
        scene_index=scene_index,
        narration_text=f"scene {scene_index}",
        illustration_hint=None,
        code_snippet=None,
        code_language=None,
        animation_template_id="concept_illustration",
        audio_path=f"/shared/proj-1/audio/{scene_index}_en.wav",
        duration_seconds=5.0,
    )


@pytest.fixture
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def test_batch_renders_every_scene(shared_volume_root):
    engine = FakeAnimationRenderer()
    batch_use_case = RenderScenesBatchUseCase(RenderSceneUseCase(engine))

    outcome = batch_use_case.execute([make_request(0), make_request(1)])

    assert isinstance(outcome, BatchRenderSuccess)
    assert len(outcome.results) == 2
    assert engine.calls == [0, 1]


def test_batch_fails_fast_and_does_not_attempt_later_scenes(shared_volume_root):
    engine = FakeAnimationRenderer(fail_at_scene=1)
    batch_use_case = RenderScenesBatchUseCase(RenderSceneUseCase(engine))

    outcome = batch_use_case.execute([make_request(0), make_request(1), make_request(2)])

    assert isinstance(outcome, BatchRenderFailure)
    assert outcome.scene_index == 1
    assert engine.calls == [0]  # scene 2 never attempted


def test_batch_invokes_start_and_rendered_callbacks_in_order(shared_volume_root):
    engine = FakeAnimationRenderer()
    batch_use_case = RenderScenesBatchUseCase(RenderSceneUseCase(engine))
    events: list[tuple[str, int]] = []

    batch_use_case.execute(
        [make_request(0), make_request(1)],
        on_scene_start=lambda i: events.append(("start", i)),
        on_scene_rendered=lambda i, result: events.append(("rendered", i)),
    )

    assert events == [("start", 0), ("rendered", 0), ("start", 1), ("rendered", 1)]


def test_batch_does_not_call_rendered_callback_on_failure(shared_volume_root):
    engine = FakeAnimationRenderer(fail_at_scene=0)
    batch_use_case = RenderScenesBatchUseCase(RenderSceneUseCase(engine))
    events: list[tuple[str, int]] = []

    batch_use_case.execute(
        [make_request(0)],
        on_scene_start=lambda i: events.append(("start", i)),
        on_scene_rendered=lambda i, result: events.append(("rendered", i)),
    )

    assert events == [("start", 0)]
