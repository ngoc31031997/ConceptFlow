"""Thẻ tiêu đề — khuôn hình mở đầu mọi phân đoạn."""

from __future__ import annotations

from manim import DOWN, Line, Text

from ..theme import Theme
from .base import Component


class TitleCard(Component):
    """Tiêu đề, gạch nhấn, phụ đề tuỳ chọn.

    Gạch nhấn không phải trang trí: nó là thứ duy nhất mang màu accent trong một
    khuôn hình toàn chữ, và là cách rẻ nhất để hai video khác chủ đề trông cùng
    một kênh.
    """

    def __init__(self, title: str, subtitle: str | None = None, theme: Theme | None = None) -> None:
        super().__init__(theme=theme)
        t = self.theme

        heading = Text(title, font=t.fonts.display, font_size=t.scale.h1, color=t.ink)
        rule = Line(
            start=heading.get_left(), end=heading.get_right(), color=t.accent, stroke_width=3
        ).next_to(heading, DOWN, buff=0.22)

        self.add(heading, rule)
        if subtitle:
            sub = Text(
                subtitle, font=t.fonts.body, font_size=t.scale.body, color=t.muted
            ).next_to(rule, DOWN, buff=0.3)
            self.add(sub)

        self.finish()
