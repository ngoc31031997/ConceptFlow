"""Khung ngang/dọc của rendering khớp `FRAMES` của conceptflow-mini/primitives.tsx."""

import re
from pathlib import Path

from domain.frame import LANDSCAPE, PORTRAIT, frame_for

PRIMITIVES = (Path(__file__).resolve().parents[2] / "remotion_project" / "src" / "conceptflow-mini"
              / "primitives.tsx")


def ts_frame(name: str) -> tuple[int, ...]:
    text = PRIMITIVES.read_text(encoding="utf-8")
    m = re.search(
        name + r": \{width: (\d+), height: (\d+), safe: \{left: (\d+), top: (\d+), "
        r"right: (\d+), bottom: (\d+)\}\}",
        text,
    )
    assert m, f"primitives.tsx không còn khung {name}"
    return tuple(int(v) for v in m.groups())


def as_tuple(frame) -> tuple[int, ...]:
    s = frame.safe
    return (frame.width, frame.height, s.left, s.top, s.right, s.bottom)


def test_hai_khung_trung_so_voi_typescript():
    assert as_tuple(LANDSCAPE) == ts_frame("landscape")
    assert as_tuple(PORTRAIT) == ts_frame("portrait")


def test_khung_doc_khi_cao_hon_rong():
    assert frame_for(1080, 1920) is PORTRAIT
    assert frame_for(1920, 1080) is LANDSCAPE
