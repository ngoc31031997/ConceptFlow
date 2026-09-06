"""Unit tests for ManimScriptParser."""

from __future__ import annotations

import pytest

from adapters.parsing.manim_script_parser import ManimScriptParser
from domain.errors import ScriptSyntaxError

VALID_SCRIPT = """from manim import *


class ForLoopIntroScene(Scene):
    def construct(self):
        # NARRATION: "Gioi thieu vong lap for trong Java"
        self.wait(AUTO)

        # NARRATION: "Bien i chay tu 0 den 4"
        self.wait(AUTO)
"""


def test_parses_scene_class_name_and_narration_order():
    parsed = ManimScriptParser().parse(VALID_SCRIPT)

    assert parsed.scene_class_name == "ForLoopIntroScene"
    assert [s.narration_text for s in parsed.scenes] == [
        "Gioi thieu vong lap for trong Java",
        "Bien i chay tu 0 den 4",
    ]
    assert [s.scene_index for s in parsed.scenes] == [0, 1]


def test_finds_scene_subclass_with_extra_base_classes():
    script = (
        "class MyScene(MovingCameraScene):\n"
        "    def construct(self):\n"
        '        # NARRATION: "hi"\n'
    )
    parsed = ManimScriptParser().parse(script)
    assert parsed.scene_class_name == "MyScene"


def test_raises_when_no_scene_class_found():
    with pytest.raises(ScriptSyntaxError, match="Scene subclass"):
        ManimScriptParser().parse('# NARRATION: "hi"\nprint("no scene here")')


def test_raises_when_no_narration_markers_found():
    with pytest.raises(ScriptSyntaxError, match="narration markers"):
        ManimScriptParser().parse("class DemoScene(Scene):\n    def construct(self):\n        pass\n")
