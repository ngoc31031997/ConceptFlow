"""Bộ minh hoạ phẳng: code, tài liệu trong prompt và danh sách import
của Code Merger phải khớp nhau.

Ba nơi cùng mô tả một bộ component:
  - remotion_project/src/conceptflow-mini/illustration.tsx — code thật;
  - authoring-service/.../prompts/illustration_kit_vi.txt — thứ Kỹ sư Remotion đọc;
  - llm-service/app/pipeline/merger.py ILLUSTRATION_KIT — thứ khung code import sẵn.
Lệch một trong ba là: model gọi một component không tồn tại (build lỗi), hoặc có
component mà không ai biết để dùng (vẽ tay lại thứ đã có).
"""

import re
import shutil
from pathlib import Path

import pytest

from adapters.rendering.typescript_checker import TypeScriptChecker

RENDERING = Path(__file__).resolve().parents[2]
SERVICES = RENDERING.parent
KIT = RENDERING / "remotion_project" / "src" / "conceptflow-mini" / "illustration.tsx"
DOC = SERVICES / "authoring-service" / "internal" / "domain" / "prompts" / "illustration_kit_vi.txt"
MERGER = SERVICES / "llm-service" / "app" / "pipeline" / "merger.py"

# Tên thẻ JSX hợp lệ trong tài liệu mà không thuộc bộ minh hoạ.
NOT_KIT_TAGS = {"AbsoluteFill"}


# Linh kiện để vẽ hình mới, không phải hình để đặt lên khung.
HELPERS = {"Figure", "Face", "GroundShadow"}


def all_exports() -> set[str]:
    text = KIT.read_text(encoding="utf-8")
    return set(re.findall(r"^export (?:function|const) (\w+)", text, re.M))


def kit_exports() -> set[str]:
    return {n for n in all_exports() if n[0].isupper() and not n.isupper()} - HELPERS


def test_bo_minh_hoa_co_du_cac_hinh_toi_thieu():
    # Những hình mà ví dụ trong prompt và luật "minh hoạ đúng cái đang nói" dựa vào.
    assert {"Backdrop", "Panel", "Person", "Tooth", "Germ", "Toothbrush", "Candy", "Bubble"} <= kit_exports()


@pytest.mark.skipif(not DOC.exists(), reason="authoring-service không có trong cây này")
def test_moi_component_deu_duoc_mo_ta_trong_prompt_va_nguoc_lai():
    doc = DOC.read_text(encoding="utf-8")
    exports = kit_exports()
    documented = set(re.findall(r"<([A-Z]\w*)", doc)) - NOT_KIT_TAGS
    assert exports - documented == set(), "component chưa có trong illustration_kit_vi.txt"
    assert documented - exports == set(), "prompt nhắc tới component không tồn tại"


@pytest.mark.skipif(not MERGER.exists(), reason="llm-service không có trong cây này")
def test_khung_code_cua_merger_import_dung_bo_minh_hoa():
    text = MERGER.read_text(encoding="utf-8")
    block = re.search(r"ILLUSTRATION_KIT = \((.*?)\)", text, re.S)
    assert block, "merger.py không còn ILLUSTRATION_KIT"
    imported = set(re.findall(r'"(\w+)"', block.group(1)))
    assert imported == kit_exports()
    helpers = re.search(r"ILLUSTRATION_HELPERS = \((.*?)\)", text, re.S)
    assert helpers, "merger.py không còn ILLUSTRATION_HELPERS"
    # Mọi thứ còn lại mà bộ minh hoạ xuất ra: hình của thư viện được dán vào script cần chúng.
    assert set(re.findall(r'"(\w+)"', helpers.group(1))) == all_exports() - kit_exports()


REAL = RENDERING / "remotion_project"
needs_real = pytest.mark.skipif(
    not (REAL / "node_modules" / "typescript").exists() or shutil.which("node") is None,
    reason="remotion_project/node_modules is not installed here",
)


@needs_real
def test_shot_dung_bo_minh_hoa_qua_duoc_tsc():
    names = ", ".join(sorted(kit_exports()))
    script = f"""import React from 'react';
import {{AbsoluteFill, interpolate, useCurrentFrame}} from 'remotion';
import {{{names}}} from './conceptflow-mini/illustration';

const PALETTE = {{room: '#FFC857', night: '#140B3A', germ: '#8BC34A'}};
const LAYOUT = {{kid: {{x: 400, y: 610, size: 440}}}};

export function Shot1_1({{duration}}: {{duration: number}}) {{
  const frame = useCurrentFrame();
  const decay = interpolate(frame, [0, duration], [0, 1],
    {{extrapolateLeft: 'clamp', extrapolateRight: 'clamp'}});
  return (
    <AbsoluteFill>
      <Backdrop color={{PALETTE.room}} floor={{PALETTE.night}} floorY={{830}} />
      <Person {{...LAYOUT.kid}} pose="ouch" mood="pain" age="child" talking />
      <Panel x={{990}} y={{0}} w={{930}} h={{1080}} color={{PALETTE.night}}>
        <Tooth x={{400}} y={{500}} size={{360}} decay={{decay}} />
        <Germ x={{200}} y={{300}} size={{140}} color={{PALETTE.germ}} variant={{2}} flip />
      </Panel>
      <Bubble x={{1500}} y={{300}} w={{300}} h={{120}} thought>Ối!</Bubble>
    </AbsoluteFill>
  );
}}
"""
    ts = TypeScriptChecker(REAL, timeout_seconds=90)
    try:
        assert ts.check(script) == []
        bad = script.replace('pose="ouch"', 'pose="dance"')
        assert [d.code for d in ts.check(bad)] == ["TS2322"]
    finally:
        ts.close()
