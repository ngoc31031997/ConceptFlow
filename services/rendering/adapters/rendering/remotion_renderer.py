"""RemotionScriptRenderer — implements ManimScriptRendererPort for the
Remotion engine (feature/remotion-engine).

Unlike Manim, whose dry pass has to actually RUN the script (narration calls
can live inside loops/conditions, so only real execution order is trustworthy
— see manim_renderer.py), a Remotion composition is React-declarative: the
Creator's script must export a plain module-level `narrations: string[]`
array alongside their `registerRoot()`/`<Composition>` (script-processing's
`ManimScriptParser` already requires the Composition/registerRoot pair to
even accept the script as Remotion — see
script-processing/adapters/parsing/manim_script_parser.py). That array can be
read straight off the source text with a regex, no Node process needed at
all for dry_run() — faster, and it means a broken dry pass can never be a
Remotion/Chromium problem, only a script problem.

Two-pass contract (mirrors manim_renderer.py's outward behaviour exactly —
see that module's docstring for the full CR-018 rationale):
- dry_run(): regex-extract `narrations` from script_content. No video, no
  Node/Chromium spent.
- render(): convert each narration_segment's real TTS duration_seconds into
  frames (fixed FPS), lay them out back-to-back as a `segments` prop, run the
  Node driver (bundle -> selectComposition -> renderMedia, per
  https://www.remotion.dev/docs/ssr-node) to produce the real video, and
  report wait_offsets — each segment's start time in seconds — the same
  contract Video Assembly already reads from the Manim path.
"""

from __future__ import annotations

import json
import logging
import os
import re
import subprocess
import tempfile
from collections.abc import Callable

from domain.errors import AnimationEngineError
from domain.models import DryRunResult, ScriptRenderRequest, ScriptRenderResult
from domain.ports import ManimScriptRendererPort

logger = logging.getLogger(__name__)

DEFAULT_RENDER_TIMEOUT_SECONDS = 1800
FPS = 30

CACHE_ROOT = "/shared/.remotion-media"
ENTRY_FILENAME = "CreatorEntry.tsx"
PROPS_FILENAME = "cf_props.json"

_SERVICE_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
PROJECT_TEMPLATE_DIR = os.path.join(_SERVICE_ROOT, "remotion_project")
RENDER_SCRIPT = os.path.join(PROJECT_TEMPLATE_DIR, "render.mjs")

# Anchored on `export const narrations` (mirrors web-gui's
# scriptValidation.ts, which rejects a script missing the `export`) so a
# `// narrations = [...]` left in a comment, or an unrelated `const
# myNarrations = [...]`, can never be picked up in place of the real array.
_NARRATIONS_HEADER_RE = re.compile(
    r"export\s+const\s+narrations\s*(?::\s*string\s*\[\s*\]\s*)?=\s*\["
)
# Double/single/back-quoted string literals — matches web-gui's
# REMOTION_STRING_LITERAL_RE exactly, including the backtick that the old
# regex here was missing (a `narrations` array of template literals used to
# silently count as empty).
_STRING_LITERAL_RE = re.compile(r"""(['"`])((?:(?!\1)[^\\]|\\.)*)\1""")


def _strip_comments(text: str) -> str:
    """Removes `//` and `/* */` comments, respecting string literals, so a
    comment can never be mistaken for the real narrations array."""
    out: list[str] = []
    i, n = 0, len(text)
    in_string: str | None = None
    while i < n:
        ch = text[i]
        if in_string:
            out.append(ch)
            if ch == "\\" and i + 1 < n:
                out.append(text[i + 1])
                i += 2
                continue
            if ch == in_string:
                in_string = None
            i += 1
            continue
        if ch in ("'", '"', "`"):
            in_string = ch
            out.append(ch)
            i += 1
            continue
        if ch == "/" and i + 1 < n and text[i + 1] == "/":
            while i < n and text[i] != "\n":
                i += 1
            continue
        if ch == "/" and i + 1 < n and text[i + 1] == "*":
            i += 2
            while i + 1 < n and not (text[i] == "*" and text[i + 1] == "/"):
                i += 1
            i += 2
            continue
        out.append(ch)
        i += 1
    return "".join(out)


def _narrations_body(script_content: str) -> str | None:
    """Finds the `export const narrations = [...]` array and returns its
    body — the raw text between the brackets — or None if absent.

    Scans bracket/string-aware instead of the old `[^\\]]*` regex, which cut
    the array short at the first `]` even when it was inside a narration
    string (e.g. `"Xem mục [1] nhé."`).
    """
    stripped = _strip_comments(script_content)
    header = _NARRATIONS_HEADER_RE.search(stripped)
    if not header:
        return None
    i, n = header.end(), len(stripped)
    depth = 1
    in_string: str | None = None
    start = i
    while i < n and depth > 0:
        ch = stripped[i]
        if in_string:
            if ch == "\\" and i + 1 < n:
                i += 2
                continue
            if ch == in_string:
                in_string = None
            i += 1
            continue
        if ch in ("'", '"', "`"):
            in_string = ch
        elif ch == "[":
            depth += 1
        elif ch == "]":
            depth -= 1
        i += 1
    return stripped[start : i - 1]


