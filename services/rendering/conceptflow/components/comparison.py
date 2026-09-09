"""So sánh hai cột — khuôn hình lặp lại nhiều nhất trong video giải thích."""

from __future__ import annotations

from manim import DOWN, RIGHT, UP, Line, Text, VGroup

from ..theme import Theme
from .base import Component


class ComparisonSplit(Component):
    """Hai cột có tiêu đề riêng, ngăn bằng một đường dọc.

    Hai cột được ép rộng bằng nhau. Nếu để chúng tự co theo nội dung, bên nào
    chữ dài hơn sẽ chiếm nhiều chỗ hơn, và người xem đọc ra thành "bên này quan
    trọng hơn" — một thông điệp không ai định gửi.
    """

    def __init__(
        self,
        left_title: str,
        left_body: str,
        right_title: str,
        right_body: str,
        theme: Theme | None = None,
    ) -> None:
        super().__init__(theme=theme)
        t = self.theme

        left = self._column(left_title, left_body, t.series_color(0))
        right = self._column(right_title, right_body, t.series_color(1))

        width = max(left.width, right.width)
        for column in (left, right):
            if column.width < width:
                column.stretch_to_fit_width(width)

        pair = VGroup(left, right).arrange(RIGHT, buff=1.1, aligned_edge=UP)
        divider = Line(
            start=pair.get_top(), end=pair.get_bottom(), color=t.muted, stroke_width=1.5
        ).move_to(pair.get_center())

        self.add(pair, divider)
        self.finish()

    def _column(self, title: str, body: str, color: str) -> VGroup:
        t = self.theme
        head = Text(title, font=t.fonts.display, font_size=t.scale.h2, color=color)
        text = Text(body, font=t.fonts.body, font_size=t.scale.body, color=t.ink)
        return VGroup(head, text).arrange(DOWN, buff=0.35)
