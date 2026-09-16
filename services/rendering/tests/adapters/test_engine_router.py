"""EngineRouterRenderer — feature/remotion-engine."""

from __future__ import annotations

import pytest
from adapters.rendering.engine_router import EngineRouterRenderer
from domain.errors import AnimationEngineError
from domain.models import DryRunResult, ScriptRenderRequest, ScriptRenderResult


class FakeRenderer:
    def __init__(self, name: str) -> None:
        self.name = name
        self.dry_run_calls = 0
        self.render_calls = 0
        self.heartbeat = None

    def dry_run(self, request):
        self.dry_run_calls += 1
        return DryRunResult(narrations=[self.name])

    def render(self, request, output_path):
        self.render_calls += 1
        return ScriptRenderResult(video_path=output_path)

    def set_heartbeat(self, callback):
        self.heartbeat = callback


def make_request(engine: str) -> ScriptRenderRequest:
    return ScriptRenderRequest(
        project_id="p1",
        script_content="x",
        scene_class_name="x",
        narration_segments=[],
        engine=engine,
    )


def test_routes_dry_run_to_manim():
    manim, remotion = FakeRenderer("manim"), FakeRenderer("remotion")
    router = EngineRouterRenderer(manim=manim, remotion=remotion)
    result = router.dry_run(make_request("manim"))
    assert result.narrations == ["manim"]
    assert manim.dry_run_calls == 1
    assert remotion.dry_run_calls == 0


def test_routes_render_to_remotion():
    manim, remotion = FakeRenderer("manim"), FakeRenderer("remotion")
    router = EngineRouterRenderer(manim=manim, remotion=remotion)
    router.render(make_request("remotion"), "/out.mp4")
    assert remotion.render_calls == 1
    assert manim.render_calls == 0


def test_defaults_to_manim_when_engine_empty():
    manim, remotion = FakeRenderer("manim"), FakeRenderer("remotion")
    router = EngineRouterRenderer(manim=manim, remotion=remotion)
    router.dry_run(make_request(""))
    assert manim.dry_run_calls == 1


def test_raises_on_unknown_engine():
    router = EngineRouterRenderer(manim=FakeRenderer("manim"), remotion=FakeRenderer("remotion"))
    with pytest.raises(AnimationEngineError, match="unknown render engine"):
        router.dry_run(make_request("blender"))


def test_set_heartbeat_forwards_to_every_renderer():
    manim, remotion = FakeRenderer("manim"), FakeRenderer("remotion")
    router = EngineRouterRenderer(manim=manim, remotion=remotion)

    def callback(elapsed, index):
        pass

    router.set_heartbeat(callback)
    assert manim.heartbeat is callback
    assert remotion.heartbeat is callback
