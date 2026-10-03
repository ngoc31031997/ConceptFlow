"""Every chunk turn names the hero/world, since the code step's system prompt
carries only the story's core lines."""

import json

from app import storyboard as sbm
from app.pipeline import merger, prompts


def _sb(**top):
    doc = {
        "palette": [{"role": "accent", "hex": "#F5B841", "meaning": "đang chú ý"}],
        "scenes": [{"id": "hook", "shots": [
            {"id": "1.1", "visual": "v1", "narration": "Câu một."},
            {"id": "1.2", "visual": "v2", "narration": "Câu hai."},
            {"id": "1.3", "visual": "v3", "narration": "Câu ba."},
        ]}],
        **top,
    }
    return sbm.parse(json.dumps(doc))


def _chunk_turns(sb):
    shots = list(sb.all_shots())
    return {
        "remotion": prompts.remotion_chunk(sb, "const LAYOUT = {};", ["1.2"], shots[0], shots[2]),
        "manim": prompts.manim_chunk(sb, "def setup_cast(self):\n    pass", [], ["1.2"], shots[0], shots[2]),
    }


def test_chunk_turn_opens_with_the_hero():
    for engine, turn in _chunk_turns(_sb(hero="Chiếc răng hàm")).items():
        assert turn.startswith("NHÂN VẬT CHÍNH: Chiếc răng hàm\n\nNHIỆM VỤ HIỆN TẠI: VIẾT CODE CHO SHOT 1.2"), engine


def test_chunk_turn_opens_with_the_world_when_there_is_no_hero():
    for engine, turn in _chunk_turns(_sb(world="Bề mặt một chiếc răng phóng to")).items():
        assert turn.startswith("THẾ GIỚI: Bề mặt một chiếc răng phóng to\n\n"), engine
        assert "NHÂN VẬT CHÍNH" not in turn, engine


def test_chunk_turn_lists_the_palette_key_the_storyboard_uses():
    # One name per colour, the one the visuals say.
    sb = sbm.parse(json.dumps({
        "hero": "Khối mô hình",
        "palette": [{"role": "conNguoi", "hex": "#F5B841", "meaning": "con người"}],
        "scenes": [{"id": "hook", "shots": [
            {"id": "1.1", "visual": "thẻ màu conNguoi", "narration": "Câu một."}]}],
    }))
    turn = prompts.remotion_chunk(sb, "const LAYOUT = {};", ["1.1"], None, None)
    assert "- PALETTE.conNguoi = #F5B841" in turn
    assert "connguoi" not in turn


def test_palette_is_written_the_way_the_code_reads_it():
    sb = _sb(hero="h")
    repair = prompts.remotion_repair(sb, "const LAYOUT = {};", "1.2", "function Shot1_2() {}", [], [])
    for turn in (_chunk_turns(sb)["remotion"], repair, prompts.remotion_layout(sb)):
        assert "- PALETTE.accent = #F5B841 — accent: đang chú ý" in turn


def test_chunk_turn_lists_every_name_the_frame_imports():
    turn = _chunk_turns(_sb(hero="h"))["remotion"]
    listed = turn[turn.index("- các dòng import ("):].split("\n", 1)[0]
    head_names = set()
    for line in merger._REMOTION_HEAD.splitlines():
        if line.startswith("import {"):
            head_names.update(n.strip() for n in line[len("import {"):line.index("}")].split(","))
    shot_names = head_names - set(merger.ILLUSTRATION_HELPERS) - set(merger.SCENE_HELPERS) \
        - {"registerRoot", "Composition", "calculateMetadataFromSegments", "Segments"}
    for name in sorted(shot_names):
        assert f" {name}" in listed, f"{name} is imported by the frame but not listed in the chunk turn"
    for name in ("random", "useFrameBox", "BACKGROUND", "Scene", "Camera", "MeadowBackdrop"):
        assert name in listed, f"{name} is missing from the chunk turn's import list"
