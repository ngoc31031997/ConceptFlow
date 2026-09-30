"""Luật tĩnh cho một hình của thư viện minh hoạ."""

from domain.illustration_asset import validate_asset_code

GOOD = """import React from 'react';
import {useCurrentFrame} from 'remotion';
import {Figure, Face, type FigureProps} from './conceptflow-mini/illustration';

export function SchoolBus({color = '#FFC72C', ...fig}: FigureProps & {color?: string}) {
  const frame = useCurrentFrame();
  return (
    <Figure {...fig} size={fig.size ?? 320} vw={320} vh={200}>
      <rect x={10} y={20} width={300} height={140} rx={28} fill={color} />
      <Face mood="happy" blink={false} talking={false} frame={frame} cx={260} cy={125} glasses={false} />
    </Figure>
  );
}
"""


def messages(name, code):
    return [i.message for i in validate_asset_code(name, code)]


def test_hinh_dung_luat_khong_co_loi():
    assert validate_asset_code("SchoolBus", GOOD) == []


def test_ten_phai_la_pascal_case_va_khop_export():
    assert any("PascalCase" in m for m in messages("school_bus", GOOD))
    assert any("export function Truck(" in m for m in messages("Truck", GOOD))


def test_chan_import_ngoai_danh_sach_va_ghi_so_dong():
    code = "import lodash from 'lodash';\n" + GOOD
    issues = validate_asset_code("SchoolBus", code)
    assert [(i.line, "lodash" in i.message) for i in issues] == [(1, True)]


def test_chan_ten_trung_khung_script_va_ham_nguy_hiem():
    code = GOOD + "\nconst PALETTE = {};\nfunction Shot1_2() { return fetch('x'); }\n"
    msgs = messages("SchoolBus", code)
    assert any("PALETTE" in m for m in msgs)
    assert any("Shot1_2" in m for m in msgs)
    assert any("fetch" in m for m in msgs)
