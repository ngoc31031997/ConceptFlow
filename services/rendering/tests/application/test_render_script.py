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
        self.dry_runs: list[ScriptRenderRequest] = []

    def dry_run(self, request: ScriptRenderRequest):
        from domain.models import DryRunResult

        self.dry_runs.append(request)
        return DryRunResult(narrations=["dòng một", "dòng hai"])

    def render(self, request: ScriptRenderRequest, output_path: str) -> ScriptRenderResult:
        self.calls.append((request, output_path))
        import os

        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, "wb") as f:
            f.write(b"stub")
        offsets = [float(i) * 5.0 for i in range(len(request.narration_segments))]
        return ScriptRenderResult(
            video_path=output_path,
            wait_offsets=offsets,
            video_duration_seconds=offsets[-1] + 10.0 if offsets else 0.0,
        )


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
    assert result.wait_offsets == [0.0]
    assert result.video_duration_seconds == 10.0
    assert len(renderer.calls) == 1


def test_render_is_idempotent_when_video_already_exists():
    renderer = FakeRenderer()
    use_case = RenderScriptUseCase(renderer)

    first = use_case.render(make_request())
    second = use_case.render(make_request())

    assert len(renderer.calls) == 1
    # CR-002: the fast path must reproduce the timing too, not just the path —
    # an offset-less "completed" event would desynchronise assembly.
    assert second.wait_offsets == first.wait_offsets
    assert second.video_duration_seconds == first.video_duration_seconds


def test_render_redoes_work_when_timing_sidecar_is_missing(shared_volume_root):
    """A video rendered before CR-002 has no timing.json. Reusing it would
    emit an event with no offsets, so the use case must re-render instead."""
    import os

    renderer = FakeRenderer()
    use_case = RenderScriptUseCase(renderer)

    use_case.render(make_request())
    os.remove(os.path.join(str(shared_volume_root), "proj-1", "video", "timing.json"))

    result = use_case.render(make_request())

    assert len(renderer.calls) == 2
    assert result.wait_offsets == [0.0]


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


def test_renders_when_audio_path_is_absent():
    # CR-001: narration disabled means no audio file exists at all — only the
    # estimated duration_seconds is used to substitute self.wait(AUTO).
    renderer = FakeRenderer()
    use_case = RenderScriptUseCase(renderer)
    request = ScriptRenderRequest(
        project_id="proj-1",
        script_content="class DemoScene(Scene):\n    def construct(self):\n        self.wait(AUTO)\n",
        scene_class_name="DemoScene",
        narration_segments=[NarrationSegment(scene_index=0, duration_seconds=2.0)],
    )

    result = use_case.render(request)

    assert isinstance(result, ScriptRenderResult)
    assert len(renderer.calls) == 1
