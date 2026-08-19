"""Unit tests for RenderSceneUseCase (business-rules.md Rule 1-5)."""

from __future__ import annotations

import os

import pytest

from application.render_scene import RenderSceneUseCase
from domain.errors import InvalidDurationError
from domain.models import SceneRenderRequest
from domain.ports import AnimationRendererPort


def _write_stub_mp4(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "wb") as f:
        f.write(b"stub-mp4-bytes")


class FakeAnimationRenderer(AnimationRendererPort):
    def __init__(self, duration_seconds: float = 5.0) -> None:
        self.calls: list[tuple[SceneRenderRequest, str]] = []
        self._duration_seconds = duration_seconds

    def render(self, request: SceneRenderRequest, output_path: str) -> float:
        self.calls.append((request, output_path))
        _write_stub_mp4(output_path)
        return self._duration_seconds


def make_request(**overrides) -> SceneRenderRequest:
    defaults = dict(
        project_id="proj-1",
        scene_index=0,
        narration_text="hello",
        illustration_hint=None,
        code_snippet=None,
        code_language=None,
        animation_template_id="concept_illustration",
        audio_path="/shared/proj-1/audio/0_en.wav",
        duration_seconds=5.0,
    )
    defaults.update(overrides)
    return SceneRenderRequest(**defaults)


@pytest.fixture
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def test_render_calls_renderer_and_returns_result(shared_volume_root, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.read_duration_seconds", lambda path: 5.0)
    renderer = FakeAnimationRenderer(duration_seconds=5.0)
    use_case = RenderSceneUseCase(renderer)

    result = use_case.render(make_request())

    assert result.duration_seconds == 5.0
    assert result.animation_path == str(shared_volume_root / "proj-1" / "animations" / "0.mp4")
    assert len(renderer.calls) == 1


def test_render_is_idempotent_when_file_exists(shared_volume_root, monkeypatch):
    monkeypatch.setattr("application.render_scene.read_duration_seconds", lambda path: 5.0)
    renderer = FakeAnimationRenderer()
    use_case = RenderSceneUseCase(renderer)
    request = make_request()

    first = use_case.render(request)
    second = use_case.render(request)

    assert len(renderer.calls) == 1
    assert second.animation_path == first.animation_path


def test_empty_project_id_raises(shared_volume_root):
    use_case = RenderSceneUseCase(FakeAnimationRenderer())

    with pytest.raises(ValueError, match="project_id"):
        use_case.render(make_request(project_id=""))


def test_negative_scene_index_raises(shared_volume_root):
    use_case = RenderSceneUseCase(FakeAnimationRenderer())

    with pytest.raises(ValueError, match="scene_index"):
        use_case.render(make_request(scene_index=-1))


def test_empty_narration_text_raises(shared_volume_root):
    use_case = RenderSceneUseCase(FakeAnimationRenderer())

    with pytest.raises(ValueError, match="narration_text"):
        use_case.render(make_request(narration_text="   "))


def test_empty_audio_path_raises(shared_volume_root):
    use_case = RenderSceneUseCase(FakeAnimationRenderer())

    with pytest.raises(ValueError, match="audio_path"):
        use_case.render(make_request(audio_path=""))


def test_non_positive_duration_raises_invalid_duration_error(shared_volume_root):
    use_case = RenderSceneUseCase(FakeAnimationRenderer())

    with pytest.raises(InvalidDurationError):
        use_case.render(make_request(duration_seconds=0))
