import json

import pytest

from app import storyboard as sbm

GOOD = {
    "hero": "Hình vuông",
    "palette": [{"role": "accent", "hex": "#f5b841", "meaning": "đang chú ý"}],
    "scenes": [
        {"id": "beat-1", "title": "Mở", "invariant": "x", "transition_in": None, "mood": "tò mò",
         "end_frame": "hình vuông", "shots": [
             {"id": "1.1", "camera": "trung cảnh", "visual": "mọc lên", "narration": "Câu một."},
             {"id": "1.2", "camera": "đẩy vào", "visual": "đổi màu", "narration": "Câu hai."}]},
        {"id": "beat-2", "shots": [{"id": "2.1", "visual": "v", "narration": "n"}]},
    ],
}


def test_parses_plain_fenced_and_chatter_wrapped_json():
    raw = json.dumps(GOOD)
    for text in (raw, f"```json\n{raw}\n```", f"Đây là kết quả:\n{raw}\nHết."):
        sb = sbm.parse(text)
        assert [s.id for _, s in sb.all_shots()] == ["1.1", "1.2", "2.1"]
        assert sb.palette[0].hex == "#F5B841"


def test_reports_every_problem_at_once():
    bad = json.loads(json.dumps(GOOD))
    bad["palette"].append({"role": "accent", "hex": "red"})
    bad["scenes"][1]["shots"][0]["id"] = "1.1"
    bad["scenes"][0]["shots"][1]["narration"] = "  "
    with pytest.raises(sbm.StoryboardError) as e:
        sbm.parse(json.dumps(bad))
    text = " | ".join(e.value.problems)
    assert "hex must look like" in text and "must not be empty" in text


def test_duplicate_ids_roles_and_missing_hero_are_rejected():
    bad = json.loads(json.dumps(GOOD))
    bad["scenes"][1]["shots"][0]["id"] = "1.1"
    bad["palette"].append({"role": "accent", "hex": "#000000"})
    del bad["hero"]
    with pytest.raises(sbm.StoryboardError) as e:
        sbm.parse(json.dumps(bad))
    text = " | ".join(e.value.problems)
    assert "duplicate shot id 1.1" in text and "roles must be unique" in text and "hero / world" in text


@pytest.mark.parametrize("text", ["", "not json at all", "[1,2]", "{"])
def test_garbage_is_rejected(text):
    with pytest.raises(sbm.StoryboardError):
        sbm.parse(text)


def test_prose_uses_the_readable_shot_format_and_roundtrips_through_dumps():
    sb = sbm.parse(json.dumps(GOOD))
    prose = sbm.to_prose(sb)
    assert '1.1 | MÁY: trung cảnh | HÌNH: mọc lên | THOẠI: "Câu một."' in prose
    assert prose.startswith("NHÂN VẬT CHÍNH: Hình vuông") and "#F5B841" in prose
    assert sbm.parse(sbm.dumps(sb)) == sb


# --- optional layout (visual_director_ai v3) ---------------------------------------

def with_layout(layout):
    data = json.loads(json.dumps(GOOD))
    data["layout"] = layout
    return json.dumps(data)


def test_layout_is_optional_and_a_storyboard_without_it_serialises_as_before():
    sb = sbm.parse(json.dumps(GOOD))
    assert sb.layout is None and '"layout"' not in sbm.dumps(sb)


def test_a_valid_layout_round_trips_and_shows_in_the_prose():
    sb = sbm.parse(with_layout({"hero": {"x": 960, "y": 480, "size": 320}, "counter": {"x": 1500.5, "y": 300}}))
    assert sb.layout["counter"]["x"] == 1500.5
    assert json.loads(sbm.dumps(sb))["layout"]["hero"] == {"x": 960, "y": 480, "size": 320}
    assert "hero: x=960, y=480, size=320" in sbm.to_prose(sb)


def test_layout_problems_are_all_reported():
    with pytest.raises(sbm.StoryboardError) as exc:
        sbm.parse(with_layout({
            "Hero Box": {"x": 1, "y": 2},
            "noY": {"x": 5},
            "outside": {"x": 2500, "y": 10},
            "flag": {"x": 1, "y": 1, "big": True},
            "text": {"x": "960", "y": 1},
        }))
    text = "; ".join(exc.value.problems)
    for want in ("layout.Hero Box: key must be camelCase", "layout.noY: needs numeric x and y",
                 "layout.outside: centre (2500, 10) is outside", "layout.flag.big: must be a number",
                 "layout.text.x: must be a number"):
        assert want in text
