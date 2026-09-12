"""Unit tests for ManimScriptRenderer.

The subprocess layer is monkeypatched in every test — these tests never
actually invoke the real `manim` CLI, only verify the two-pass contract
(CR-018), timing-mark handling, subprocess error mapping, cache behaviour, and
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
    DEFAULT_RENDER_QUALITY,
    DEFAULT_RENDER_TIMEOUT_SECONDS,
    QUALITY_FLAGS,
    QUALITY_FPS,
    ManimScriptRenderer,
)
from domain.errors import AnimationEngineError
from domain.models import NarrationSegment, ScriptRenderRequest

VALID_SCRIPT = (
    "from conceptflow import *\n\n"
    "class DemoScene(ConceptFlowScene):\n"
    "    def construct(self):\n"
    '        self.narrate("dòng một")\n'
    '        self.narrate("dòng hai")\n'
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
            f.write(json.dumps({"kind": "mark", "index": 0, "t": 0.0}) + "\n")
            f.write(json.dumps({"kind": "mark", "index": 1, "t": 7.25}) + "\n")
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
            f.write(json.dumps({"kind": "mark", "index": 0, "t": 0.0}) + "\n")
            f.write(json.dumps({"kind": "mark", "index": 1, "t": 1.0}) + "\n")

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
            f.write(json.dumps({"kind": "mark", "index": 0, "t": 0.0}) + "\n")
            f.write(json.dumps({"kind": "mark", "index": 1, "t": 1.0}) + "\n")

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
    marks.write_text('{"kind": "mark", "index": 1, "t": 9.5}\n{"kind": "mark", "index": 0, "t": 0.0}\n')

    assert ManimScriptRenderer._read_wait_offsets(str(marks), expected=2) == [0.0, 9.5]


def test_read_wait_offsets_rejects_a_missing_mark(tmp_path):
    """The two passes disagreed about how many narration lines the script has.

    After CR-018 a loop or an `if` around narration is perfectly legal — both
    passes run the same code, so both see the same count. A mismatch therefore
    means something genuinely worse: the script is non-deterministic. Returning
    a partial list would put every later narration on the wrong offset, the
    exact bug CR-002 removes, so this has to fail loudly."""
    marks = tmp_path / "cf_marks.jsonl"
    marks.write_text('{"kind": "mark", "index": 0, "t": 0.0}\n')

    with pytest.raises(AnimationEngineError, match="not deterministic"):
        ManimScriptRenderer._read_wait_offsets(str(marks), expected=2)


def test_read_wait_offsets_rejects_a_duplicated_mark(tmp_path):
    marks = tmp_path / "cf_marks.jsonl"
    marks.write_text('{"kind": "mark", "index": 0, "t": 0.0}\n{"kind": "mark", "index": 0, "t": 4.0}\n')

    with pytest.raises(AnimationEngineError):
        ManimScriptRenderer._read_wait_offsets(str(marks), expected=2)


def test_read_wait_offsets_raises_when_file_absent(tmp_path):
    with pytest.raises(AnimationEngineError, match="no timing marks"):
        ManimScriptRenderer._read_wait_offsets(str(tmp_path / "nope.jsonl"), expected=1)


def test_read_layout_marks_collects_only_layout_records(tmp_path):
    """CR-021 FR58: the layout snapshots ride the same JSONL as the timing
    marks, so parsing has to pick them out by `kind` and leave the rest alone."""
    marks = tmp_path / "cf_marks.jsonl"
    marks.write_text(
        '{"kind": "mark", "index": 0, "t": 0.0}\n'
        '{"kind": "layout", "index": 0, "t": 0.0, "mobjects": '
        '[{"cls": "Text", "bbox": [-1.0, 1.0, 0.5, -0.5], "color": "#FFFFFF", "font_size": 36.0}]}\n'
        '{"kind": "mark", "index": 1, "t": 4.0}\n'
        '{"kind": "layout", "index": 1, "t": 4.0, "mobjects": []}\n'
    )

    layouts = ManimScriptRenderer._read_layout_marks(str(marks))

    assert [r["index"] for r in layouts] == [0, 1]
    assert layouts[0]["mobjects"][0]["font_size"] == 36.0
    # The record travels whole to QC — index and t included.
    assert layouts[1] == {"kind": "layout", "index": 1, "t": 4.0, "mobjects": []}


def test_read_layout_marks_is_empty_when_the_script_recorded_none(tmp_path):
    """Layout capture is best-effort upstream, so its absence must not be an
    error here: a video with no layout data still gets its audio scored."""
    marks = tmp_path / "cf_marks.jsonl"
    marks.write_text('{"kind": "mark", "index": 0, "t": 0.0}\n')

    assert ManimScriptRenderer._read_layout_marks(str(marks)) == []
    assert ManimScriptRenderer._read_layout_marks(str(tmp_path / "nope.jsonl")) == []


def test_read_clip_marks_collects_only_clip_records(tmp_path):
    """CR-007 FR19.2: clip selections ride the same JSONL, picked out by
    `kind` like layout marks already are."""
    marks = tmp_path / "cf_marks.jsonl"
    marks.write_text(
        '{"kind": "mark", "index": 0, "t": 0.0}\n'
        '{"kind": "clip", "name": "vi du chay that", "index": 0, '
        '"t_start": 1.0, "t_end": 5.0}\n'
        '{"kind": "mark", "index": 1, "t": 6.0}\n'
    )

    clips = ManimScriptRenderer._read_clip_marks(str(marks))

    assert clips == [
        {
            "kind": "clip",
            "name": "vi du chay that",
            "index": 0,
            "t_start": 1.0,
            "t_end": 5.0,
        }
    ]


def test_read_clip_marks_is_empty_when_the_script_recorded_none(tmp_path):
    """Most scripts never call `self.clip(...)` — an empty list here just
    means there is nothing for Video Assembly to derive vertical clips from."""
    marks = tmp_path / "cf_marks.jsonl"
    marks.write_text('{"kind": "mark", "index": 0, "t": 0.0}\n')

    assert ManimScriptRenderer._read_clip_marks(str(marks)) == []
    assert ManimScriptRenderer._read_clip_marks(str(tmp_path / "nope.jsonl")) == []


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


def test_default_quality_is_1080p60(tmp_path, monkeypatch):
    """CR-004 FR12.1. 720p30 was hardcoded, which is below what a monetized
    channel should publish and throws away Manim's main strength — smooth
    motion."""
    assert DEFAULT_RENDER_QUALITY == "1080p60"

    renderer = ManimScriptRenderer(cache_root=None)
    captured = {}

    def fake_popen(cmd, **kwargs):
        import json
        import os

        captured["cmd"] = cmd
        media_dir = kwargs["cwd"]
        nested = os.path.join(media_dir, "videos")
        os.makedirs(nested, exist_ok=True)
        with open(os.path.join(nested, "DemoScene.mp4"), "wb") as f:
            f.write(b"stub")
        with open(kwargs["env"]["CF_MARKS_PATH"], "w") as f:
            f.write(json.dumps({"kind": "mark", "index": 0, "t": 0.0}) + "\n")
            f.write(json.dumps({"kind": "mark", "index": 1, "t": 1.0}) + "\n")
        return FakePopen()

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.Popen", fake_popen)
    stub_ffprobe(monkeypatch)

    renderer.render(make_request(), str(tmp_path / "out.mp4"))

    assert "-qh" in captured["cmd"]
    assert "-qm" not in captured["cmd"]


def test_quality_is_configurable(tmp_path, monkeypatch):
    renderer = ManimScriptRenderer(cache_root=None, quality="720p30")
    captured = {}

    def fake_popen(cmd, **kwargs):
        import json
        import os

        captured["cmd"] = cmd
        media_dir = kwargs["cwd"]
        nested = os.path.join(media_dir, "videos")
        os.makedirs(nested, exist_ok=True)
        with open(os.path.join(nested, "DemoScene.mp4"), "wb") as f:
            f.write(b"stub")
        with open(kwargs["env"]["CF_MARKS_PATH"], "w") as f:
            f.write(json.dumps({"kind": "mark", "index": 0, "t": 0.0}) + "\n")
            f.write(json.dumps({"kind": "mark", "index": 1, "t": 1.0}) + "\n")
        return FakePopen()

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.Popen", fake_popen)
    stub_ffprobe(monkeypatch)

    renderer.render(make_request(), str(tmp_path / "out.mp4"))

    assert "-qm" in captured["cmd"]


def test_unknown_quality_is_rejected_at_construction():
    """Better to fail on startup than to silently render at the wrong quality
    for every video until someone notices."""
    with pytest.raises(ValueError, match="unknown render quality"):
        ManimScriptRenderer(quality="8k120")


def test_quality_flags_cover_the_documented_presets():
    assert set(QUALITY_FLAGS) == {"480p15", "720p30", "1080p60", "4k60"}


def test_quality_fps_covers_every_quality_flag():
    """QUALITY_FPS từng thiếu "480p15" — QUALITY_FLAGS cho phép chọn preset đó
    (Test/Draft ở RenderQualityPicker) nhưng render() tra QUALITY_FPS[quality]
    vô điều kiện, nên chọn preset này chỉ crash ở lượt render THẬT (KeyError),
    không bao giờ bị dry-run hay lint bắt trước. Hai dict phải luôn khớp key."""
    assert set(QUALITY_FPS) == set(QUALITY_FLAGS)


def test_per_project_quality_overrides_the_service_default(tmp_path, monkeypatch):
    """CR-004 FR12.6: a Creator checks content with a fast 720p30 draft, then
    renders the upload pass at 1080p60 — same project, different pass."""
    renderer = ManimScriptRenderer(cache_root=None, quality="1080p60")
    captured = {}

    def fake_popen(cmd, **kwargs):
        import json
        import os

        captured["cmd"] = cmd
        media_dir = kwargs["cwd"]
        nested = os.path.join(media_dir, "videos")
        os.makedirs(nested, exist_ok=True)
        with open(os.path.join(nested, "DemoScene.mp4"), "wb") as f:
            f.write(b"stub")
        with open(kwargs["env"]["CF_MARKS_PATH"], "w") as f:
            f.write(json.dumps({"kind": "mark", "index": 0, "t": 0.0}) + "\n")
            f.write(json.dumps({"kind": "mark", "index": 1, "t": 1.0}) + "\n")
        return FakePopen()

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.Popen", fake_popen)
    stub_ffprobe(monkeypatch)

    request = make_request()
    request = ScriptRenderRequest(
        project_id=request.project_id,
        script_content=request.script_content,
        scene_class_name=request.scene_class_name,
        narration_segments=request.narration_segments,
        render_quality="720p30",
    )
    renderer.render(request, str(tmp_path / "out.mp4"))

    assert "-qm" in captured["cmd"]


def test_unknown_per_project_quality_falls_back_to_the_default():
    """A bad value from an old or hand-edited payload should cost the Creator
    the wrong resolution, not the whole render."""
    renderer = ManimScriptRenderer(cache_root=None, quality="1080p60")

    assert renderer._resolve_quality("nonsense") == "1080p60"
    assert renderer._resolve_quality(None) == "1080p60"
    assert renderer._resolve_quality("4k60") == "4k60"


# --- CR-018: hai lượt render ---------------------------------------------------


def test_rounds_durations_onto_frame_boundaries():
    """Manim băm cache theo nội dung từng segment, nên thời lượng lẻ tới
    micro-giây làm đổi hash của mọi segment phía sau một chỉnh sửa nhỏ và ném
    đi đúng cái cache mà RENDER_CACHE_ROOT sinh ra để có (đo được: nhanh ~5 lần
    khi render lại)."""
    from adapters.rendering.manim_renderer import _round_to_frames

    assert _round_to_frames([2.5133333, 3.0016666], 60) == [2.516667, 3.0]
    assert _round_to_frames([1.0], 30) == [1.0]


def test_dry_run_collects_narration_beats_and_chapters(tmp_path, monkeypatch):
    import json
    import os

    renderer = ManimScriptRenderer(cache_root=None)
    captured: dict = {}

    def fake_popen(cmd, **kwargs):
        captured["cmd"] = cmd
        captured["env"] = kwargs["env"]
        with open(kwargs["env"]["CF_MARKS_PATH"], "w", encoding="utf-8") as f:
            for record in [
                {"kind": "beat", "index": 0, "id": "hook"},
                {"kind": "narration", "index": 0, "text": "dòng một"},
                {"kind": "chapter", "index": 1, "title": "Phần hai"},
                {"kind": "narration", "index": 1, "text": "dòng hai"},
            ]:
                f.write(json.dumps(record, ensure_ascii=False) + "\n")
        return FakePopen()

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.Popen", fake_popen)

    result = renderer.dry_run(make_request())

    assert result.narrations == ["dòng một", "dòng hai"]
    assert result.beats == [(0, "hook")]
    assert result.chapters == [(1, "Phần hai")]
    # Lượt dry không được ghi video ra đĩa, và phải chạy ở chất lượng thấp nhất.
    assert "--dry_run" in captured["cmd"]
    assert "-ql" in captured["cmd"]
    assert captured["env"]["CF_MODE"] == "dry"
    # Cùng mức cách ly như lượt thật (FR49.3): không rò credential nào.
    assert set(captured["env"]) == {
        "PATH", "HOME", "CF_MARKS_PATH", "PYTHONPATH", "CF_MODE",
    }
    assert not os.path.exists(os.path.join(str(tmp_path), "out.mp4"))


def test_dry_run_collects_clip_marks(tmp_path, monkeypatch):
    """Bug report (2026-09-12): a project that picked video_output_mode
    short/both with a script that never called self.clip(...) only found out
    "Chưa có clip nào" after TTS + render had already run — because dry_run()
    discarded "clip" records the same marks file already had. This locks the
    fix: clip_marks must come back from the dry pass, same shape as
    _read_clip_marks reads for the real render."""
    import json

    renderer = ManimScriptRenderer(cache_root=None)

    def fake_popen(cmd, **kwargs):
        with open(kwargs["env"]["CF_MARKS_PATH"], "w", encoding="utf-8") as f:
            for record in [
                {"kind": "narration", "index": 0, "text": "dòng một"},
                {
                    "kind": "clip", "name": "vi du", "index": 0,
                    "t_start": 1.0, "t_end": 5.0,
                },
            ]:
                f.write(json.dumps(record, ensure_ascii=False) + "\n")
        return FakePopen()

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.Popen", fake_popen)

    result = renderer.dry_run(make_request())

    assert result.clip_marks == [
        {"kind": "clip", "name": "vi du", "index": 0, "t_start": 1.0, "t_end": 5.0},
    ]


def test_dry_run_clip_marks_empty_when_script_never_calls_self_clip(tmp_path, monkeypatch):
    import json

    renderer = ManimScriptRenderer(cache_root=None)

    def fake_popen(cmd, **kwargs):
        with open(kwargs["env"]["CF_MARKS_PATH"], "w", encoding="utf-8") as f:
            f.write(json.dumps({"kind": "narration", "index": 0, "text": "x"}) + "\n")
        return FakePopen()

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.Popen", fake_popen)

    result = renderer.dry_run(make_request())

    assert result.clip_marks == []


def test_dry_run_fails_when_script_produces_no_narration(monkeypatch):
    renderer = ManimScriptRenderer(cache_root=None)

    def fake_popen(cmd, **kwargs):
        open(kwargs["env"]["CF_MARKS_PATH"], "w").close()
        return FakePopen()

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.Popen", fake_popen)

    with pytest.raises(AnimationEngineError, match="no narration"):
        renderer.dry_run(make_request())


def test_render_hands_durations_to_the_script_and_writes_it_unmodified(tmp_path, monkeypatch):
    """Trước CR-018 script bị viết lại trước khi chạy, nên thứ chạy không bao
    giờ đúng là thứ Creator viết — và thứ được lưu thậm chí không phải Python
    hợp lệ."""
    import json
    import os

    renderer = ManimScriptRenderer(cache_root=None)
    seen: dict = {}

    def fake_popen(cmd, **kwargs):
        media_dir = kwargs["cwd"]
        seen["script"] = open(os.path.join(media_dir, "script.py")).read()
        seen["durations"] = json.load(open(kwargs["env"]["CF_DURATIONS_PATH"]))
        seen["mode"] = kwargs["env"]["CF_MODE"]
        nested = os.path.join(media_dir, "videos")
        os.makedirs(nested, exist_ok=True)
        with open(os.path.join(nested, "DemoScene.mp4"), "wb") as f:
            f.write(b"stub")
        with open(kwargs["env"]["CF_MARKS_PATH"], "w") as f:
            f.write(json.dumps({"kind": "mark", "index": 0, "t": 0.0}) + "\n")
            f.write(json.dumps({"kind": "mark", "index": 1, "t": 7.25}) + "\n")
        return FakePopen()

    monkeypatch.setattr("adapters.rendering.manim_renderer.subprocess.Popen", fake_popen)
    stub_ffprobe(monkeypatch)

    renderer.render(make_request(), str(tmp_path / "out.mp4"))

    assert seen["script"] == VALID_SCRIPT
    assert seen["durations"] == [2.5, 3.0]
    assert seen["mode"] == "render"
