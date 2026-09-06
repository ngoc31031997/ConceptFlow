"""ManimScriptParser — implements ScriptParserPort.

Replaces the Markdown-scene grammar: `script_content` is now a full,
hand-written Manim Python script (a `from manim import *` file defining one
or more `Scene` subclasses). This parser never executes the script — it
only scans the source text for two markers, in file order:

- `class XxxScene(Scene):` (or any base class containing "Scene", e.g.
  `MovingCameraScene`) — the first one found is the scene the Rendering
  Service will execute.
- `# NARRATION: "..."` comments — each becomes one narration segment for
  TTS. The Rendering Service later substitutes each corresponding
  `self.wait(AUTO)` call (matched by ordinal position) with the real
  synthesized-audio duration, so the animation's pacing stays in lockstep
  with the voiceover without this service ever running the script.
"""

from __future__ import annotations

import re

from domain.errors import ScriptSyntaxError
from domain.models import ParsedScript, Scene
from domain.ports import ScriptParserPort

SCENE_CLASS_RE = re.compile(r"^class\s+(\w+)\s*\([^)]*Scene[^)]*\)\s*:")
NARRATION_RE = re.compile(r'^\s*#\s*NARRATION:\s*"(.*)"\s*$')


class ManimScriptParser(ScriptParserPort):
    def parse(self, raw_script: str) -> ParsedScript:
        lines = raw_script.splitlines()

        scene_class_name = self._find_scene_class(lines)
        if scene_class_name is None:
            raise ScriptSyntaxError(
                None, "no Manim Scene subclass found (expected `class XxxScene(Scene):`)"
            )

        scenes = self._find_narration_scenes(lines)
        if not scenes:
            raise ScriptSyntaxError(
                None,
                'no narration markers found (expected `# NARRATION: "..."` comments)',
            )

        return ParsedScript(scenes=scenes, scene_class_name=scene_class_name)

    @staticmethod
    def _find_scene_class(lines: list[str]) -> str | None:
        for line in lines:
            match = SCENE_CLASS_RE.match(line.strip())
            if match:
                return match.group(1)
        return None

    @staticmethod
    def _find_narration_scenes(lines: list[str]) -> list[Scene]:
        scenes: list[Scene] = []
        for line in lines:
            match = NARRATION_RE.match(line)
            if not match:
                continue
            narration_text = match.group(1).strip()
            if not narration_text:
                continue
            scenes.append(
                Scene(
                    scene_index=len(scenes),
                    narration_text=narration_text,
                    illustration_hint=None,
                    code_snippet=None,
                    code_language=None,
                )
            )
        return scenes
