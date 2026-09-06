"""Unit tests for RenderScriptUseCase."""

from __future__ import annotations

import pytest

from application.render_script import RenderScriptUseCase
from domain.errors import InvalidDurationError
from domain.models import NarrationSegment, ScriptRenderRequest, ScriptRenderResult
from domain.ports import ManimScriptRendererPort


class FakeRenderer(ManimScriptRendererPort):
    def __init__(self) -> None:
        self.calls: list[tuple[ScriptRenderRequest, str]] = []

    def render(self, request: ScriptRenderRequest, output_path: str) -> None:
        self.calls.append((request, output_path))
        import os

        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, "wb") as f:
            f.write(b"stub")


def make_request(project_id: str = "proj-1") -> ScriptRenderRequest:
    return ScriptRenderRequest(
        project_id=project_id,
        script_content="class DemoScene(Scene):\n    def construct(self):\n        self.wait(AUTO)\n",
        scene_class_name="DemoScene",
        narration_segments=[NarrationSegment(scene_index=0, audio_path="/shared/proj-1/audio/0.wav", duration_seconds=2.0)],
    )


@pytest.fixture(autouse=True)
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def test_render_delegates_and_returns_video_path():
    renderer = FakeRenderer()
    use_case = RenderScriptUseCase(renderer)

    result = use_case.render(make_request())

    assert isinstance(result, ScriptRenderResult)
    assert result.video_path.endswith("rendered.mp4")
    assert len(renderer.calls) == 1


def test_render_is_idempotent_when_video_already_exists():
    renderer = FakeRenderer()
    use_case = RenderScriptUseCase(renderer)

    use_case.render(make_request())
    use_case.render(make_request())

    assert len(renderer.calls) == 1


def test_render_rejects_empty_project_id():
    renderer = FakeRenderer()
    use_case = RenderScriptUseCase(renderer)

    request = ScriptRenderRequest(
        project_id="",
        script_content="x",
        scene_class_name="DemoScene",
        narration_segments=[NarrationSegment(scene_index=0, audio_path="a.wav", duration_seconds=1.0)],
    )
    with pytest.raises(ValueError):
        use_case.render(request)


def test_render_rejects_non_positive_duration():
    renderer = FakeRenderer()
    use_case = RenderScriptUseCase(renderer)

    request = ScriptRenderRequest(
        project_id="proj-1",
        script_content="x",
        scene_class_name="DemoScene",
        narration_segments=[NarrationSegment(scene_index=0, audio_path="a.wav", duration_seconds=0)],
    )
    with pytest.raises(InvalidDurationError):
        use_case.render(request)


def test_render_rejects_empty_narration_segments():
    renderer = FakeRenderer()
    use_case = RenderScriptUseCase(renderer)

    request = ScriptRenderRequest(
        project_id="proj-1", script_content="x", scene_class_name="DemoScene", narration_segments=[]
    )
    with pytest.raises(ValueError):
        use_case.render(request)
