import json

from app import storyboard as sbm
from app.pipeline import merger

SB = sbm.parse(json.dumps({
    "hero": "h",
    "palette": [{"role": "màu nhấn", "hex": "#F5B841", "meaning": "chú ý"},
                {"role": "Đã xong", "hex": "#4A5670", "meaning": "chìm"}],
    "scenes": [
        {"id": "hook", "shots": [{"id": "1.1", "visual": "v", "narration": 'Câu "một".'},
                                  {"id": "1.2", "visual": "v", "narration": "Câu hai."}]},
        {"id": "recap", "shots": [{"id": "2.1", "visual": "v", "narration": "Câu ba."}]},
    ],
}))
TSX = {i: f"// Shot {i}\nfunction {merger.remotion_fn(i)}({{duration}}: ShotProps) {{\n  return null;\n}}" for i in ("1.1", "1.2", "2.1")}
PY = {i: f"def {merger.manim_fn(i)}(self):\n    self.narrate('x')" for i in ("1.1", "1.2", "2.1")}


def test_palette_keys_are_ascii_camel_and_unique():
    assert merger.palette_keys(SB) == {"màu nhấn": "mauNhan", "Đã xong": "daXong"}


def test_remotion_narrations_shots_and_order_come_from_the_storyboard():
    m = merger.merge_remotion(SB, "const LAYOUT = {hero: {x: 1, y: 2, size: 3}};", TSX)
    assert "  \"Câu \\\"một\\\".\"," in m.code
    assert "const SHOTS: React.FC<ShotProps>[] = [Shot1_1, Shot1_2, Shot2_1];" in m.code
    assert "mauNhan: '#F5B841'" in m.code and 'id="creator"' in m.code
    # The prompt tells the model LottieClip is already imported, so the frame must import it.
    assert "import {LottieClip} from './conceptflow-mini/lottie';" in m.code
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


def test_manim_scene_calls_every_shot_in_order_with_beats():
    m = merger.merge_manim(SB, "TCP và UDP", "def setup_cast(self):\n    self.hero = None", PY)
    assert "class TcpVaUdpScene(ConceptFlowScene):" in m.code and m.scene_class_name == "TcpVaUdpScene"
    body = m.code.split("def setup_cast")[0]
    order = [ln.strip() for ln in body.splitlines() if "self." in ln]
    assert order == ['self.setup_cast()', 'self.beat("hook")', 'self.shot_1_1()', 'self.shot_1_2()',
                     'self.beat("recap")', 'self.shot_2_1()']
    lines = m.code.splitlines()
    ca, cb = m.lines[merger.CAST_KEY]
    assert lines[ca - 1] == "    def setup_cast(self):" and lines[cb - 1] == "        self.hero = None"
    for sid, (a, _b) in m.lines.items():
        if sid == merger.CAST_KEY:
            continue
        assert lines[a - 1].startswith(f"    def shot_{sid.replace('.', '_')}(self)")


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
    assert "function Shot1_1({duration}: ShotProps) {\n  return null;\n}" in m.code
    assert "const SHOTS: React.FC<ShotProps>[] = [Shot1_1, Shot1_2, Shot2_1];" in m.code
    a, b = m.lines["1.2"]
    assert m.code.splitlines()[a - 1] == "// Shot 1.2"
