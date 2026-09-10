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

Rendering runs the script **twice** (CR-018):

1. A dry pass (`dry_run`) that executes it with `--dry_run`, producing no video
   but collecting every `self.narrate(...)` line in the order it really runs.
2. A real pass (`render`) where each narrate waits exactly as long as its
   synthesized audio, and records where in the finished video that wait begins.

Before CR-018 this was done by rewriting the source: `self.wait(AUTO)` was
textually substituted with `(_cf_mark(self, i), self.wait(D))`. That forced the
narration count to match the wait count exactly and in file order, which in turn
banned narration from loops, branches and helpers. Running the script is what
removes the need for any of it — the two passes execute the same code, so the
count and the order agree by construction.

Each mark records **where in the finished video that wait begins** (CR-002
FR10.1). This matters because the video's timeline is

    video_duration = sum(self.play(...) durations) + sum(self.wait(...))

so narration i does not start at the sum of the preceding narration durations —
it starts after all the animation that ran before it too. Video Assembly needs
those real offsets to place each audio segment; without them the narration runs
ahead of the picture by the accumulated animation time (measured at 61.6s on a
3.6-minute reference video before this was fixed).
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
from domain.models import (
    ChannelAssetRenderRequest,
    ChannelAssetRenderResult,
    DryRunResult,
    ScriptRenderRequest,
    ScriptRenderResult,
)
from domain.ports import ChannelAssetRendererPort, ManimScriptRendererPort

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

# Manim's quality flags. 720p30 was hardcoded, which is below what a monetized
# channel should publish and throws away Manim's main strength — smooth motion
# (CR-004 FR12.1). 1080p60 is the default; the Phase 0 benchmark measured it at
# 3.6x the render time of 720p30 and 2.1x the peak memory, both well inside the
# limits CR-003 raised.
QUALITY_FLAGS = {
    "720p30": "-qm",
    "1080p60": "-qh",
    "4k60": "-qk",
}
DEFAULT_RENDER_QUALITY = "1080p60"

MARKS_FILENAME = "cf_marks.jsonl"

# Directory that must be on the child's PYTHONPATH for `import conceptflow` to
# resolve (CR-017). Derived from this file's own location rather than hardcoded,
# so it stays correct whether the service runs from /app inside the image or
# from a checkout during development.
_CONCEPTFLOW_PARENT = os.path.dirname(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
)

# Framerate per quality flag, used to round narration durations onto a whole
# number of frames (CR-018 FR49.5).
#
# Manim keys each cached animation segment by a content hash. A duration carried
# to microsecond precision changes that hash for every segment after an edit,
# so the cache RENDER_CACHE_ROOT exists to provide (measured: ~5x faster on a
# re-render) is thrown away on every pass. Rounding to a frame boundary keeps
# the hashes stable without shifting anything the viewer can perceive.
QUALITY_FPS = {
    "720p30": 30,
    "1080p60": 60,
    "4k60": 60,
}

DURATIONS_FILENAME = "cf_durations.json"

#: A dry pass must not be allowed to run as long as a real render — a script
#: that hangs should surface on the cheap pass, not the expensive one
#: (CR-018 FR49.2).
DEFAULT_DRY_RUN_TIMEOUT_SECONDS = 300

#: CR-023 D3/D4 — the only two scene classes `render_channel_asset` may run.
#: A fixed map (not an arbitrary `scene_class_name` from the request) because,
#: unlike `render()`, this path is not validated by CR-020's lint/dry-run gate
#: first: the caller is the admin flow in D3, not a Creator-authored script.
CHANNEL_ASSET_SCENES = {
    "intro": "DefaultIntroSting",
    "outro": "ChannelOutro",
}

#: Written to a throwaway script.py so `manim <script> <SceneClassName>` can
#: find the class in the script module's own namespace — the same mechanism
#: `_run_manim` already relies on for Creator scripts, just importing a fixed
#: class instead of embedding Creator-authored source.
_CHANNEL_ASSET_SCRIPT_TEMPLATE = "from conceptflow.channel_idents import {scene_class_name}\n"


