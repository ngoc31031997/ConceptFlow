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
- `# CHAPTER: "..."` comments (CR-006 FR15.1) — each marks where a YouTube
  chapter begins. It attaches to the NEXT narration marker, so the chapter
  timestamp is the real offset Rendering measured for that line rather than
  an estimate. Chapters are optional; a script with none simply gets no
  chapter list.

The Creator writes chapters explicitly rather than having an LLM infer them,
because they already decide the structure when writing the script, and a model
guessing at section boundaries produces chapters that drift from what the video
actually does.
"""

from __future__ import annotations

import re

from domain.errors import ScriptSyntaxError
from domain.models import Chapter, ParsedScript, Scene
from domain.ports import ScriptParserPort

SCENE_CLASS_RE = re.compile(r"^class\s+(\w+)\s*\([^)]*Scene[^)]*\)\s*:")
NARRATION_RE = re.compile(r'^\s*#\s*NARRATION:\s*"(.*)"\s*$')
CHAPTER_RE = re.compile(r'^\s*#\s*CHAPTER:\s*"(.*)"\s*$')


class ManimScriptParser(ScriptParserPort):
    def parse(self, raw_script: str) -> ParsedScript:
        lines = raw_script.splitlines()

        scene_class_name = self._find_scene_class(lines)
        if scene_class_name is None:
            raise ScriptSyntaxError(
                None, "no Manim Scene subclass found (expected `class XxxScene(Scene):`)"
            )

        scenes, chapters = self._find_narration_and_chapters(lines)
        if not scenes:
            raise ScriptSyntaxError(
                None,
                'no narration markers found (expected `# NARRATION: "..."` comments)',
            )

        return ParsedScript(
            scenes=scenes, scene_class_name=scene_class_name, chapters=chapters
        )

    @staticmethod
    def _find_scene_class(lines: list[str]) -> str | None:
        for line in lines:
            match = SCENE_CLASS_RE.match(line.strip())
            if match:
                return match.group(1)
        return None

    @staticmethod
    def _find_narration_and_chapters(
        lines: list[str],
    ) -> tuple[list[Scene], list[Chapter]]:
        scenes: list[Scene] = []
        chapters: list[Chapter] = []
        pending_chapter: str | None = None

        for line in lines:
            chapter_match = CHAPTER_RE.match(line)
            if chapter_match:
                title = chapter_match.group(1).strip()
                if title:
                    # Held until the next narration marker: a chapter's
                    # timestamp is the offset of the line that opens it.
                    pending_chapter = title
                continue

            match = NARRATION_RE.match(line)
            if not match:
                continue
            narration_text = match.group(1).strip()
            if not narration_text:
                continue

            if pending_chapter is not None:
                chapters.append(Chapter(scene_index=len(scenes), title=pending_chapter))
                pending_chapter = None

            scenes.append(
                Scene(
                    scene_index=len(scenes),
                    narration_text=narration_text,
                    illustration_hint=None,
                    code_snippet=None,
                    code_language=None,
                )
            )
        # A trailing chapter with no narration after it is dropped: there is no
        # timestamp to give it.
        return scenes, chapters
