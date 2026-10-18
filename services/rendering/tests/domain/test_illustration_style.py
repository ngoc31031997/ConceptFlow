"""Bộ kiểm tra style: mỗi luật có ca vi phạm, và hình mẫu chuẩn phải sạch hoàn toàn."""

import json
from pathlib import Path

import pytest

from domain.illustration_style import check_style

SKELETONS = json.loads(
    (Path(__file__).resolve().parents[2] / "remotion_project/src/conceptflow-mini/skeletons.json").read_text()
)

SHAPES = "\n".join(
    f"      <Shape material=\"matte\" color={{color}} rect={{{{x: {i * 8}, y: 40, w: 6, h: 20}}}} />"
    for i in range(30)
)

CLEAN = f"""import React from 'react';
import {{useCurrentFrame}} from 'remotion';
import {{Figure, type FigureProps}} from './conceptflow-mini/illustration';
import {{Shape}} from './conceptflow-mini/material';

export function Blob({{color = '#FF9F43', ...fig}}: FigureProps & {{color?: string}}) {{
  const frame = useCurrentFrame();
  const bob = fig.still ? 0 : Math.sin(frame / 14) * 2;
  return (
    <Figure {{...fig}} size={{fig.size ?? 200}} vw={{200}} vh={{200}}>
      <g transform={{`translate(0 ${{bob}})`}}>
{SHAPES}
      </g>
    </Figure>
  );
}}
"""

HUMANOID_BONES = list(SKELETONS["humanoid"])


def rig_code(bones):
    inner = "\n".join(f'<Bone name="{b}" pivot={{[0, 0]}}>' for b in bones) + "\n" + "</Bone>\n" * len(bones)
    return CLEAN.replace("<g transform", f'<Rig skeleton="humanoid" height={{200}}>{inner}</Rig><g transform', 1)


def rules(code, **kwargs):
    errors, warnings = check_style(code, **kwargs)
    return sorted(f.rule for f in errors), sorted(f.rule for f in warnings)


def test_hinh_sach_khong_co_gi():
    assert rules(CLEAN) == ([], [])


@pytest.mark.parametrize("insert", [
    '<pattern id={p} />',
    '<text x={1}>Hi</text>',
    '<image href="x.png" />',
    '<filter id={f}><feGaussianBlur stdDeviation={2} /></filter>',
    '<feDropShadow dx={2} />',
])
def test_the_forbidden_tags_and_hand_written_filters_block_saving(insert):
    errors, _ = rules(CLEAN.replace("<g transform", insert + "<g transform", 1))
    assert errors and set(errors) == {"S1"}, insert


def test_random_by_the_clock_blocks_saving():
    errors, _ = rules(CLEAN.replace("Math.sin(frame / 14)", "Math.random()", 1))
    assert errors == ["S22"]


def test_thieu_figure_la_loi():
    code = CLEAN.replace("<Figure {...fig} size={fig.size ?? 200} vw={200} vh={200}>", "<svg>").replace("</Figure>", "</svg>")
    assert "S16" in rules(code)[0]


@pytest.mark.parametrize("tier, limit", [("hero", 8), ("support", 4), ("background", 0)])
def test_hand_written_gradients_are_limited_by_tier(tier, limit):
    gradients = "<linearGradient />" * (limit + 1)
    errors, _ = rules(CLEAN.replace("<g transform", gradients + "<g transform", 1), tier=tier)
    assert "S1" in errors
    at_limit = "<linearGradient />" * limit
    errors, _ = rules(CLEAN.replace("<g transform", at_limit + "<g transform", 1), tier=tier)
    assert "S1" not in errors


@pytest.mark.parametrize("tier, copies, warned", [
    ("hero", 1, True), ("support", 1, False), ("support", 3, True), ("background", 1, True),
])
def test_the_number_of_parts_is_judged_by_tier(tier, copies, warned):
    # CLEAN draws 30 shapes; `copies` repeats them (3 copies = 90 shapes)
    _, warnings = rules(CLEAN.replace(SHAPES, SHAPES * copies), tier=tier)
    assert ("S4" in warnings) is warned


