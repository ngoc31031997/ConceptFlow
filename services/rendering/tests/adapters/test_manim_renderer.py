"""Unit tests for ManimScriptRenderer.

The subprocess layer is monkeypatched in every test — these tests never
actually invoke the real `manim` CLI, only verify AUTO-wait substitution,
timing-mark handling, subprocess error mapping, cache behaviour, and
output-file discovery.

Manim is launched with Popen (so a long render can stream heartbeats), while
ffprobe still uses subprocess.run, so both are stubbed.
"""

from __future__ import annotations

import io
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


class FakePopen:
    """Stands in for the Manim child process, which is launched with Popen so a
    long render can stream heartbeats while it works."""

    def __init__(self, returncode: int = 0, stderr: str = "", hang: bool = False) -> None:
        self.returncode = returncode
        self.stderr = io.StringIO(stderr)
        self._hang = hang
        self.killed = False

    def wait(self, timeout=None):
        # A real process stops hanging once it has been killed, so the
        # post-kill reap must not raise again.
        if self._hang and not self.killed:
            raise subprocess.TimeoutExpired("manim", timeout or 0)
        return self.returncode

    def kill(self):
        self.killed = True


def stub_ffprobe(monkeypatch, duration: str = "21.5"):
    """ffprobe still goes through subprocess.run."""
    monkeypatch.setattr(
        "adapters.rendering.manim_renderer.subprocess.run",
        lambda cmd, **kw: subprocess.CompletedProcess(cmd, 0, stdout=f"{duration}\n", stderr=""),
    )


def test_render_invokes_manim_and_moves_output(tmp_path, monkeypatch):
    renderer = ManimScriptRenderer(cache_root=None)
    output_path = str(tmp_path / "out.mp4")

    def fake_popen(cmd, **kwargs):
        import json
        import os

        media_dir = kwargs["cwd"]
        nested = os.path.join(media_dir, "videos")
        os.makedirs(nested, exist_ok=True)
        with open(os.path.join(nested, "DemoScene.mp4"), "wb") as f:
            f.write(b"stub-mp4-bytes")
        # Stand in for what the patched script's _cf_mark would have written.
        with open(kwargs["env"]["CF_MARKS_PATH"], "w") as f:
            f.write(json.dumps({"index": 0, "t": 0.0}) + "\n")
            f.write(json.dumps({"index": 1, "t": 7.25}) + "\n")
        return FakePopen()

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.Popen", fake_popen)
    stub_ffprobe(monkeypatch)

    result = renderer.render(make_request(), output_path)

    with open(output_path, "rb") as f:
        assert f.read() == b"stub-mp4-bytes"
    assert result.wait_offsets == [0.0, 7.25]
    assert result.video_duration_seconds == 21.5


def test_render_raises_on_nonzero_exit(tmp_path, monkeypatch):
    renderer = ManimScriptRenderer(cache_root=None)

    monkeypatch.setattr(
        "adapters.rendering.manim_renderer.subprocess.Popen",
        lambda cmd, **kw: FakePopen(returncode=1, stderr="Traceback: boom"),
    )

    with pytest.raises(AnimationEngineError, match="Manim render failed"):
        renderer.render(make_request(), str(tmp_path / "out.mp4"))


def test_render_error_message_keeps_manim_stderr(tmp_path, monkeypatch):
    """stderr is streamed line by line for heartbeats, so it still has to be
    collected in full — otherwise a failing render reports nothing useful."""
    renderer = ManimScriptRenderer(cache_root=None)

    monkeypatch.setattr(
        "adapters.rendering.manim_renderer.subprocess.Popen",
        lambda cmd, **kw: FakePopen(returncode=1, stderr="line one\nNameError: nope\n"),
    )

    with pytest.raises(AnimationEngineError, match="NameError: nope"):
        renderer.render(make_request(), str(tmp_path / "out.mp4"))


def test_render_raises_on_timeout_and_kills_the_child(tmp_path, monkeypatch):
    renderer = ManimScriptRenderer(timeout_seconds=1, cache_root=None)
    child = FakePopen(hang=True)

    monkeypatch.setattr(
        "adapters.rendering.manim_renderer.subprocess.Popen", lambda cmd, **kw: child
    )

    with pytest.raises(AnimationEngineError, match="timed out"):
        renderer.render(make_request(), str(tmp_path / "out.mp4"))
    # Popen does not kill on timeout the way subprocess.run does, so a hung
    # Manim would otherwise keep burning CPU after the render gave up.
    assert child.killed


def test_heartbeat_reports_elapsed_time_and_animation_index(tmp_path, monkeypatch):
    """CR-003 FR11.4: a multi-minute render must show it is still alive."""
    import adapters.rendering.manim_renderer as mod

    beats: list[tuple[float, int | None]] = []
    renderer = ManimScriptRenderer(
        cache_root=None, on_heartbeat=lambda elapsed, idx: beats.append((elapsed, idx))
    )
    monkeypatch.setattr(mod, "HEARTBEAT_INTERVAL_SECONDS", 0.01)

    def fake_popen(cmd, **kwargs):
        import json
        import os
        import time

        media_dir = kwargs["cwd"]
        nested = os.path.join(media_dir, "videos")
        os.makedirs(nested, exist_ok=True)
        with open(os.path.join(nested, "DemoScene.mp4"), "wb") as f:
            f.write(b"stub")
        with open(kwargs["env"]["CF_MARKS_PATH"], "w") as f:
            f.write(json.dumps({"index": 0, "t": 0.0}) + "\n")
            f.write(json.dumps({"index": 1, "t": 1.0}) + "\n")

        class SlowPopen(FakePopen):
            def wait(self, timeout=None):
                time.sleep(0.06)
                return 0

        return SlowPopen(stderr="Animation 7: doing things\n")

    monkeypatch.setattr(mod.subprocess, "Popen", fake_popen)
    stub_ffprobe(monkeypatch)

    renderer.render(make_request(), str(tmp_path / "out.mp4"))

    assert beats, "expected at least one heartbeat during a slow render"
    assert beats[-1][1] == 7  # the animation index parsed out of Manim's stderr


