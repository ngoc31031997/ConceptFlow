"""Biểu đồ cột — so sánh độ lớn giữa vài hạng mục."""

from __future__ import annotations

from manim import DOWN, LEFT, RIGHT, UP, Line, Rectangle, Text, VGroup

from ..theme import Theme
from .base import Component

#: Chiều cao của cột cao nhất, theo đơn vị Manim. Các cột khác tỉ lệ theo nó.
MAX_BAR_HEIGHT = 4.0
BAR_WIDTH = 0.9


class BarChart(Component):
    """Cột tự dựng từ `Rectangle` + `Text`, không dùng `manim.BarChart`.

    `manim.BarChart` đi qua `Axes`, mà nhãn số của `Axes` render bằng LaTeX —
    kéo thêm một phụ thuộc hệ thống chỉ để in vài con số, và font của nó không
    phải font của theme. Ở đây số được in bằng `Text` như mọi chữ khác.

    Chỉ nhận giá trị không âm: cột âm cần trục giữa khung, một khuôn hình khác
    hẳn, và gộp chung sẽ làm ca phổ biến (so sánh độ lớn) khó đọc hơn.
    """

    def __init__(
        self,
        labels: list[str],
        values: list[float],
        unit: str = "",
        theme: Theme | None = None,
    ) -> None:
        super().__init__(theme=theme)
        if not values or len(labels) != len(values):
            raise ValueError("BarChart cần labels và values cùng độ dài, không rỗng")
        if any(v < 0 for v in values):
            raise ValueError("BarChart chỉ nhận giá trị không âm")
        t = self.theme
        peak = max(values) or 1.0

        self.bars = VGroup()
        columns = VGroup()
        for i, (label, value) in enumerate(zip(labels, values)):
            color = t.series_color(i)
            bar = Rectangle(
                width=BAR_WIDTH,
                height=max(value / peak * MAX_BAR_HEIGHT, 0.02),
                stroke_width=0,
                fill_color=color,
                fill_opacity=0.9,
            )
            number = Text(
                f"{_format(value)}{unit}", font=t.fonts.body, font_size=t.scale.caption, color=color
            ).next_to(bar, UP, buff=0.15)
            name = Text(label, font=t.fonts.body, font_size=t.scale.caption, color=t.ink)
            name.next_to(bar, DOWN, buff=0.25)
            self.bars.add(bar)
            columns.add(VGroup(bar, number, name))

        # Canh đáy các cột trước khi xếp ngang, để cột thấp không lơ lửng giữa.
        for column in columns:
            column.shift(-column[0].get_bottom()[1] * UP)
        columns.arrange(RIGHT, buff=0.6, aligned_edge=DOWN)
        for column in columns[1:]:
            column.shift((columns[0][0].get_bottom()[1] - column[0].get_bottom()[1]) * UP)

        baseline = Line(
            columns.get_left() + LEFT * 0.2, columns.get_right() + RIGHT * 0.2,
            color=t.muted, stroke_width=1.5,
        ).set_y(columns[0][0].get_bottom()[1])

        self.add(columns, baseline)
        self.finish()


def _format(value: float) -> str:
    return str(int(value)) if float(value).is_integer() else f"{value:g}"
