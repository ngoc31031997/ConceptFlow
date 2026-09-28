"""CR-048 T3 — every chunk turn names the hero/world, now that the code step's
system prompt carries only the story's core lines."""

import json

from app import storyboard as sbm
from app.pipeline import prompts


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
