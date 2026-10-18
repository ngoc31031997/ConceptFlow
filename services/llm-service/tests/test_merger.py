import json
import re

import pytest

from app import storyboard as sbm
from app.pipeline import merger
from tests.lights import light

SB = sbm.parse(json.dumps({
    "hero": "h",
    "palette": [{"role": "màu nhấn", "hex": "#F5B841", "meaning": "chú ý"},
                {"role": "Đã xong", "hex": "#4A5670", "meaning": "chìm"}],
    "scenes": [
        {"id": "hook", "light": light("màu nhấn"), "shots": [{"id": "1.1", "visual": "v", "narration": 'Câu "một".'},
                                  {"id": "1.2", "visual": "v", "narration": "Câu hai."}]},
        {"id": "recap", "light": light("màu nhấn"), "shots": [{"id": "2.1", "visual": "v", "narration": "Câu ba."}]},
    ],
}))
TSX = {i: f"// Shot {i}\nfunction {merger.remotion_fn(i)}({{duration}}: ShotProps) {{\n  return null;\n}}" for i in ("1.1", "1.2", "2.1")}


def test_palette_keys_are_ascii_camel_and_unique():
    assert merger.palette_keys(SB) == {"màu nhấn": "mauNhan", "Đã xong": "daXong"}


def test_palette_keys_keep_the_camel_case_a_role_already_has():
    # The model reads `conNguoi` in every visual; a lowered key `connguoi` would
    # make it write a PALETTE key that does not exist (TS2551).
    sb = sbm.parse(json.dumps({
        "hero": "h",
        "palette": [{"role": r, "hex": "#000000", "meaning": "m"}
                    for r in ("conNguoi", "NenPanel", "RED", "vàng khao khát", "3d glow", "màu NHẤN")],
        "scenes": [{"id": "s", "light": light("conNguoi"), "shots": [{"id": "1.1", "visual": "v", "narration": "n"}]}],
    }))
    assert list(merger.palette_keys(sb).values()) == [
        "conNguoi", "nenPanel", "red", "vangKhaoKhat", "c3dGlow", "mauNhan"]
    assert "  conNguoi: '#000000'," in merger.remotion_frame_text(sb)["palette"]


def test_remotion_narrations_shots_and_order_come_from_the_storyboard():
    m = merger.merge_remotion(SB, "const LAYOUT = {hero: {x: 1, y: 2, size: 3}};", TSX)
    assert "  \"Câu \\\"một\\\".\"," in m.code
    assert "const SHOTS: React.FC<ShotProps>[] = [Shot1_1, Shot1_2, Shot2_1];" in m.code
    assert "mauNhan: '#F5B841'" in m.code and 'id="creator"' in m.code
    # The prompt tells the model LottieClip is already imported, so the frame must import it.
    assert "import {LottieClip} from './conceptflow-mini/lottie';" in m.code
    # Likewise the whole illustration kit.
    assert "import {Backdrop, Panel, Person, Tooth, Germ," in m.code
    assert "} from './conceptflow-mini/illustration';" in m.code
    assert m.code.index("const LAYOUT") < m.code.index("function Shot1_1")


def test_line_map_points_at_the_right_lines():
    m = merger.merge_remotion(SB, "const LAYOUT = {};", TSX)
    all_lines = m.code.splitlines()
    la, lb = m.lines[merger.LAYOUT_KEY]
    assert all_lines[la - 1:lb] == ["const LAYOUT = {};"]
    for sid, (a, b) in m.lines.items():
        if sid == merger.LAYOUT_KEY:
            continue
        assert all_lines[a - 1].startswith(("// Shot", "function Shot")), sid
        assert all_lines[b - 1] == "}", sid
        assert m.shot_at(a) == sid and m.shot_at(b) == sid
    assert m.shot_at(1) is None


def test_missing_shot_is_refused_not_papered_over():
    import pytest
    with pytest.raises(ValueError, match=r"\['2.1'\]"):
        merger.merge_remotion(SB, "x", {k: v for k, v in TSX.items() if k != "2.1"})


