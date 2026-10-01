"""Bộ kiểm tra style: mỗi luật có ca vi phạm, và hình mẫu chuẩn phải sạch hoàn toàn."""

import re
from pathlib import Path

import pytest

from domain.illustration_style import check_style

# Ba hình mẫu gốc là dữ liệu trong DB của authoring-service; bản
# code của chúng được giữ ở đây làm dữ liệu kiểm thử cho bộ kiểm tra style.
EXEMPLARS = Path(__file__).resolve().parents[1] / "fixtures" / "illustration_exemplars_vi.txt"

CLEAN = """import React from 'react';
import {useCurrentFrame} from 'remotion';
import {Figure, Face, shadeOf, type FigureProps} from './conceptflow-mini/illustration';

export function Blob({color = '#FF9F43', ...fig}: FigureProps & {color?: string}) {
  const frame = useCurrentFrame();
  const bob = fig.still ? 0 : Math.sin(frame / 14) * 2;
  return (
    <Figure {...fig} size={fig.size ?? 200} vw={200} vh={200}>
      <rect x={20} y={40 + bob} width={160} height={140} rx={40} fill={color} />
      <rect x={120} y={40} width={60} height={140} rx={30} fill={shadeOf(color, -0.12)} />
      <Face mood="happy" blink={false} talking={false} frame={frame} cx={100} cy={100} glasses={false} />
    </Figure>
  );
}
"""


def rules(code):
    errors, warnings = check_style(code)
    return sorted(f.rule for f in errors), sorted(f.rule for f in warnings)


def exemplars():
    text = EXEMPLARS.read_text(encoding="utf-8")
    return re.findall(r"=== (\w+) .*?===\n```tsx\n(.*?)```", text, re.S)


def test_hinh_sach_khong_co_gi():
    assert rules(CLEAN) == ([], [])


def test_ba_hinh_mau_chuan_sach_hoan_toan():
    found = exemplars()
    assert [n for n, _ in found] == ["SchoolBus", "Cat", "Microscope"]
    for name, code in found:
        assert rules(code) == ([], []), name


@pytest.mark.parametrize("change, rule", [
    ('<Face', '<pattern id={p} /><Face'),
    ('<Face', '<filter id={f}><feDropShadow dx={2} /></filter><Face'),
    ('<Face', '<text x={1}>Hi</text><Face'),
    ('<Face', '<image href="x.png" /><Face'),
    ("Math.sin(frame / 14)", "Math.random()"),
])
def test_loi_chan_luu(change, rule):
    errors, _ = rules(CLEAN.replace(change, rule, 1))
    assert errors, (change, rule)


def test_thieu_figure_la_loi():
    code = CLEAN.replace("<Figure {...fig} size={fig.size ?? 200} vw={200} vh={200}>", "<svg>")
    code = code.replace("</Figure>", "</svg>")
    assert "S16" in rules(code)[0]


@pytest.mark.parametrize("change, replacement, rule", [
    ('rx={40} fill={color}', 'rx={40} fill={color} stroke="#3A1F4B"', "S2"),
    ('rx={40} fill={color}', 'fill={color}', "S3"),
    ("fig.still ? 0 : ", "", "S20"),
])
def test_canh_bao_khong_chan(change, replacement, rule):
    errors, warnings = rules(CLEAN.replace(change, replacement, 1))
    assert errors == []
    assert rule in warnings


def test_mau_ngoai_bang_mau_kenh_khong_canh_bao():
    # Hình minh hoạ dùng màu của chính vật; bảng màu kênh chỉ là gợi ý.
    errors, warnings = rules(CLEAN.replace("'#FF9F43'", "'#123456'", 1))
    assert errors == [] and warnings == []


