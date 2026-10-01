"""Hai khung hình của một video và vùng an toàn của từng khung.

Video dài dựng ở khung ngang 1920x1080; short dựng ở khung dọc 1080x1920, vùng
an toàn chừa chỗ cho giao diện Shorts (tiêu đề, kênh ở đáy; cột nút ở mép phải).
Cùng số với `FRAMES` của `remotion_project/src/conceptflow-mini/primitives.tsx`,
`authoring-service/internal/domain/frame.go` và `llm-service/app/frame.py`;
test của module này đối chiếu với file TypeScript.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class SafeArea:
    """Hình chữ nhật mọi vật có nghĩa phải nằm trong, tính bằng px từ góc trên-trái."""

    left: int
    top: int
    right: int
    bottom: int


@dataclass(frozen=True)
class Frame:
    """Kích thước khung và vùng an toàn của nó."""

    width: int
    height: int
    safe: SafeArea


LANDSCAPE = Frame(1920, 1080, SafeArea(96, 96, 1824, 984))
PORTRAIT = Frame(1080, 1920, SafeArea(72, 200, 940, 1560))
MAX_HEIGHT = max(LANDSCAPE.height, PORTRAIT.height)


def frame_for(width: float, height: float) -> Frame:
    """Khung của một composition rộng `width`, cao `height`: dọc khi cao hơn rộng."""
    return PORTRAIT if height > width else LANDSCAPE
