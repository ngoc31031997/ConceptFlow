"""Dòng thời gian — các mốc trên một trục ngang."""

from __future__ import annotations

from manim import DOWN, UP, Dot, Line, Text, VGroup

from ..theme import Theme
from .base import Component

SPACING = 2.6


class Timeline(Component):
    """Mỗi mốc là `(thời_điểm, mô_tả)` hoặc chỉ một chuỗi mô tả.

    Thời điểm nằm trên trục, mô tả nằm dưới: mắt quét hàng trên để thấy nhịp
    thời gian, hàng dưới để đọc chuyện gì xảy ra — hai tầng thông tin không
    chen vào nhau. `.marks[i]` trỏ tới từng mốc.
    """

    def __init__(self, events: list[object], theme: Theme | None = None) -> None:
        super().__init__(theme=theme)
        if not events:
            raise ValueError("Timeline cần ít nhất một mốc")
        t = self.theme

        self.marks = VGroup()
        for i, event in enumerate(events):
            when, what = event if isinstance(event, (tuple, list)) else ("", event)
            color = t.series_color(i)
            dot = Dot(point=[i * SPACING, 0, 0], radius=0.12, color=color)
            parts = [dot]
            if when:
                parts.append(
                    Text(str(when), font=t.fonts.body, font_size=t.scale.caption, color=color)
                    .next_to(dot, UP, buff=0.25)
                )
            parts.append(
                Text(str(what), font=t.fonts.body, font_size=t.scale.caption, color=t.ink)
                .next_to(dot, DOWN, buff=0.3)
            )
            self.marks.add(VGroup(*parts))

        axis = Line([-0.6, 0, 0], [(len(events) - 1) * SPACING + 0.6, 0, 0], color=t.muted, stroke_width=2)
        self.add(axis, self.marks)
        self.finish()
