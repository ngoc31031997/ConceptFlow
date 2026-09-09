"""Danh sách bước — thay cho gạch đầu dòng chữ trơn."""

from __future__ import annotations

from manim import DOWN, LEFT, RIGHT, Circle, Text, VGroup

from ..theme import Theme
from .base import Component


class StepList(Component):
    """Mỗi bước là một số trong vòng tròn cộng một dòng chữ.

    Số trong vòng tròn chứ không phải dấu chấm: nó cho phép narration nhắc "bước
    hai" và người xem tìm được ngay, đồng thời đưa vào khung hình một phần tử
    hình học — đúng thứ FR46.5 đòi ở mỗi beat.
    """

    def __init__(self, items: list[str], theme: Theme | None = None) -> None:
        super().__init__(theme=theme)
        t = self.theme

        rows = VGroup()
        for index, item in enumerate(items, start=1):
            color = t.series_color(index - 1)
            bullet = Circle(radius=0.22, stroke_color=color, stroke_width=2.5, fill_opacity=0)
            number = Text(
                str(index), font=t.fonts.body, font_size=t.scale.caption, color=color
            ).move_to(bullet)
            label = Text(item, font=t.fonts.body, font_size=t.scale.body, color=t.ink)
            row = VGroup(VGroup(bullet, number), label).arrange(RIGHT, buff=0.3)
            rows.add(row)

        rows.arrange(DOWN, buff=0.38, aligned_edge=LEFT)
        self.add(rows)
        self.finish()
