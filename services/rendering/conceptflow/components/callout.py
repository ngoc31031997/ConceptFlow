"""Chú thích nhấn mạnh — một ý, đóng khung, có màu theo sắc thái."""

from __future__ import annotations

from manim import RoundedRectangle, Text

from ..theme import Theme
from .base import Component

#: Sắc thái -> tên thuộc tính màu trên Theme. Script chọn bằng từ khoá, không
#: bằng mã màu, nên không có đường nào để một màu lạ lọt vào khung hình.
TONES = {
    "accent": "accent",
    "success": "success",
    "warning": "warning",
    "danger": "danger",
}


class Callout(Component):
    def __init__(self, text: str, tone: str = "accent", theme: Theme | None = None) -> None:
        super().__init__(theme=theme)
        t = self.theme
        color = getattr(t, TONES.get(tone, "accent"))

        label = Text(text, font=t.fonts.body, font_size=t.scale.body, color=t.ink)
        box = RoundedRectangle(
            width=label.width + 0.8,
            height=label.height + 0.6,
            corner_radius=0.18,
            stroke_color=color,
            stroke_width=2.5,
            fill_color=t.surface,
            fill_opacity=0.55,
        ).move_to(label)

        self.add(box, label)
        self.finish()
