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


def test_patch_auto_waits_marks_each_wait_with_its_index():
    """CR-002: each wait is wrapped so the render reports where it begins."""
    patched = ManimScriptRenderer._patch_auto_waits(VALID_SCRIPT, [2.5, 3.0])

    assert "(_cf_mark(self, 0), self.wait(2.5))" in patched
    assert "(_cf_mark(self, 1), self.wait(3.0))" in patched


def test_patch_auto_waits_puts_preamble_after_the_manim_import():
    """A `from manim import *` after the helper would shadow it."""
    patched = ManimScriptRenderer._patch_auto_waits(VALID_SCRIPT, [2.5, 3.0])
    lines = patched.splitlines()

    import_line = next(i for i, ln in enumerate(lines) if ln.startswith("from manim import"))
    helper_line = next(i for i, ln in enumerate(lines) if "def _cf_mark" in ln)

    assert helper_line > import_line


def test_patch_auto_waits_output_is_valid_python():
    compile(ManimScriptRenderer._patch_auto_waits(VALID_SCRIPT, [2.5, 3.0]), "<patched>", "exec")


def test_patch_auto_waits_raises_on_count_mismatch():
    with pytest.raises(AnimationEngineError, match="self.wait\\(AUTO\\)"):
        ManimScriptRenderer._patch_auto_waits(VALID_SCRIPT, [2.5])


def test_render_invokes_manim_and_moves_output(tmp_path, monkeypatch):
    renderer = ManimScriptRenderer()
    output_path = str(tmp_path / "out.mp4")

    def fake_run(cmd, **kwargs):
        import json
        import os

        # ffprobe, standing in for the duration probe.
        if cmd[0] == "ffprobe":
            return subprocess.CompletedProcess(cmd, 0, stdout="21.5\n", stderr="")

        media_dir = kwargs["cwd"]
        nested = os.path.join(media_dir, "videos")
        os.makedirs(nested, exist_ok=True)
        with open(os.path.join(nested, "DemoScene.mp4"), "wb") as f:
            f.write(b"stub-mp4-bytes")
        # Stand in for what the patched script's _cf_mark would have written.
        with open(kwargs["env"]["CF_MARKS_PATH"], "w") as f:
            f.write(json.dumps({"index": 0, "t": 0.0}) + "\n")
            f.write(json.dumps({"index": 1, "t": 7.25}) + "\n")
        return subprocess.CompletedProcess(cmd, 0, stdout="", stderr="")

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.run", fake_run)

    result = renderer.render(make_request(), output_path)

    with open(output_path, "rb") as f:
        assert f.read() == b"stub-mp4-bytes"
    assert result.wait_offsets == [0.0, 7.25]
    assert result.video_duration_seconds == 21.5


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


def test_read_wait_offsets_returns_marks_in_scene_order(tmp_path):
    marks = tmp_path / "cf_marks.jsonl"
    # Written in whatever order the render happened to flush them.
    marks.write_text('{"index": 1, "t": 9.5}\n{"index": 0, "t": 0.0}\n')

    assert ManimScriptRenderer._read_wait_offsets(str(marks), expected=2) == [0.0, 9.5]


def test_read_wait_offsets_rejects_a_missing_mark(tmp_path):
    """A wait inside a loop or an `if` fires a different number of times than
    there are narration segments. Returning a partial list would put every
    later narration on the wrong offset — the exact bug CR-002 removes — so
    this has to fail loudly."""
    marks = tmp_path / "cf_marks.jsonl"
    marks.write_text('{"index": 0, "t": 0.0}\n')

    with pytest.raises(AnimationEngineError, match="exactly once"):
        ManimScriptRenderer._read_wait_offsets(str(marks), expected=2)


def test_read_wait_offsets_rejects_a_duplicated_mark(tmp_path):
    marks = tmp_path / "cf_marks.jsonl"
    marks.write_text('{"index": 0, "t": 0.0}\n{"index": 0, "t": 4.0}\n')

    with pytest.raises(AnimationEngineError):
        ManimScriptRenderer._read_wait_offsets(str(marks), expected=2)


def test_read_wait_offsets_raises_when_file_absent(tmp_path):
    with pytest.raises(AnimationEngineError, match="no timing marks"):
        ManimScriptRenderer._read_wait_offsets(str(tmp_path / "nope.jsonl"), expected=1)
