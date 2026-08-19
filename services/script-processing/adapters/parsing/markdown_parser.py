"""MarkdownScriptParser — implements ScriptParserPort (ADR-0011).

Grammar (business-logic-model.md / business-rules.md):
- Each scene starts with a `## Scene N` heading; N must be sequential
  starting at 1 (Rule 2).
- A `> ` blockquote line right after the heading becomes illustration_hint
  (optional, first one wins — Rule 4).
- Plain text lines become narration_text (mandatory, non-empty — Rule 3).
- At most one fenced code block per scene becomes code_snippet; the
  fence's language annotation (```` ```python ````) becomes code_language
  (Rule 5, revised for Story B3's syntax-highlight requirement).
- Content before the first heading is ignored (Rule 6).
- The parser fails fast on the first violation (Rule 7).
"""

from __future__ import annotations

import re

from domain.errors import ScriptSyntaxError
from domain.models import ParsedScript, Scene
from domain.ports import ScriptParserPort

SCENE_HEADING_RE = re.compile(r"^##\s*Scene\s+(\d+)\s*$")
CODE_FENCE_RE = re.compile(r"^```(\w*)\s*$")
BLOCKQUOTE_RE = re.compile(r"^>\s?(.*)$")


class MarkdownScriptParser(ScriptParserPort):
    def parse(self, raw_script: str) -> ParsedScript:
        lines = raw_script.splitlines()
        headings = self._find_headings(lines)
        if not headings:
            raise ScriptSyntaxError(None, "no scenes found")

        scenes: list[Scene] = []
        for position, (line_index, number) in enumerate(headings):
            expected_number = position + 1
            if number != expected_number:
                raise ScriptSyntaxError(
                    line_index + 1,
                    f"scene numbering must be sequential "
                    f"(expected Scene {expected_number}, got Scene {number})",
                )

            body_start = line_index + 1
            body_end = headings[position + 1][0] if position + 1 < len(headings) else len(lines)
            scene = self._parse_scene_body(
                scene_index=position,
                heading_line=line_index + 1,
                body_lines=lines[body_start:body_end],
                body_start_line=body_start + 1,
            )
            scenes.append(scene)

        return ParsedScript(scenes=scenes)

    @staticmethod
    def _find_headings(lines: list[str]) -> list[tuple[int, int]]:
        headings = []
        for i, line in enumerate(lines):
            match = SCENE_HEADING_RE.match(line.strip())
            if match:
                headings.append((i, int(match.group(1))))
        return headings

    @staticmethod
    def _parse_scene_body(
        *, scene_index: int, heading_line: int, body_lines: list[str], body_start_line: int
    ) -> Scene:
        narration_lines: list[str] = []
        illustration_hint: str | None = None
        code_snippet: str | None = None
        code_language: str | None = None
        in_fence = False
        fence_lines: list[str] = []
        fence_start_line: int | None = None

        for offset, line in enumerate(body_lines):
            line_number = body_start_line + offset

            fence_match = CODE_FENCE_RE.match(line.strip())
            if fence_match:
                if in_fence:
                    in_fence = False
                    code_snippet = "\n".join(fence_lines)
                    fence_lines = []
                else:
                    if code_snippet is not None:
                        raise ScriptSyntaxError(
                            line_number, "a scene may contain at most one code block"
                        )
                    in_fence = True
                    fence_start_line = line_number
                    code_language = fence_match.group(1) or None
                continue

            if in_fence:
                fence_lines.append(line)
                continue

            blockquote_match = BLOCKQUOTE_RE.match(line)
            if blockquote_match:
                if illustration_hint is None:
                    illustration_hint = blockquote_match.group(1).strip()
                continue

            if line.strip():
                narration_lines.append(line.strip())

        if in_fence:
            raise ScriptSyntaxError(fence_start_line, "unterminated code block")

        narration_text = "\n".join(narration_lines).strip()
        if not narration_text:
            raise ScriptSyntaxError(heading_line, "narration_text must not be empty")

        return Scene(
            scene_index=scene_index,
            narration_text=narration_text,
            illustration_hint=illustration_hint,
            code_snippet=code_snippet,
            code_language=code_language,
        )