class ManimScriptRenderer(ManimScriptRendererPort, ChannelAssetRendererPort):
    def __init__(
        self,
        timeout_seconds: int = DEFAULT_RENDER_TIMEOUT_SECONDS,
        memory_limit_gb: int = DEFAULT_RENDER_MEMORY_LIMIT_GB,
        cache_root: str | None = CACHE_ROOT,
        cache_budget_bytes: int = DEFAULT_CACHE_BUDGET_BYTES,
        on_heartbeat: Callable[[float, int | None], None] | None = None,
        quality: str = DEFAULT_RENDER_QUALITY,
    ) -> None:
        self._timeout_seconds = timeout_seconds
        self._memory_limit_bytes = memory_limit_gb * 1024 * 1024 * 1024
        self._cache_root = cache_root
        self._cache_budget_bytes = cache_budget_bytes
        # Called from the render thread every HEARTBEAT_INTERVAL_SECONDS with
        # (elapsed_seconds, latest_animation_index). Optional so tests and the
        # use case can ignore progress entirely.
        self._on_heartbeat = on_heartbeat
        if quality not in QUALITY_FLAGS:
            raise ValueError(
                f"unknown render quality {quality!r}; expected one of {sorted(QUALITY_FLAGS)}"
            )
        self._quality = quality

    def set_heartbeat(self, callback: Callable[[float, int | None], None] | None) -> None:
        """Swapped per command, since the callback carries the project_id."""
        self._on_heartbeat = callback

    def dry_run(self, request: ScriptRenderRequest) -> DryRunResult:
        """Executes the script without producing a video (CR-018 FR49.1).

        Returns the narration lines **in the order they actually run**, plus the
        beat and chapter markers attached to them. This is what replaces parsing
        `# NARRATION:` comments out of the source text: a comment can only be
        read in file order, so narration could never live inside a loop, a
        branch, or a helper — and therefore hook/CTA could never be components
        (CR-006 §Quyết định #2).

        It doubles as the validation pass: a script that fails here fails before
        any TTS quota is spent (CR-020 FR56).
        """
        media_dir, ephemeral = self._media_dir_for(request.project_id)
        marks_path = os.path.join(media_dir, MARKS_FILENAME)
        script_path = os.path.join(media_dir, "script.py")
        try:
            if os.path.exists(marks_path):
                os.remove(marks_path)
            with open(script_path, "w") as f:
                f.write(request.script_content)

            self._run_manim(
                script_path,
                request.scene_class_name,
                media_dir,
                marks_path,
                self._resolve_quality(request.render_quality),
                dry=True,
            )

            records = _read_marks(marks_path)
        finally:
            if ephemeral:
                shutil.rmtree(media_dir, ignore_errors=True)

        narration_records = [r for r in records if r.get("kind") == "narration"]
        narrations = [r["text"] for r in narration_records]
        visuals = [r.get("visual", "") for r in narration_records]
        if not narrations:
            raise AnimationEngineError(
                "the script produced no narration — it needs at least one "
                "`self.narrate(\"...\")` call"
            )
        return DryRunResult(
            narrations=narrations,
            visuals=visuals,
            beats=[(int(r["index"]), r["id"]) for r in records if r.get("kind") == "beat"],
            chapters=[
                (int(r["index"]), r["title"]) for r in records if r.get("kind") == "chapter"
            ],
        )

    def render(self, request: ScriptRenderRequest, output_path: str) -> ScriptRenderResult:
        quality = self._resolve_quality(request.render_quality)
        durations = _round_to_frames(
            [
                seg.duration_seconds
                for seg in sorted(request.narration_segments, key=lambda s: s.scene_index)
            ],
            QUALITY_FPS[quality],
        )

        media_dir, ephemeral = self._media_dir_for(request.project_id)
        script_path = os.path.join(media_dir, "script.py")
        marks_path = os.path.join(media_dir, MARKS_FILENAME)
        durations_path = os.path.join(media_dir, DURATIONS_FILENAME)
        try:
            # The marks file is appended to, so a reused media_dir must not
            # carry the previous render's marks into this one.
            if os.path.exists(marks_path):
                os.remove(marks_path)
            # The script is written out exactly as the Creator wrote it. Before
            # CR-018 it was rewritten here (`self.wait(AUTO)` substituted with a
            # tuple expression), which meant what ran was never quite what they
            # authored — and what was stored was not valid Python at all.
            with open(script_path, "w") as f:
                f.write(request.script_content)
            with open(durations_path, "w") as f:
                json.dump(durations, f)

            self._run_manim(
                script_path,
                request.scene_class_name,
                media_dir,
                marks_path,
                quality,
                durations_path=durations_path,
            )

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

    def render_channel_asset(
        self, request: ChannelAssetRenderRequest, output_path: str
    ) -> ChannelAssetRenderResult:
        """Runs one of the two fixed `conceptflow.channel_idents` scenes
        (CR-023 D3/D4).

        Deliberately simpler than `render()`: no marks file, no durations
        file, no wait-offset reconciliation — neither scene calls
        `self.narrate(...)`, so there is nothing for those to reconcile.
        Reuses `_run_manim`/`_find_rendered_file`/`_probe_duration` exactly as
        `render()` does, just without the narration-timing machinery that
        does not apply here.
        """
        scene_class_name = CHANNEL_ASSET_SCENES.get(request.kind)
        if scene_class_name is None:
            raise ValueError(
                f"unknown channel asset kind {request.kind!r}; "
                f"expected one of {sorted(CHANNEL_ASSET_SCENES)}"
            )
        quality = self._resolve_quality(request.render_quality)

        # Not cached per-project like Creator scripts (there is no project_id
        # here) — a fresh tempdir per call, always torn down.
        media_dir = tempfile.mkdtemp(prefix="manim-media-channel-asset-")
        script_path = os.path.join(media_dir, "script.py")
        marks_path = os.path.join(media_dir, MARKS_FILENAME)
        try:
            with open(script_path, "w") as f:
                f.write(_CHANNEL_ASSET_SCRIPT_TEMPLATE.format(scene_class_name=scene_class_name))

            self._run_manim(script_path, scene_class_name, media_dir, marks_path, quality)

            rendered_path = self._find_rendered_file(media_dir)
            video_duration = _probe_duration(rendered_path)
            shutil.move(rendered_path, output_path)
        finally:
            shutil.rmtree(media_dir, ignore_errors=True)

        return ChannelAssetRenderResult(
            video_path=output_path,
            video_duration_seconds=video_duration,
            render_quality=quality,
        )

    def resolve_render_quality(self, requested: str | None) -> str:
        return self._resolve_quality(requested)

    def _resolve_quality(self, requested: str | None) -> str:
        """Per-project quality wins; an unknown or absent one falls back to the
        service default rather than failing, since a render at the wrong
        resolution still gives the Creator something to look at."""
        if requested in QUALITY_FLAGS:
            return requested
        if requested:
            logger.warning(
                "unknown render_quality %r, falling back to %s", requested, self._quality
            )
        return self._quality

    def _run_manim(
        self,
        script_path: str,
        scene_class_name: str,
        media_dir: str,
        marks_path: str,
        quality: str,
        durations_path: str | None = None,
        dry: bool = False,
    ) -> None:
        """Runs one pass. `dry=True` is the validation pass (CR-018 FR49).

        The dry pass renders at the lowest quality and writes no video file,
        but it *does* execute every animation — that is the point. Skipping the
        animations would make it faster and would also stop it from catching
        the API misuse it exists to catch (CR-020 FR56.2).
        """
        if dry:
            cmd = ["manim", "-ql", "--dry_run", "--disable_caching",
                   "--media_dir", media_dir]
        else:
            cmd = ["manim", QUALITY_FLAGS[quality], "--media_dir", media_dir]
            if self._cache_root is None:
                cmd.append("--disable_caching")
        cmd += [script_path, scene_class_name]

        safe_env = {
            "PATH": os.environ.get("PATH", "/usr/bin:/bin"),
            "HOME": media_dir,
            # The only channel by which the script reports back — narration on
            # the dry pass, timing marks on the real one. It stays inside
            # media_dir, which is torn down after every ephemeral render.
            "CF_MARKS_PATH": marks_path,
            # The design system the script imports (CR-017). The subprocess runs
            # with a stripped environment, so without this `from conceptflow
            # import *` cannot resolve and every script fails on its first line.
            # This is the only variable added — no new channel *out* of the
            # subprocess is opened.
            "PYTHONPATH": _CONCEPTFLOW_PARENT,
            "CF_MODE": "dry" if dry else "render",
        }
        if durations_path is not None:
            safe_env["CF_DURATIONS_PATH"] = durations_path

        # The dry pass gets the same isolation as the real one (FR49.3). It runs
        # unvetted code earlier in the pipeline, not safer code.
        timeout = DEFAULT_DRY_RUN_TIMEOUT_SECONDS if dry else self._timeout_seconds
        returncode, stderr = self._run_with_heartbeat(
            cmd, media_dir, safe_env, timeout=timeout
        )

        if returncode != 0:
            label = "Manim dry run" if dry else "Manim render"
            logger.warning("%s failed: %s", label, stderr)
            raise AnimationEngineError(f"{label} failed:\n{stderr}")

    def _run_with_heartbeat(
        self,
        cmd: list[str],
        media_dir: str,
        safe_env: dict[str, str],
        timeout: int | None = None,
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
            returncode = process.wait(timeout=timeout or self._timeout_seconds)
        except subprocess.TimeoutExpired as exc:
            process.kill()
            process.wait()
            raise AnimationEngineError(
                f"Manim render timed out after {timeout or self._timeout_seconds}s"
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
        """Reads the offsets the render pass recorded, in narration order.

        A mismatch here means the dry pass and the real pass disagreed about how
        many times `narrate()` runs — which can only happen if the script is
        non-deterministic (random, wall-clock, external state). Rendering must
        fail loudly: silently returning a partial list would put every later
        narration on the wrong offset, the exact class of bug CR-002 exists to
        remove.
        """
        records = _read_marks(marks_path)
        marks = {int(r["index"]): float(r["t"]) for r in records if r.get("kind") == "mark"}

        if not marks:
            raise AnimationEngineError(
                "the render produced no timing marks — the script may never have "
                "reached a `self.narrate(...)` call"
            )
        if sorted(marks) != list(range(expected)):
            raise AnimationEngineError(
                f"expected timing marks for narration 0..{expected - 1}, got "
                f"{sorted(marks)} — the dry pass and the render pass disagree "
                "about how many narration lines this script produces, which "
                "means it is not deterministic"
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


def _read_marks(marks_path: str) -> list[dict]:
    """Parses the JSONL the script wrote back through CF_MARKS_PATH.

    A malformed line is fatal rather than skipped: this file is the only channel
    out of the subprocess, so a line we cannot read is a line of timing or
    narration we would otherwise silently drop.
    """
    if not os.path.isfile(marks_path):
        return []
    records: list[dict] = []
    with open(marks_path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                records.append(json.loads(line))
            except ValueError as exc:
                raise AnimationEngineError(f"unreadable record {line!r}") from exc
    return records


def _round_to_frames(durations: list[float], fps: int) -> list[float]:
    """Snaps each duration onto a whole number of frames (CR-018 FR49.5).

    Manim hashes each cached animation segment by content, so a duration carried
    to microsecond precision reshuffles the hash of everything after an edited
    line and throws away the cache that RENDER_CACHE_ROOT exists to provide
    (measured: ~5x faster on a re-render). One frame at 60fps is 17ms — below
    anything a viewer can perceive against a spoken line.
    """
    return [round(round(d * fps) / fps, 6) for d in durations]
