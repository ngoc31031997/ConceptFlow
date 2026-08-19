"""Unit tests for MarkdownScriptParser (business-rules.md, ADR-0011)."""

from __future__ import annotations

import pytest

from adapters.parsing.markdown_parser import MarkdownScriptParser
from domain.errors import ScriptSyntaxError


@pytest.fixture
def parser() -> MarkdownScriptParser:
    return MarkdownScriptParser()


def test_parses_single_scene_with_all_fields(parser):
    script = (
        "## Scene 1\n"
        "> minh hoa vong lap for\n"
        "Day la loi thoai giai thich vong lap for.\n"
        "```python\n"
        "for i in range(10):\n"
        "    print(i)\n"
        "```\n"
    )

    result = parser.parse(script)

    assert len(result.scenes) == 1
    scene = result.scenes[0]
    assert scene.scene_index == 0
    assert scene.narration_text == "Day la loi thoai giai thich vong lap for."
    assert scene.illustration_hint == "minh hoa vong lap for"
    assert scene.code_snippet == "for i in range(10):\n    print(i)"


def test_parses_multiple_scenes_sequential(parser):
    script = "## Scene 1\nnarration one\n\n## Scene 2\nnarration two\n"

    result = parser.parse(script)

    assert [s.scene_index for s in result.scenes] == [0, 1]
    assert result.scenes[0].narration_text == "narration one"
    assert result.scenes[1].narration_text == "narration two"


def test_illustration_hint_is_optional(parser):
    script = "## Scene 1\njust narration, no hint\n"

    result = parser.parse(script)

    assert result.scenes[0].illustration_hint is None


def test_code_snippet_is_optional(parser):
    script = "## Scene 1\njust narration, no code\n"

    result = parser.parse(script)

    assert result.scenes[0].code_snippet is None


def test_content_before_first_heading_is_ignored(parser):
    script = "# My Project\nsome notes\n\n## Scene 1\nnarration\n"

    result = parser.parse(script)

    assert len(result.scenes) == 1
    assert result.scenes[0].narration_text == "narration"


def test_no_headings_raises_no_scenes_found(parser):
    with pytest.raises(ScriptSyntaxError) as exc_info:
        parser.parse("just plain text, no scene heading")

    assert exc_info.value.reason == "no scenes found"
    assert exc_info.value.line_number is None


def test_non_sequential_numbering_raises_error(parser):
    script = "## Scene 1\nnarration one\n\n## Scene 3\nnarration two\n"

    with pytest.raises(ScriptSyntaxError) as exc_info:
        parser.parse(script)

    assert "sequential" in exc_info.value.reason
    assert exc_info.value.line_number == 4


def test_scene_starting_at_two_raises_error(parser):
    script = "## Scene 2\nnarration\n"

    with pytest.raises(ScriptSyntaxError) as exc_info:
        parser.parse(script)

    assert "sequential" in exc_info.value.reason
    assert exc_info.value.line_number == 1


def test_empty_narration_text_raises_error(parser):
    script = "## Scene 1\n> only a hint, no narration\n"

    with pytest.raises(ScriptSyntaxError) as exc_info:
        parser.parse(script)

    assert exc_info.value.reason == "narration_text must not be empty"
    assert exc_info.value.line_number == 1


def test_multiple_code_fences_raises_error(parser):
    script = "## Scene 1\nnarration\n```python\ncode one\n```\n```python\ncode two\n```\n"

    with pytest.raises(ScriptSyntaxError) as exc_info:
        parser.parse(script)

    assert "at most one code block" in exc_info.value.reason


def test_unterminated_code_fence_raises_error(parser):
    script = "## Scene 1\nnarration\n```python\ncode without closing fence\n"

    with pytest.raises(ScriptSyntaxError) as exc_info:
        parser.parse(script)

    assert exc_info.value.reason == "unterminated code block"


def test_fail_fast_stops_at_first_error(parser):
    # Scene 1 has empty narration (error) AND scene numbering also jumps to 3 —
    # only the first error encountered (scene 1's empty narration) should surface.
    script = "## Scene 1\n> hint only\n\n## Scene 3\nnarration\n"

    with pytest.raises(ScriptSyntaxError) as exc_info:
        parser.parse(script)

    assert exc_info.value.reason == "narration_text must not be empty"
