"""Bộ kiểm tra style (CR-044): mỗi luật có ca vi phạm, và hình mẫu chuẩn phải sạch hoàn toàn."""

import re
from pathlib import Path

import pytest

from domain.illustration_style import check_style

EXEMPLARS = (Path(__file__).resolve().parents[3] / "authoring-service" / "internal" / "domain" / "prompts"
             / "illustration_exemplars_vi.txt")

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


@pytest.mark.skipif(not EXEMPLARS.exists(), reason="authoring-service không có trong cây này")
def test_ba_hinh_mau_chuan_sach_hoan_toan():
    found = exemplars()
    assert [n for n, _ in found] == ["SchoolBus", "Cat", "Microscope"]
    for name, code in found:
        assert rules(code) == ([], []), name


@pytest.mark.parametrize("change, rule", [
    ('fill={color} />', 'fill="url(#g)" /><linearGradient id="g" />'),
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
    ("'#FF9F43'", "'#123456'", "S9"),
    ("fig.still ? 0 : ", "", "S20"),
])
def test_canh_bao_khong_chan(change, replacement, rule):
    errors, warnings = rules(CLEAN.replace(change, replacement, 1))
    assert errors == []
    assert rule in warnings


def test_qua_nhieu_mau_va_mau_khong_doi_duoc():
    many = "".join(f'<circle cx={{1}} cy={{1}} r={{1}} fill="{c}" />' for c in
                   ["#FFC857", "#FF9F43", "#E8453C", "#FF5C8A", "#7B3FC4", "#2D5BFF", "#2BB673"])
    code = CLEAN.replace("<Face", many + "<Face")
    props = "{color = '#FF9F43', ...fig}: FigureProps & {color?: string}"
    code = code.replace(props, "{...fig}: FigureProps")
    code = code.replace("fill={color}", 'fill="#FF9F43"')
    _, warnings = rules(code)
    assert "S10" in warnings and "S12" in warnings
