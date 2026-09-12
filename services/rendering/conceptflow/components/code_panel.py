"""Khối code có nhãn ngôn ngữ."""

from __future__ import annotations

from manim import DOWN, LEFT, Code, Text

from ..theme import Theme
from .base import Component


class CodePanel(Component):
    """`Code` của Manim cộng một nhãn ngôn ngữ nhỏ phía trên.

    Nhãn tồn tại vì video lập trình thường trộn nhiều ngôn ngữ, và người xem
    không phải lúc nào cũng nhận ra cú pháp trong hai giây đầu.
    """

    def __init__(self, source: str, language: str = "python", theme: Theme | None = None) -> None:
        super().__init__(theme=theme)
        t = self.theme

        # Manim's Text mobject builds one submobject per character via Pango
        # glyph layout, which emits zero glyphs for a raw tab — the char count
        # then no longer matches the source string and Code._gen_colored_lines
        # crashes with IndexError. Expand tabs to spaces so every character
        # source code may contain actually gets a glyph.
        source = source.expandtabs(4)

        block = Code(
            code=source,
            language=language,
            style="monokai",
            background="rectangle",
            background_stroke_color=t.accent,
            background_stroke_width=1.5,
            corner_radius=0.15,
            font=t.fonts.mono,
        )
        label = Text(
            language, font=t.fonts.body, font_size=t.scale.caption, color=t.muted
        ).next_to(block, DOWN, buff=0.18).align_to(block, LEFT)

        self.add(block, label)
        self.finish()
