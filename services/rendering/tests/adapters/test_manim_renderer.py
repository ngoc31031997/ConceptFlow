"""Unit tests for ManimAnimationRenderer.

_render_to_file (the only place that actually touches Manim) is
monkeypatched in every test, so these tests never import the real manim
package — they verify template lookup, threadpool/timeout behavior, and
error mapping only.
"""

from __future__ import annotations

import time

import pytest

from adapters.rendering.manim_renderer import ManimAnimationRenderer
from adapters.rendering.registry import AnimationTemplateRegistry
from domain.errors import AnimationEngineError, UnsupportedTemplateError
from domain.models import SceneRenderRequest
from domain.ports import AnimationTemplatePort


class FakeTemplate(AnimationTemplatePort):
    @property
    def template_id(self) -> str:
        return "concept_illustration"

    def build_scene(self, request: SceneRenderRequest):
        return object()


def make_request() -> SceneRenderRequest:
    return SceneRenderRequest(
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


def test_unsupported_template_raises_immediately(tmp_path):
    registry = AnimationTemplateRegistry([])
    renderer = ManimAnimationRenderer(registry)

    with pytest.raises(UnsupportedTemplateError):
        renderer.render(make_request(), str(tmp_path / "out.mp4"))


def test_successful_render_returns_measured_duration(tmp_path, monkeypatch):
    registry = AnimationTemplateRegistry([FakeTemplate()])
    renderer = ManimAnimationRenderer(registry)
    output_path = str(tmp_path / "out.mp4")

    def fake_render_to_file(template, request, output_path):
        with open(output_path, "wb") as f:
            f.write(b"stub-mp4-bytes")

    monkeypatch.setattr(ManimAnimationRenderer, "_render_to_file", staticmethod(fake_render_to_file))
    monkeypatch.setattr("adapters.rendering.manim_renderer.read_duration_seconds", lambda path: 5.0)

    duration = renderer.render(make_request(), output_path)

    assert duration == 5.0


def test_engine_exception_becomes_animation_engine_error(tmp_path, monkeypatch):
    registry = AnimationTemplateRegistry([FakeTemplate()])
    renderer = ManimAnimationRenderer(registry)

    def fake_render_to_file(template, request, output_path):
        raise RuntimeError("manim crashed")

    monkeypatch.setattr(ManimAnimationRenderer, "_render_to_file", staticmethod(fake_render_to_file))

    with pytest.raises(AnimationEngineError):
        renderer.render(make_request(), str(tmp_path / "out.mp4"))


def test_timeout_becomes_animation_engine_error(tmp_path, monkeypatch):
    registry = AnimationTemplateRegistry([FakeTemplate()])
    renderer = ManimAnimationRenderer(registry, timeout_seconds=0)

    def slow_render_to_file(template, request, output_path):
        time.sleep(0.5)

    monkeypatch.setattr(ManimAnimationRenderer, "_render_to_file", staticmethod(slow_render_to_file))

    with pytest.raises(AnimationEngineError, match="timed out"):
        renderer.render(make_request(), str(tmp_path / "out.mp4"))


def test_find_rendered_file_locates_mp4(tmp_path):
    nested = tmp_path / "videos" / "1080p60"
    nested.mkdir(parents=True)
    (nested / "scene.mp4").write_bytes(b"stub")

    found = ManimAnimationRenderer._find_rendered_file(str(tmp_path))

    assert found == str(nested / "scene.mp4")


def test_find_rendered_file_raises_when_missing(tmp_path):
    with pytest.raises(AnimationEngineError):
        ManimAnimationRenderer._find_rendered_file(str(tmp_path))
