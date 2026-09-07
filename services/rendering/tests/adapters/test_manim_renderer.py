"""Unit tests for ManimScriptRenderer.

subprocess.run is monkeypatched in every test — these tests never actually
invoke the real `manim` CLI, only verify AUTO-wait substitution, subprocess
error mapping, and output-file discovery.
"""

from __future__ import annotations

import resource
import subprocess

import pytest

from adapters.rendering.manim_renderer import (
    DEFAULT_RENDER_MEMORY_LIMIT_GB,
    DEFAULT_RENDER_TIMEOUT_SECONDS,
    ManimScriptRenderer,
)
from domain.errors import AnimationEngineError
from domain.models import NarrationSegment, ScriptRenderRequest

VALID_SCRIPT = (
    "from manim import *\n\n"
    "class DemoScene(Scene):\n"
    "    def construct(self):\n"
    "        self.wait(AUTO)\n"
    "        self.wait(AUTO)\n"
)


def make_request(script: str = VALID_SCRIPT) -> ScriptRenderRequest:
    return ScriptRenderRequest(
        project_id="proj-1",
        script_content=script,
        scene_class_name="DemoScene",
        narration_segments=[
            NarrationSegment(scene_index=0, audio_path="/shared/proj-1/audio/0.wav", duration_seconds=2.5),
            NarrationSegment(scene_index=1, audio_path="/shared/proj-1/audio/1.wav", duration_seconds=3.0),
        ],
    )


def test_patch_auto_waits_substitutes_in_order():
    patched = ManimScriptRenderer._patch_auto_waits(VALID_SCRIPT, [2.5, 3.0])
    assert "self.wait(2.5)" in patched
    assert "self.wait(3.0)" in patched
    assert "AUTO" not in patched


def test_patch_auto_waits_raises_on_count_mismatch():
    with pytest.raises(AnimationEngineError, match="self.wait\\(AUTO\\)"):
        ManimScriptRenderer._patch_auto_waits(VALID_SCRIPT, [2.5])


def test_render_invokes_manim_and_moves_output(tmp_path, monkeypatch):
    renderer = ManimScriptRenderer()
    output_path = str(tmp_path / "out.mp4")

    def fake_run(cmd, **kwargs):
        media_dir = kwargs["cwd"]
        import os

        nested = os.path.join(media_dir, "videos")
        os.makedirs(nested, exist_ok=True)
        with open(os.path.join(nested, "DemoScene.mp4"), "wb") as f:
            f.write(b"stub-mp4-bytes")
        return subprocess.CompletedProcess(cmd, 0, stdout="", stderr="")

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.run", fake_run)

    renderer.render(make_request(), output_path)

    with open(output_path, "rb") as f:
        assert f.read() == b"stub-mp4-bytes"


def test_render_raises_on_nonzero_exit(tmp_path, monkeypatch):
    renderer = ManimScriptRenderer()

    def fake_run(cmd, **kwargs):
        return subprocess.CompletedProcess(cmd, 1, stdout="", stderr="Traceback: boom")

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.run", fake_run)

    with pytest.raises(AnimationEngineError, match="Manim render failed"):
        renderer.render(make_request(), str(tmp_path / "out.mp4"))


def test_render_raises_on_timeout(tmp_path, monkeypatch):
    renderer = ManimScriptRenderer(timeout_seconds=1)

    def fake_run(cmd, **kwargs):
        raise subprocess.TimeoutExpired(cmd, kwargs.get("timeout", 1))

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.run", fake_run)

    with pytest.raises(AnimationEngineError, match="timed out"):
        renderer.render(make_request(), str(tmp_path / "out.mp4"))


def test_find_rendered_file_locates_mp4(tmp_path):
    nested = tmp_path / "videos" / "1080p60"
    nested.mkdir(parents=True)
    (nested / "DemoScene.mp4").write_bytes(b"stub")

    found = ManimScriptRenderer._find_rendered_file(str(tmp_path))

    assert found == str(nested / "DemoScene.mp4")


def test_find_rendered_file_raises_when_missing(tmp_path):
    with pytest.raises(AnimationEngineError):
        ManimScriptRenderer._find_rendered_file(str(tmp_path))


def test_child_resource_limits_cap_memory_but_not_cpu_time(monkeypatch):
    """CR-003 FR11.3 regression.

    RLIMIT_CPU used to be set equal to the wall-clock timeout. That is wrong:
    Phase 0 benchmarking measured Manim burning CPU-time at 2.21x wall-clock
    (it renders on several cores), so the limit fired at roughly 45% of the
    configured timeout and killed legitimate long renders with SIGXCPU. Only
    the address-space cap may be set here.
    """
    renderer = ManimScriptRenderer(timeout_seconds=1800, memory_limit_gb=4)
    applied: dict[int, tuple[int, int]] = {}

    monkeypatch.setattr(
        "adapters.rendering.manim_renderer.resource.setrlimit",
        lambda which, limits: applied.__setitem__(which, limits),
    )

    renderer._limit_child_resources()

    assert resource.RLIMIT_CPU not in applied
    assert applied == {resource.RLIMIT_AS: (4 * 1024**3, 4 * 1024**3)}


def test_memory_limit_is_configurable(monkeypatch):
    renderer = ManimScriptRenderer(memory_limit_gb=2)
    applied: dict[int, tuple[int, int]] = {}

    monkeypatch.setattr(
        "adapters.rendering.manim_renderer.resource.setrlimit",
        lambda which, limits: applied.__setitem__(which, limits),
    )

    renderer._limit_child_resources()

    assert applied[resource.RLIMIT_AS] == (2 * 1024**3, 2 * 1024**3)


def test_defaults_are_sized_for_long_form_video():
    """Guards the Phase 0 numbers against being silently reverted to the
    demo-sized values (300s / 2 GiB) that could not render a 10-minute video."""
    assert DEFAULT_RENDER_TIMEOUT_SECONDS == 1800
    assert DEFAULT_RENDER_MEMORY_LIMIT_GB == 4