def test_heartbeat_failure_does_not_fail_the_render(tmp_path, monkeypatch):
    """Progress is UX-only — a broken publisher must not cost the Creator a
    render that is otherwise succeeding."""
    import adapters.rendering.manim_renderer as mod

    def explode(_elapsed, _idx):
        raise RuntimeError("publisher down")

    renderer = ManimScriptRenderer(cache_root=None, on_heartbeat=explode)
    monkeypatch.setattr(mod, "HEARTBEAT_INTERVAL_SECONDS", 0.01)

    def fake_popen(cmd, **kwargs):
        import json
        import os
        import time

        media_dir = kwargs["cwd"]
        nested = os.path.join(media_dir, "videos")
        os.makedirs(nested, exist_ok=True)
        with open(os.path.join(nested, "DemoScene.mp4"), "wb") as f:
            f.write(b"stub")
        with open(kwargs["env"]["CF_MARKS_PATH"], "w") as f:
            f.write(json.dumps({"index": 0, "t": 0.0}) + "\n")
            f.write(json.dumps({"index": 1, "t": 1.0}) + "\n")

        class SlowPopen(FakePopen):
            def wait(self, timeout=None):
                time.sleep(0.05)
                return 0

        return SlowPopen()

    monkeypatch.setattr(mod.subprocess, "Popen", fake_popen)
    stub_ffprobe(monkeypatch)

    result = renderer.render(make_request(), str(tmp_path / "out.mp4"))
    assert result.wait_offsets == [0.0, 1.0]


def test_find_rendered_file_locates_mp4(tmp_path):
    nested = tmp_path / "videos" / "1080p60"
    nested.mkdir(parents=True)
    (nested / "DemoScene.mp4").write_bytes(b"stub")

    found = ManimScriptRenderer._find_rendered_file(str(tmp_path))

    assert found == str(nested / "DemoScene.mp4")


def test_find_rendered_file_ignores_cached_partial_movies(tmp_path):
    """partial_movie_files holds the per-animation cache segments, which are
    .mp4 files too. Returning one would ship a fraction of a second of
    animation as the whole video."""
    nested = tmp_path / "videos" / "1080p60"
    partials = nested / "partial_movie_files" / "DemoScene"
    partials.mkdir(parents=True)
    (partials / "0000.mp4").write_bytes(b"cached-segment")
    (nested / "DemoScene.mp4").write_bytes(b"real-output")

    found = ManimScriptRenderer._find_rendered_file(str(tmp_path))

    assert found == str(nested / "DemoScene.mp4")


def test_media_dir_is_reused_per_project_when_caching_is_on(tmp_path):
    """Manim 0.18 keeps its cache inside media_dir, so the directory has to
    survive between renders for caching to do anything at all."""
    renderer = ManimScriptRenderer(cache_root=str(tmp_path))

    first, ephemeral_first = renderer._media_dir_for("proj-1")
    second, _ = renderer._media_dir_for("proj-1")
    other, _ = renderer._media_dir_for("proj-2")

    assert first == second
    assert first != other
    assert ephemeral_first is False


def test_media_dir_is_a_throwaway_tempdir_when_caching_is_off(tmp_path):
    renderer = ManimScriptRenderer(cache_root=None)

    first, ephemeral = renderer._media_dir_for("proj-1")
    second, _ = renderer._media_dir_for("proj-1")

    assert first != second
    assert ephemeral is True


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


def test_cache_prune_evicts_oldest_projects_over_budget(tmp_path):
    """A persistent per-project media_dir is a cache; without a ceiling it
    grows until the shared volume fills."""
    import os
    import time

    root = tmp_path / "cache"
    root.mkdir()
    for name, age in [("old", 300), ("middle", 200), ("new", 100)]:
        d = root / name
        d.mkdir()
        (d / "blob.bin").write_bytes(b"x" * 1000)
        os.utime(d, (time.time() - age, time.time() - age))

    renderer = ManimScriptRenderer(cache_root=str(root), cache_budget_bytes=2500)
    renderer._media_dir_for("current")

    remaining = sorted(p.name for p in root.iterdir())
    # "old" is evicted to get back under budget; "current" is never a candidate.
    assert "old" not in remaining
    assert "middle" in remaining and "new" in remaining and "current" in remaining


def test_cache_prune_never_evicts_the_running_project(tmp_path):
    import os

    root = tmp_path / "cache"
    root.mkdir()
    victim = root / "proj-1"
    victim.mkdir()
    (victim / "blob.bin").write_bytes(b"x" * 10_000)

    renderer = ManimScriptRenderer(cache_root=str(root), cache_budget_bytes=1)
    media_dir, _ = renderer._media_dir_for("proj-1")

    assert os.path.isdir(media_dir)
