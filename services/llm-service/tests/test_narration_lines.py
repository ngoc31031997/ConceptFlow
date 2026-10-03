"""Narration lines: a shot read as several TTS clips, each with its own change on screen."""

import json

import pytest

from app import storyboard as sbm
from app.pipeline import merger, prompts

BASE = {
    "hero": "h",
    "palette": [{"role": "accent", "hex": "#F5B841", "meaning": "chú ý"}],
    "scenes": [
        {"id": "hook", "shots": [
            {"id": "1.1", "visual": "v", "lines": [
                {"say": "Mười hai chia hết cho hai.", "show": "12 tách thành 2 nhóm"},
                {"say": "Cho ba, cho bốn.", "show": "nhóm lại thành 3 rồi 4"},
            ]},
            {"id": "1.2", "visual": "v", "narration": "Ba."},
        ]},
    ],
}
SB = sbm.parse(json.dumps(BASE))


def _with_lines(lines):
    doc = json.loads(json.dumps(BASE))
    doc["scenes"][0]["shots"][0]["lines"] = lines
    return doc


def test_a_shot_with_lines_reads_them_in_order_and_joins_them_into_its_narration():
    shot = SB.scenes[0].shots[0]
    assert shot.spoken_lines() == ["Mười hai chia hết cho hai.", "Cho ba, cho bốn."]
    assert shot.narration == "Mười hai chia hết cho hai. Cho ba, cho bốn."
    assert '"Mười hai chia hết cho hai." / "Cho ba, cho bốn."' in sbm.to_prose(SB)
    dumped = json.loads(sbm.dumps(SB))["scenes"][0]["shots"]
    assert dumped[0]["lines"][1]["show"] == "nhóm lại thành 3 rồi 4"
    assert dumped[0]["narration"] == shot.narration


def test_a_shot_without_lines_is_one_line_and_dumps_without_them():
    shot = SB.scenes[0].shots[1]
    assert shot.spoken_lines() == ["Ba."]
    assert "lines" not in json.loads(sbm.dumps(SB))["scenes"][0]["shots"][1]


@pytest.mark.parametrize("lines, problem", [
    ([{"say": "Một.", "show": "a"}] * 5, "at most 4 lines"),
    ([{"say": " ".join(["chữ"] * 31), "show": "a"}], "at most 30 words"),
    ([{"say": "Một.", "show": "  "}], "must not be empty"),
    ([], "narration: must not be empty"),
])
def test_bad_lines_are_rejected(lines, problem):
    with pytest.raises(sbm.StoryboardError) as e:
        sbm.parse(json.dumps(_with_lines(lines)))
    assert problem in " | ".join(e.value.problems)


def test_remotion_writes_one_narration_per_line_and_the_line_count_of_each_shot():
    tsx = {i: f"function {merger.remotion_fn(i)}({{duration, lines}}: ShotProps) {{\n  return null;\n}}"
           for i in ("1.1", "1.2")}
    code = merger.merge_remotion(SB, "const LAYOUT = {};", tsx).code
    assert ('export const narrations: string[] = [\n  "Mười hai chia hết cho hai.",\n'
            '  "Cho ba, cho bốn.",\n  "Ba.",\n];') in code
    assert "export const shotLineCounts: number[] = [2, 1];" in code
    assert "type ShotProps = {duration: number; lines: number[]};" in code
    assert "evenLines(segment.durationInFrames, shotLineCounts[index] ?? 1)" in code
    assert "<Shot duration={segment.durationInFrames} lines={lines} />" in code
    assert ("import {calculateMetadataFromSegments, Segments, evenLines, lineSpan} "
            "from './conceptflow-mini/segments';") in code
    assert "import {ReachingHand} from './conceptflow-mini/rig';" in code


def test_available_names_list_the_line_helpers_and_the_rig():
    names = merger.available_names_text()
    assert "./conceptflow-mini/segments: calculateMetadataFromSegments, Segments, evenLines, lineSpan" in names
    assert "./conceptflow-mini/rig: ReachingHand" in names


def test_the_chunk_prompt_carries_each_shots_lines_and_the_lines_prop():
    text = prompts.remotion_chunk(SB, "const LAYOUT = {};", ["1.1", "1.2"], None, None)
    assert '"say": "Cho ba, cho bốn."' in text and '"show": "nhóm lại thành 3 rồi 4"' in text
    assert "function ShotN_M({duration, lines}: ShotProps)" in text
    assert "`lines[i]` là frame bắt đầu câu thoại thứ i" in text
