"""Sanity tests for domain value objects (domain-entities.md)."""

from __future__ import annotations

from domain.models import ParsedScript, Scene


def test_scene_fields():
    scene = Scene(scene_index=0, narration_text="hello", illustration_hint="hint", code_snippet="code")
    assert scene.scene_index == 0
    assert scene.narration_text == "hello"
    assert scene.illustration_hint == "hint"
    assert scene.code_snippet == "code"


def test_scene_optional_fields_can_be_none():
    scene = Scene(scene_index=0, narration_text="hello", illustration_hint=None, code_snippet=None)
    assert scene.illustration_hint is None
    assert scene.code_snippet is None


def test_parsed_script_holds_scenes():
    scenes = [Scene(0, "a", None, None), Scene(1, "b", None, None)]
    parsed = ParsedScript(scenes=scenes)
    assert parsed.scenes == scenes