def test_big_forms_that_are_not_shapes_get_a_material_warning():
    raw = "".join('<rect x={1} y={1} width={9} height={9} fill={color} />' for _ in range(3))
    code = CLEAN.replace(SHAPES, raw * 12)
    assert "M1" in rules(code)[1]
    assert "M1" not in rules(CLEAN)[1]


def test_a_hero_without_parts_is_told_to_split_into_parts():
    assert "T1" in rules(CLEAN, tier="hero")[1]
    with_part = CLEAN.replace("<g transform", '<Mechanism><Part name="a">x</Part></Mechanism><g transform', 1)
    assert "T1" not in rules(with_part, tier="hero")[1]


def test_a_rig_must_declare_every_bone_of_its_skeleton():
    errors, _ = rules(rig_code(HUMANOID_BONES), skeletons=SKELETONS)
    assert errors == []
    errors, _ = rules(rig_code(HUMANOID_BONES[:-2]), skeletons=SKELETONS)
    assert errors == ["R1"]
    findings, _ = check_style(rig_code(HUMANOID_BONES[:-2]), skeletons=SKELETONS)
    assert "thighR" in findings[0].message and "shinR" in findings[0].message
    assert "armL" not in findings[0].message


def test_a_rig_with_an_unknown_skeleton_is_an_error():
    code = CLEAN.replace("<g transform", '<Rig skeleton="dragon" height={200}></Rig><g transform', 1)
    findings, _ = check_style(code, skeletons=SKELETONS)
    assert findings and findings[0].rule == "R1" and "dragon" in findings[0].message


def test_an_unknown_tier_is_refused():
    with pytest.raises(ValueError, match="hạng"):
        check_style(CLEAN, tier="star")


@pytest.mark.parametrize("change, replacement, rule", [
    ("<g transform", '<rect fill={color} stroke="#3A1F4B" /><g transform', "S2"),
    ("fig.still ? 0 : ", "", "S20"),
])
def test_canh_bao_khong_chan(change, replacement, rule):
    errors, warnings = rules(CLEAN.replace(change, replacement, 1))
    assert errors == []
    assert rule in warnings


def test_mau_ngoai_bang_mau_kenh_khong_canh_bao():
    errors, warnings = rules(CLEAN.replace("'#FF9F43'", "'#123456'", 1))
    assert errors == [] and warnings == []


def test_mau_chinh_khong_doi_duoc_qua_prop_color():
    props = "{color = '#FF9F43', ...fig}: FigureProps & {color?: string}"
    assert "S12" in rules(CLEAN.replace(props, "{...fig}: FigureProps"))[1]


def test_id_viet_cung_canh_bao_S25():
    code = CLEAN.replace("<g transform", '<clipPath id="a" /><g transform', 1)
    assert "S25" in rules(code)[1]


BACKDROP = """import React from 'react';
import {useVideoConfig} from 'remotion';
import {useLayerBox} from './conceptflow-mini/scene';

export function Dusk({layer, color, ground, light}) {
  const {width, height} = useVideoConfig();
  const box = useLayerBox();
  if (layer !== 'sky') return null;
  return <svg width={box.width} height={box.height}><rect width={width} height={height} fill={color} /></svg>;
}
"""


def test_a_backdrop_that_follows_the_rules_is_clean():
    errors, _ = check_style(BACKDROP, kind="backdrop")
    assert errors == []


def test_a_backdrop_without_a_layer_prop_is_an_error():
    errors, _ = rules(BACKDROP.replace("{layer, color, ground, light}", "{color, ground, light}"), kind="backdrop")
    assert errors == ["B1"]


def test_a_backdrop_with_a_fixed_frame_size_is_an_error():
    code = BACKDROP.replace("import {useVideoConfig} from 'remotion';\n", "").replace(
        "import {useLayerBox} from './conceptflow-mini/scene';\n", "").replace(
        "const {width, height} = useVideoConfig();\n  const box = useLayerBox();",
        "const width = 1920, height = 1080, box = {width, height};",
    )
    errors, _ = rules(code, kind="backdrop")
    assert errors == ["B2"]


def test_a_backdrop_with_a_face_is_an_error():
    errors, _ = rules(BACKDROP.replace("return <svg", "return <Eyes /> || <svg"), kind="backdrop")
    assert errors == ["B5"]
