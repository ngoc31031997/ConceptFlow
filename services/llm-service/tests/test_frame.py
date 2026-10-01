"""The llm-service frames match FRAMES in conceptflow-mini/primitives.tsx, and
the frame reaches the storyboard check, the merged script and the plan."""

import json
import re
from pathlib import Path

import pytest

from app import storyboard as sbm
from app.frame import LANDSCAPE, PORTRAIT, frame_of
from app.pipeline import merger, prompts
from app.pipeline.run import CodeRequest, make_plan

PRIMITIVES = (Path(__file__).resolve().parents[2] / "rendering" / "remotion_project" / "src"
              / "conceptflow-mini" / "primitives.tsx")


def _storyboard(layout=None, setting=""):
    scene = {"id": "hook", "shots": [{"id": "1.1", "visual": "v", "narration": "n"}]}
    if setting:
        scene["setting"] = setting
    doc = {"hero": "h", "palette": [{"role": "nhấn", "hex": "#F5B841"}], "scenes": [scene]}
    if layout is not None:
        doc["layout"] = layout
    return json.dumps(doc, ensure_ascii=False)


@pytest.mark.skipif(not PRIMITIVES.exists(), reason="rendering is not in this tree")
def test_frames_match_the_typescript_kit():
    text = PRIMITIVES.read_text(encoding="utf-8")
    for name, frame in (("landscape", LANDSCAPE), ("portrait", PORTRAIT)):
        m = re.search(name + r": \{width: (\d+), height: (\d+), safe: \{left: (\d+), top: (\d+), "
                      r"right: (\d+), bottom: (\d+)\}\}", text)
        assert m, f"primitives.tsx has no {name} frame"
        assert tuple(int(v) for v in m.groups()) == (frame.width, frame.height, *frame.safe)


def test_only_the_two_frames_exist():
    assert frame_of(1080, 1920) is PORTRAIT
    with pytest.raises(ValueError):
        frame_of(1280, 720)


def test_layout_is_checked_against_the_frame_it_is_drawn_on():
    tall = _storyboard({"hero": {"x": 540, "y": 1500, "size": 400}})
    with pytest.raises(sbm.StoryboardError) as err:
        sbm.parse(tall)
    assert "outside the 1920x1080 frame" in str(err.value)
    assert sbm.parse(tall, PORTRAIT).layout == {"hero": {"x": 540, "y": 1500, "size": 400}}


def test_scene_setting_is_kept_and_shown_only_when_the_director_wrote_one():
    sb = sbm.parse(_storyboard(setting="lòng quả sung, quầng sáng cam"))
    assert sb.scenes[0].setting == "lòng quả sung, quầng sáng cam"
    assert "Bối cảnh: lòng quả sung, quầng sáng cam" in sbm.to_prose(sb)
    assert prompts._shot_json(sb.scenes[0], sb.scenes[0].shots[0])["scene_setting"].startswith("lòng")
    old = sbm.parse(_storyboard())
    assert "setting" not in sbm.dumps(old)
    assert "scene_setting" not in prompts._shot_json(old.scenes[0], old.scenes[0].shots[0])


def test_the_merged_script_registers_a_composition_of_the_frame_size():
    sb = sbm.parse(_storyboard())
    shot = {"1.1": "function Shot1_1({duration}: ShotProps) {\n  return null;\n}"}
    wide = merger.merge_remotion(sb, "const LAYOUT = {};", shot).code
    tall = merger.merge_remotion(sb, "const LAYOUT = {};", shot, canvas=PORTRAIT).code
    assert "width={1920}\n    height={1080}" in wide
    assert "width={1080}\n    height={1920}" in tall
    assert "from './conceptflow-mini/scene';" in wide


def test_the_portrait_frame_changes_the_fingerprints_and_the_landscape_one_does_not():
    base = dict(engine="remotion", topic="t", storyboard=_storyboard(), system="SYS")
    wide = make_plan(CodeRequest(**base), 3)
    default = make_plan(CodeRequest(**base, canvas=LANDSCAPE), 3)
    tall = make_plan(CodeRequest(**base, canvas=PORTRAIT), 3)
    assert [s.fingerprint for s in wide.segments] == [s.fingerprint for s in default.segments]
    assert all(a.fingerprint != b.fingerprint for a, b in zip(wide.segments, tall.segments, strict=True))