class RemotionScriptRenderer(ManimScriptRendererPort):
    def __init__(
        self,
        timeout_seconds: int = DEFAULT_RENDER_TIMEOUT_SECONDS,
        cache_root: str | None = CACHE_ROOT,
        project_template_dir: str = PROJECT_TEMPLATE_DIR,
    ) -> None:
        self._timeout_seconds = timeout_seconds
        self._cache_root = cache_root
        self._project_template_dir = project_template_dir

    def dry_run(self, request: ScriptRenderRequest) -> DryRunResult:
        narrations = _extract_narrations(request.script_content)
        if not narrations:
            raise AnimationEngineError(
                "the script produced no narration — it needs a module-level "
                '`export const narrations: string[] = [...]` with at least one line'
            )
        return DryRunResult(narrations=narrations)

    def render(self, request: ScriptRenderRequest, output_path: str) -> ScriptRenderResult:
        segments = _segments_from(request.narration_segments)
        if not segments:
            raise AnimationEngineError(
                "render() called with no narration_segments — nothing to time the "
                "composition against"
            )

        media_dir, ephemeral = self._media_dir_for(request.project_id)
        entry_path = self._entry_path()
        props_path = os.path.join(media_dir, PROPS_FILENAME)
        rendered_path = os.path.join(media_dir, "output.mp4")
        try:
            with open(entry_path, "w", encoding="utf-8") as f:
                f.write(request.script_content)
            with open(props_path, "w", encoding="utf-8") as f:
                props: dict = {"segments": segments}
                if request.video_font:
                    props["videoFont"] = request.video_font
                json.dump(props, f)

            self._run_node(
                [
                    "--entry", entry_path,
                    "--id", request.scene_class_name,
                    "--props", props_path,
                    "--out", rendered_path,
                ],
                timeout=self._timeout_seconds,
            )

            video_duration = _probe_duration(rendered_path)
            os.replace(rendered_path, output_path)
        finally:
            if ephemeral:
                import shutil

                shutil.rmtree(media_dir, ignore_errors=True)

        wait_offsets = [seg["startFrame"] / FPS for seg in segments]
        return ScriptRenderResult(
            video_path=output_path,
            wait_offsets=wait_offsets,
            video_duration_seconds=video_duration,
        )

    def _run_node(self, script_args: list[str], timeout: int) -> None:
        cmd = ["node", RENDER_SCRIPT] + script_args

        try:
            result = subprocess.run(  # noqa: S603 — cmd is built here, not user input
                cmd,
                cwd=self._project_template_dir,
                capture_output=True,
                text=True,
                timeout=timeout,
            )
        except subprocess.TimeoutExpired as exc:
            raise AnimationEngineError(f"Remotion render timed out after {timeout}s") from exc

        if result.returncode != 0:
            logger.warning("Remotion render failed: %s", result.stderr)
            raise AnimationEngineError(f"Remotion render failed:\n{result.stderr}")

    def _entry_path(self) -> str:
        """Where the Creator's script gets written before bundle() reads it.

        Deliberately a **fixed** path inside the project template's own
        `src/` directory, not the per-project media_dir: the entry file's
        `import {TitleText} from './conceptflow-mini/primitives'` is a
        relative ES module specifier resolved against the *importing file's
        own location* — writing it anywhere outside `src/` breaks that
        import. Same tradeoff, same reasoning, as the discarded Motion
        Canvas attempt's `_scene_path()`: one render at a time shares this
        slot, which already holds in practice since
        RenderScriptCommandHandler/ValidateScriptCommandHandler process one
        message at a time per consumer.
        """
        return os.path.join(self._project_template_dir, "src", ENTRY_FILENAME)

    def _media_dir_for(self, project_id: str) -> tuple[str, bool]:
        if self._cache_root is None:
            return tempfile.mkdtemp(prefix="remotion-media-"), True
        media_dir = os.path.join(self._cache_root, project_id)
        os.makedirs(media_dir, exist_ok=True)
        return media_dir, False


def _extract_narrations(script_content: str) -> list[str]:
    body = _narrations_body(script_content)
    if body is None:
        return []
    return [m.group(2) for m in _STRING_LITERAL_RE.finditer(body)]


def _segments_from(narration_segments) -> list[dict]:
    ordered = sorted(narration_segments, key=lambda s: s.scene_index)
    segments = []
    frame_cursor = 0
    for seg in ordered:
        duration_frames = max(1, round(seg.duration_seconds * FPS))
        segments.append({
            "startFrame": frame_cursor,
            "durationInFrames": duration_frames,
        })
        frame_cursor += duration_frames
    return segments


def _probe_duration(video_path: str) -> float:
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