def test_layout_from_storyboard_is_a_plain_ts_declaration():
    assert merger.layout_from_storyboard(SB) is None
    sb = sbm.parse(json.dumps({**json.loads(sbm.dumps(SB)), "layout": {
        "hero": {"x": 960, "y": 480.0, "size": 320}, "dot": {"x": 10.5, "y": 20}}}))
    assert merger.layout_from_storyboard(sb) == (
        "const LAYOUT = {\n  hero: {x: 960, y: 480, size: 320},\n  dot: {x: 10.5, y: 20},\n};")
    assert merger.layout_from_storyboard(sb.model_copy(update={"layout": {}})) == "const LAYOUT = {\n};"


def test_remotion_stubs_the_shots_a_chunk_does_not_own():
    only = {"1.2": TSX["1.2"]}
    try:
        merger.merge_remotion(SB, "const LAYOUT = {};", only)
    except ValueError:
        pass
    else:
        raise AssertionError("missing shots must still be refused without stub_missing")
    m = merger.merge_remotion(SB, "const LAYOUT = {};", only, stub_missing=True)
    assert "function Shot1_1({duration, lines}: ShotProps) {\n  return null;\n}" in m.code
    assert "const SHOTS: React.FC<ShotProps>[] = [Shot1_1, Shot1_2, Shot2_1];" in m.code
    a, b = m.lines["1.2"]
    assert m.code.splitlines()[a - 1] == "// Shot 1.2"


def test_frame_imports_every_name_the_rule_book_offers_the_shots():
    # A shot may call remotion's seeded `random` and read the frame box without
    # any library drawing having imported them first.
    code = merger.merge_remotion(SB, "const LAYOUT = {};", TSX).code
    imports = [line for line in code.splitlines() if line.startswith("import ")]
    remotion = next(line for line in imports if line.endswith("from 'remotion';"))
    primitives = next(line for line in imports if line.endswith("from './conceptflow-mini/primitives';"))
    assert ", random}" in remotion
    assert "BACKGROUND" in primitives and "useFrameBox" in primitives


def test_frame_names_cover_every_import_of_the_head():
    imported = {"React"}
    for clause in re.findall(r"import (?:type )?\{([^}]*)\}", merger._REMOTION_HEAD):
        imported.update(n.strip() for n in clause.split(",") if n.strip())
    names = merger.frame_names()
    assert imported <= names, f"the head imports names frame_names() lacks: {sorted(imported - names)}"
    assert {"useCurrentFrame", "PALETTE", "LAYOUT", "clamp"} <= names
    assert "useCurrentFrameSafe" not in names


def test_looks_are_built_from_the_scene_light_and_only_use_palette_entries():
    text = merger.looks_text(SB)
    assert text.startswith("const LOOKS: Record<string, Look> = {")
    assert "scene1: {light: {x: 0.7, y: 0.2, color: PALETTE.mauNhan, intensity: 0.8}" in text
    assert "scene2:" in text  # shot 2.1 belongs to the second scene
    assert "ambient: PALETTE.mauNhan" in text and "atmosphere: {color: PALETTE.mauNhan, density: 0.3}" in text
    assert "grade: {warmth: 0.1, contrast: 0, saturation: 0}" in text


def test_the_merged_script_declares_the_looks_after_the_layout():
    shots = {
        sh.id: f"function {merger.remotion_fn(sh.id)}({{duration}}: ShotProps) {{ return null; }}"
        for _, sh in SB.all_shots()
    }
    merged = merger.merge_remotion(SB, "const LAYOUT = {};", shots)
    assert merged.code.index("const LAYOUT") < merged.code.index("const LOOKS")


def test_a_light_that_names_a_role_outside_the_palette_is_refused():
    doc = json.loads(sbm.dumps(SB))
    doc["scenes"][0]["light"]["color"] = "ghost"
    with pytest.raises(sbm.StoryboardError, match="light.color 'ghost'"):
        sbm.parse(json.dumps(doc))


def test_a_scene_without_light_is_refused():
    doc = json.loads(sbm.dumps(SB))
    del doc["scenes"][0]["light"]
    with pytest.raises(sbm.StoryboardError, match="light"):
        sbm.parse(json.dumps(doc))
