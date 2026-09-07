"""ManimScriptRenderer — implements ManimScriptRendererPort.

Runs the Creator's own Manim script (not a pre-built template) as a
**subprocess**, never in-process — this is untrusted, hand-written code, so
it must never run inside the Rendering Service's own Python process (which
holds RabbitMQ/Postgres credentials in its environment). Guardrails applied
to the subprocess:

- A stripped environment (no RABBITMQ_URL/DATABASE_URL/etc. — only PATH/HOME).
- A wall-clock timeout (RENDER_TIMEOUT_SECONDS, default 1800s).
- An address-space (memory) resource limit via `resource.setrlimit`, applied in
  the child before exec via `preexec_fn`.

There is deliberately **no** RLIMIT_CPU (CR-003 FR11.3). It used to be set equal
to the wall-clock timeout, which was wrong: benchmarking measured Manim burning
CPU-time at 2.21x wall-clock (it renders on multiple cores), so that limit fired
at roughly 45% of the time the config claimed to allow and killed legitimate
renders with SIGXCPU well before the timeout. The wall-clock timeout on
`subprocess.run` is the correct and sufficient bound.

This is deliberate, bounded hardening for a single-Creator tool — not a full
sandbox (no seccomp/container-per-render/network isolation). It assumes
scripts are authored by the Creator themselves, not submitted by untrusted
third parties (ADR pending).

Every `self.wait(AUTO)` call in the script is textually substituted (in
order) with the real TTS-measured duration for the corresponding
"# NARRATION: ..." marker before the subprocess ever runs — the raw
script_content as stored is not valid Python (`AUTO` is not a real name)
until this substitution happens.

Each substitution also records **where in the finished video that wait
begins** (CR-002 FR10.1). This matters because the video's timeline is

    video_duration = sum(self.play(...) durations) + sum(self.wait(...))

so narration i does not start at the sum of the preceding narration
durations — it starts after all the animation that ran before it too.
Video Assembly needs those real offsets to place each audio segment;
without them the narration runs ahead of the picture by the accumulated
animation time (measured at 61.6s on a 3.6-minute reference video before
this was fixed).

The mechanism is still pure text substitution — this service never
executes the Creator's script itself. `self.wait(AUTO)` becomes
`(_cf_mark(self, i), self.wait(D))`, and a small preamble defines
`_cf_mark` to append `scene.renderer.time` to a JSONL file. Python
evaluates tuple elements left to right, so the mark is taken *before* the
wait — i.e. it is the wait's start, which is exactly what `adelay` needs
downstream.
"""

from __future__ import annotations

import json
import logging
import os
import re
import resource
import shutil
import subprocess
import tempfile
import threading
import time
from collections.abc import Callable

from domain.errors import AnimationEngineError
from domain.models import ScriptRenderRequest, ScriptRenderResult
from domain.ports import ManimScriptRendererPort

logger = logging.getLogger(__name__)

# Sized from the Phase 0 benchmark (long-form-baseline.md): a 215s 1080p60
# render took 98.9s wall-clock and peaked at 764 MB, i.e. roughly 0.46x the
# video's duration. A 10-minute video therefore lands near 276s; 1800s leaves
# ~6.5x headroom for scripts far heavier than the reference fixture.
DEFAULT_RENDER_TIMEOUT_SECONDS = 1800

# 4 GiB, not 8: the Docker VM this runs on has only 7 GiB total, so a larger cap
# could not actually be honoured. Measured peak was 764 MB.
DEFAULT_RENDER_MEMORY_LIMIT_GB = 4

# Manim 0.18 has no configurable cache location — it keeps cached animation
# segments in `partial_movie_files/` *inside* media_dir. Rendering used to hand
# it a fresh tempdir and delete it afterwards, so the cache could never survive
# a run and --disable_caching was the honest setting. Keeping media_dir per
# project on the shared volume is what actually makes caching possible: Manim
# keys each segment by its own content hash, so editing one narration line
# re-renders only what changed.
CACHE_ROOT = "/shared/.manim-media"

# A persistent per-project media_dir is a cache, so it needs a ceiling or it
# grows without bound on the shared volume. Measured at ~1.6 MB per 215s 720p30
# project; 1080p60 and longer videos cost proportionally more, so 5 GB holds a
# healthy working set while staying well clear of filling the volume.
DEFAULT_CACHE_BUDGET_BYTES = 5 * 1024 * 1024 * 1024

