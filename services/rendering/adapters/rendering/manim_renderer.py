"""ManimScriptRenderer — implements ManimScriptRendererPort.

Runs the Creator's own Manim script (not a pre-built template) as a
**subprocess**, never in-process — this is untrusted, hand-written code, so
it must never run inside the Rendering Service's own Python process (which
holds RabbitMQ/Postgres credentials in its environment). Guardrails applied
to the subprocess:

- A stripped environment (no RABBITMQ_URL/DATABASE_URL/etc. — only PATH/HOME).
- A wall-clock timeout (RENDER_TIMEOUT_SECONDS, default 300s).
- CPU-time and address-space (memory) resource limits via `resource.setrlimit`,
  applied in the child before exec via `preexec_fn`.

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

DEFAULT_RENDER_TIMEOUT_SECONDS = 300
RENDER_MEMORY_LIMIT_BYTES = 2 * 1024 * 1024 * 1024  # 2 GiB address-space cap

AUTO_WAIT_RE = re.compile(r"self\.wait\(\s*AUTO\s*\)")


class ManimScriptRenderer(ManimScriptRendererPort):
    def __init__(self, timeout_seconds: int = DEFAULT_RENDER_TIMEOUT_SECONDS) -> None:
        self._timeout_seconds = timeout_seconds

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
        resource.setrlimit(resource.RLIMIT_AS, (RENDER_MEMORY_LIMIT_BYTES, RENDER_MEMORY_LIMIT_BYTES))
        resource.setrlimit(resource.RLIMIT_CPU, (self._timeout_seconds, self._timeout_seconds))

    @staticmethod
    def _find_rendered_file(media_dir: str) -> str:
        for root, _dirs, files in os.walk(media_dir):
            for name in files:
                if name.endswith(".mp4"):
                    return os.path.join(root, name)
        raise AnimationEngineError(f"Manim did not produce an .mp4 file under {media_dir}")
