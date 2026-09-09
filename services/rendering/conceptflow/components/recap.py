"""Màn tóm tắt cuối video."""

from __future__ import annotations

from manim import DOWN, LEFT, RIGHT, Square, Text, VGroup

from ..theme import Theme
from .base import Component


class Recap(Component):
    """Tiêu đề cộng các ý đã đi qua, mỗi ý có một dấu hình học.

    Dùng ô vuông xoay 45 độ thay cho dấu tick: tick ngụ ý "đã hoàn thành", còn
    đây là "đã nói qua" — và một hình học đơn giản giữ được sự sạch sẽ của màn
    cuối, vốn là khung hình đọng lại lâu nhất.
    """

    def __init__(self, points: list[str], title: str = "Tóm lại", theme: Theme | None = None) -> None:
        super().__init__(theme=theme)
        t = self.theme

        head = Text(title, font=t.fonts.display, font_size=t.scale.h2, color=t.ink)

        rows = VGroup()
        for index, point in enumerate(points):
            marker = Square(side_length=0.16, stroke_width=0, fill_opacity=1).set_fill(
                t.series_color(index)
            ).rotate(0.7853981633974483)
            label = Text(point, font=t.fonts.body, font_size=t.scale.body, color=t.ink)
            rows.add(VGroup(marker, label).arrange(RIGHT, buff=0.32))

        rows.arrange(DOWN, buff=0.34, aligned_edge=LEFT)
        body = VGroup(head, rows).arrange(DOWN, buff=0.5, aligned_edge=LEFT)

        self.add(body)
        self.finish()