# How often to report that a long render is still alive (CR-003 FR11.4).
HEARTBEAT_INTERVAL_SECONDS = 15

# Manim logs "Animation 12: ..." as it works through a scene. There is no
# reliable total to divide by — animations inside loops mean the count of
# `self.play` calls in the source under-counts them — so this is reported as a
# position, never as a fabricated percentage.
ANIMATION_LINE_RE = re.compile(r"Animation (\d+)\s*:")

AUTO_WAIT_RE = re.compile(r"self\.wait\(\s*AUTO\s*\)")

MARKS_FILENAME = "cf_marks.jsonl"

# Prepended to the patched script. Names are `_cf_`-prefixed so they cannot
# collide with anything the Creator wrote.
MARK_PREAMBLE = """
import json as _cf_json, os as _cf_os
_CF_MARKS_PATH = _cf_os.environ["CF_MARKS_PATH"]


def _cf_mark(_cf_scene, _cf_index):
    with open(_CF_MARKS_PATH, "a") as _cf_f:
        _cf_f.write(_cf_json.dumps({"index": _cf_index, "t": _cf_scene.renderer.time}) + "\\n")
"""


class ManimScriptRenderer(ManimScriptRendererPort):
    def __init__(
        self,
        timeout_seconds: int = DEFAULT_RENDER_TIMEOUT_SECONDS,
        memory_limit_gb: int = DEFAULT_RENDER_MEMORY_LIMIT_GB,
        cache_root: str | None = CACHE_ROOT,
        cache_budget_bytes: int = DEFAULT_CACHE_BUDGET_BYTES,
        on_heartbeat: Callable[[float, int | None], None] | None = None,
    ) -> None:
        self._timeout_seconds = timeout_seconds
        self._memory_limit_bytes = memory_limit_gb * 1024 * 1024 * 1024
        self._cache_root = cache_root
        self._cache_budget_bytes = cache_budget_bytes
        # Called from the render thread every HEARTBEAT_INTERVAL_SECONDS with
        # (elapsed_seconds, latest_animation_index). Optional so tests and the
        # use case can ignore progress entirely.
        self._on_heartbeat = on_heartbeat

    def set_heartbeat(self, callback: Callable[[float, int | None], None] | None) -> None:
        """Swapped per command, since the callback carries the project_id."""
        self._on_heartbeat = callback

    def render(self, request: ScriptRenderRequest, output_path: str) -> ScriptRenderResult:
        durations = [
            seg.duration_seconds
            for seg in sorted(request.narration_segments, key=lambda s: s.scene_index)
        ]
        patched_script = self._patch_auto_waits(request.script_content, durations)

        media_dir, ephemeral = self._media_dir_for(request.project_id)
        script_path = os.path.join(media_dir, "script.py")
        marks_path = os.path.join(media_dir, MARKS_FILENAME)
        try:
            # The marks file is appended to, so a reused media_dir must not
            # carry the previous render's marks into this one.
            if os.path.exists(marks_path):
                os.remove(marks_path)
            with open(script_path, "w") as f:
                f.write(patched_script)

            self._run_manim(script_path, request.scene_class_name, media_dir, marks_path)

            rendered_path = self._find_rendered_file(media_dir)
            wait_offsets = self._read_wait_offsets(marks_path, expected=len(durations))
            video_duration = _probe_duration(rendered_path)
            shutil.move(rendered_path, output_path)
        finally:
            if ephemeral:
                shutil.rmtree(media_dir, ignore_errors=True)

        return ScriptRenderResult(
            video_path=output_path,
            wait_offsets=wait_offsets,
            video_duration_seconds=video_duration,
        )

    @staticmethod
    def _patch_auto_waits(script_content: str, durations: list[float]) -> str:
        occurrences = len(AUTO_WAIT_RE.findall(script_content))
        if occurrences != len(durations):
            raise AnimationEngineError(
                f"script has {occurrences} self.wait(AUTO) call(s) but "
                f"{len(durations)} narration segment(s) were provided — "
                "each '# NARRATION: \"...\"' marker needs exactly one "
                "self.wait(AUTO) call right after it"
            )
        it = iter(durations)
        indices = iter(range(len(durations)))

        def substitute(_match: re.Match) -> str:
            # A tuple expression, so this stays a single expression statement and
            # slots into whatever indentation the Creator used. Left-to-right
            # evaluation means the mark is taken at the START of the wait.
            return f"(_cf_mark(self, {next(indices)}), self.wait({next(it)}))"

        patched = AUTO_WAIT_RE.sub(substitute, script_content)
        return _insert_preamble(patched)

    def _run_manim(
        self, script_path: str, scene_class_name: str, media_dir: str, marks_path: str
    ) -> None:
        cmd = ["manim", "-qm", "--media_dir", media_dir]
        if self._cache_root is None:
            cmd.append("--disable_caching")
        cmd += [script_path, scene_class_name]
        safe_env = {
            "PATH": os.environ.get("PATH", "/usr/bin:/bin"),
            "HOME": media_dir,
            # The only channel by which the patched script reports timing back.
            # It stays inside media_dir, which is torn down after every render.
            "CF_MARKS_PATH": marks_path,
        }

        returncode, stderr = self._run_with_heartbeat(cmd, media_dir, safe_env)

        if returncode != 0:
            logger.warning("Manim render failed: %s", stderr)
            raise AnimationEngineError(f"Manim render failed:\n{stderr}")

    def _run_with_heartbeat(
        self, cmd: list[str], media_dir: str, safe_env: dict[str, str]
    ) -> tuple[int, str]:
        """Runs Manim, streaming stderr so a long render can report that it is
        still alive (CR-003 FR11.4).

        Popen rather than subprocess.run because the latter only hands back
        output once the process has exited — which for a multi-minute render
        means the Creator watches a motionless screen with no way to tell a
        working render from a hung one.
        """
        process = subprocess.Popen(  # noqa: S603 — cmd is built here, not user input
            cmd,
            cwd=media_dir,
            env=safe_env,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.PIPE,
            text=True,
            preexec_fn=self._limit_child_resources,
        )

        started = time.monotonic()
        collected: list[str] = []
        latest_animation: list[int | None] = [None]
        done = threading.Event()

        def drain_stderr() -> None:
            # Manim's output is bounded (log lines, not a data stream), so
            # collecting it is safe and keeps the failure message intact.
            for line in process.stderr:
                collected.append(line)
                match = ANIMATION_LINE_RE.search(line)
                if match:
                    latest_animation[0] = int(match.group(1))

        def beat() -> None:
            while not done.wait(HEARTBEAT_INTERVAL_SECONDS):
                try:
                    self._on_heartbeat(time.monotonic() - started, latest_animation[0])
                except Exception:  # noqa: BLE001 — progress is UX-only
                    logger.exception("Heartbeat callback failed; continuing the render")

        reader = threading.Thread(target=drain_stderr, daemon=True)
        reader.start()
        heartbeat = None
        if self._on_heartbeat is not None:
            heartbeat = threading.Thread(target=beat, daemon=True)
            heartbeat.start()

        try:
            returncode = process.wait(timeout=self._timeout_seconds)
        except subprocess.TimeoutExpired as exc:
            process.kill()
            process.wait()
            raise AnimationEngineError(
                f"Manim render timed out after {self._timeout_seconds}s"
            ) from exc
        finally:
            done.set()
            reader.join(timeout=5)
            if heartbeat is not None:
                heartbeat.join(timeout=1)

        return returncode, "".join(collected)

    def _limit_child_resources(self) -> None:
        """Address space only — see the module docstring for why RLIMIT_CPU is
        deliberately absent."""
        resource.setrlimit(
            resource.RLIMIT_AS, (self._memory_limit_bytes, self._memory_limit_bytes)
        )

    @staticmethod
    def _read_wait_offsets(marks_path: str, expected: int) -> list[float]:
        """Reads the offsets the patched script recorded, in scene order.

        A mismatch means the script's control flow diverged from a simple
        top-to-bottom pass — e.g. a `self.wait(AUTO)` inside a loop or an `if`,
        which would fire a different number of times than there are narration
        segments. Rendering must fail loudly here: silently returning a partial
        list would put every later narration on the wrong offset, which is the
        exact class of bug CR-002 exists to remove.
        """
        if not os.path.isfile(marks_path):
            raise AnimationEngineError(
                "the render produced no timing marks — the script may not have "
                "reached any `self.wait(AUTO)` call"
            )

        marks: dict[int, float] = {}
        with open(marks_path, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    record = json.loads(line)
                    marks[int(record["index"])] = float(record["t"])
                except (ValueError, KeyError, TypeError) as exc:
                    raise AnimationEngineError(f"unreadable timing mark {line!r}") from exc

        if sorted(marks) != list(range(expected)):
            raise AnimationEngineError(
                f"expected timing marks for narration segments 0..{expected - 1}, "
                f"got {sorted(marks)} — each `self.wait(AUTO)` must run exactly "
                "once, so it cannot sit inside a loop or a conditional"
            )
        return [marks[i] for i in range(expected)]

    @staticmethod
    def _find_rendered_file(media_dir: str) -> str:
        """Locates the finished scene video.

        `partial_movie_files/` is skipped explicitly: it holds the per-animation
        cache segments, which are also .mp4 files. A reused media_dir is full of
        them, and picking one up instead of the real output would ship a video
        containing a fraction of a second of animation. The newest match wins,
        so a stale output from an earlier render is never preferred over this
        one's.
        """
        candidates = []
        for root, dirs, files in os.walk(media_dir):
            dirs[:] = [d for d in dirs if d != "partial_movie_files"]
            for name in files:
                if name.endswith(".mp4"):
                    candidates.append(os.path.join(root, name))
        if not candidates:
            raise AnimationEngineError(f"Manim did not produce an .mp4 file under {media_dir}")
        return max(candidates, key=os.path.getmtime)

    def _media_dir_for(self, project_id: str) -> tuple[str, bool]:
        """Returns (media_dir, delete_it_afterwards).

        With caching on, each project keeps its own directory so Manim's
        partial_movie_files survive between renders. With caching off, a
        tempdir is used and torn down, exactly as before.
        """
        if self._cache_root is None:
            return tempfile.mkdtemp(prefix="manim-media-"), True
        media_dir = os.path.join(self._cache_root, project_id)
        os.makedirs(media_dir, exist_ok=True)
        self._prune_cache(keep=project_id)
        return media_dir, False

    def _prune_cache(self, keep: str) -> None:
        """Evicts least-recently-used project caches once the budget is
        exceeded. Never touches `keep` — that is the render about to run.

        Failures here are logged and swallowed: a cache that cannot be trimmed
        is a disk-space problem, not a reason to fail the Creator's render.
        """
        if self._cache_root is None:
            return
        try:
            entries = []
            total = 0
            with os.scandir(self._cache_root) as it:
                for entry in it:
                    if not entry.is_dir():
                        continue
                    size = _directory_size(entry.path)
                    total += size
                    if entry.name != keep:
                        entries.append((entry.stat().st_mtime, size, entry.path))

            entries.sort()  # oldest first
            for _mtime, size, path in entries:
                if total <= self._cache_budget_bytes:
                    break
                shutil.rmtree(path, ignore_errors=True)
                total -= size
                logger.info("Evicted Manim cache %s to stay within budget", path)
        except OSError:
            logger.exception("Could not prune the Manim cache at %s", self._cache_root)


def _insert_preamble(script: str) -> str:
    """Puts the `_cf_mark` helper after the script's manim import.

    It has to land after `from manim import *`, since a star-import later in
    the file would otherwise be free to shadow the helper.
    """
    lines = script.splitlines()
    for i, line in enumerate(lines):
        stripped = line.strip()
        if stripped.startswith("from manim import") or stripped.startswith("import manim"):
            lines.insert(i + 1, MARK_PREAMBLE)
            return "\n".join(lines)
    return MARK_PREAMBLE + "\n" + script


def _probe_duration(video_path: str) -> float:
    """Real length of the rendered file, from ffprobe.

    Taken from the file rather than computed as sum(play) + sum(wait), because
    Manim's own frame rounding makes the arithmetic drift slightly from what it
    actually wrote, and Video Assembly pads against this number.
    """
    result = subprocess.run(
        [
            "ffprobe", "-v", "error",
            "-show_entries", "format=duration",
            "-of", "csv=p=0",
            video_path,
        ],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise AnimationEngineError(f"ffprobe failed on the rendered video: {result.stderr}")
    try:
        return float(result.stdout.strip())
    except ValueError as exc:
        raise AnimationEngineError(
            f"ffprobe returned an unreadable duration: {result.stdout!r}"
        ) from exc


def _directory_size(path: str) -> int:
    total = 0
    for root, _dirs, files in os.walk(path):
        for name in files:
            try:
                total += os.path.getsize(os.path.join(root, name))
            except OSError:
                continue
    return total
