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
"""

from __future__ import annotations

import logging
import os
import re
import resource
import shutil
import subprocess
import tempfile

from domain.errors import AnimationEngineError
from domain.models import ScriptRenderRequest
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

AUTO_WAIT_RE = re.compile(r"self\.wait\(\s*AUTO\s*\)")


class ManimScriptRenderer(ManimScriptRendererPort):
    def __init__(
        self,
        timeout_seconds: int = DEFAULT_RENDER_TIMEOUT_SECONDS,
        memory_limit_gb: int = DEFAULT_RENDER_MEMORY_LIMIT_GB,
    ) -> None:
        self._timeout_seconds = timeout_seconds
        self._memory_limit_bytes = memory_limit_gb * 1024 * 1024 * 1024

    def render(self, request: ScriptRenderRequest, output_path: str) -> None:
        durations = [seg.duration_seconds for seg in sorted(request.narration_segments, key=lambda s: s.scene_index)]
        patched_script = self._patch_auto_waits(request.script_content, durations)

        media_dir = tempfile.mkdtemp(prefix="manim-media-")
        script_path = os.path.join(media_dir, "script.py")
        try:
            with open(script_path, "w") as f:
                f.write(patched_script)

            self._run_manim(script_path, request.scene_class_name, media_dir)

            rendered_path = self._find_rendered_file(media_dir)
            shutil.move(rendered_path, output_path)
        finally:
            shutil.rmtree(media_dir, ignore_errors=True)

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
        return AUTO_WAIT_RE.sub(lambda _match: f"self.wait({next(it)})", script_content)

    def _run_manim(self, script_path: str, scene_class_name: str, media_dir: str) -> None:
        cmd = [
            "manim",
            "-qm",
            "--disable_caching",
            "--media_dir",
            media_dir,
            script_path,
            scene_class_name,
        ]
        safe_env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "HOME": media_dir}

        try:
            result = subprocess.run(
                cmd,
                cwd=media_dir,
                env=safe_env,
                timeout=self._timeout_seconds,
                capture_output=True,
                text=True,
                preexec_fn=self._limit_child_resources,
            )
        except subprocess.TimeoutExpired as exc:
            raise AnimationEngineError(f"Manim render timed out after {self._timeout_seconds}s") from exc

        if result.returncode != 0:
            logger.warning("Manim render failed: %s", result.stderr)
            raise AnimationEngineError(f"Manim render failed:\n{result.stderr}")

    def _limit_child_resources(self) -> None:
        """Address space only — see the module docstring for why RLIMIT_CPU is
        deliberately absent."""
        resource.setrlimit(
            resource.RLIMIT_AS, (self._memory_limit_bytes, self._memory_limit_bytes)
        )

    @staticmethod
    def _find_rendered_file(media_dir: str) -> str:
        for root, _dirs, files in os.walk(media_dir):
            for name in files:
                if name.endswith(".mp4"):
                    return os.path.join(root, name)
        raise AnimationEngineError(f"Manim did not produce an .mp4 file under {media_dir}")