def test_qua_nhieu_mau_va_mau_khong_doi_duoc():
    many = "".join(f'<circle cx={{1}} cy={{1}} r={{1}} fill="{c}" />' for c in
                   ["#FFC857", "#FF9F43", "#E8453C", "#FF5C8A", "#7B3FC4", "#2D5BFF", "#2BB673"])
    code = CLEAN.replace("<Face", many + "<Face")
    props = "{color = '#FF9F43', ...fig}: FigureProps & {color?: string}"
    code = code.replace(props, "{...fig}: FigureProps")
    code = code.replace("fill={color}", 'fill="#FF9F43"')
    _, warnings = rules(code)
    assert "S10" in warnings and "S12" in warnings


GRADIENT = """  const glow = useSvgId('glow');
  const blur = useSvgId('blur');
  return (
    <Figure {...fig} size={fig.size ?? 200} vw={200} vh={200}>
      <defs>
        <radialGradient id={glow}>
          <stop offset="0" stopColor={color} /><stop offset="1" stopColor={color} stopOpacity={0} />
        </radialGradient>
        <filter id={blur}><feGaussianBlur stdDeviation={6} /></filter>
      </defs>
      <circle cx={100} cy={100} r={90} fill={`url(#${glow})`} filter={`url(#${blur})`} />"""


def with_gradient(code: str = CLEAN) -> str:
    code = code.replace("{Figure, Face,", "{Figure, Face, useSvgId,", 1)
    return code.replace("""  return (
    <Figure {...fig} size={fig.size ?? 200} vw={200} vh={200}>""", GRADIENT, 1)


def test_gradient_va_lam_mo_qua_useSvgId_la_sach():
    assert rules(with_gradient()) == ([], [])


def test_id_viet_cung_canh_bao_S25():
    errors, warnings = rules(with_gradient().replace("id={glow}", 'id="glow"', 1))
    assert errors == [] and warnings == ["S25"]


def test_qua_nhieu_gradient_canh_bao():
    extra = "".join(f"<linearGradient id={{g{i}}} />" for i in range(4))
    errors, warnings = rules(with_gradient().replace("</defs>", extra + "</defs>", 1))
    assert errors == [] and warnings == ["S1"]


BACKDROP = """import React from 'react';
import {useVideoConfig} from 'remotion';
import {shadeOf} from './conceptflow-mini/illustration';
import {useLayerBox} from './conceptflow-mini/scene';

export function OrchardBackdrop({layer, color = '#8FD3FF'}: {layer: 'sky' | 'far' | 'mid' | 'near'; color?: string}) {
  const {width, height} = useVideoConfig();
  const box = useLayerBox();
  if (layer !== 'sky') return null;
  return (
    <svg viewBox={`${box.left} ${box.top} ${box.width} ${box.height}`}>
      <rect x={box.left} y={box.top} width={box.width} height={box.height} rx={0} fill={shadeOf(color, -0.1)} />
      <circle cx={width * 0.8} cy={height * 0.2} r={60} fill="#FFD23F" />
    </svg>
  );
}
"""


def test_mot_nen_khong_can_figure_va_khong_canh_bao_prop_color():
    errors, _ = rules_kind(BACKDROP)
    assert errors == []


def test_nen_thieu_layer_hoac_kich_thuoc_khung_hoac_co_mat_la_loi():
    no_layer = BACKDROP.replace("{layer, color = '#8FD3FF'}: {layer: 'sky' | 'far' | 'mid' | 'near'; color?: string}",
                                "{color = '#8FD3FF'}: {color?: string}").replace("if (layer !== 'sky') ", "")
    assert "B1" in rules_kind(no_layer)[0]
    fixed = BACKDROP.replace("useLayerBox", "fixedBox").replace("useVideoConfig", "fixedSize")
    assert "B2" in rules_kind(fixed)[0]
    assert "B5" in rules_kind(BACKDROP.replace("</svg>", "<Face mood=\"happy\" /></svg>"))[0]


def rules_kind(code):
    errors, warnings = check_style(code, "backdrop")
    return sorted(f.rule for f in errors), sorted(f.rule for f in warnings)
