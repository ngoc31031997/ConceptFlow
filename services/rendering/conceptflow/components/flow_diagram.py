"""Sơ đồ luồng — các khối nối nhau bằng mũi tên."""

from __future__ import annotations

from manim import DOWN, RIGHT, Arrow, RoundedRectangle, Text, VGroup

from ..theme import Theme
from .base import Component

#: Hướng -> (hướng xếp, cạnh ra của khối trước, cạnh vào của khối sau).
DIRECTIONS = {
    "right": (RIGHT, RIGHT, -RIGHT),
    "down": (DOWN, DOWN, -DOWN),
}


class FlowDiagram(Component):
    """Pipeline, vòng đời, luồng dữ liệu: "A rồi tới B rồi tới C".

    Khác `StepList` ở chỗ nó vẽ **quan hệ** giữa các bước chứ không chỉ liệt kê:
    mũi tên là thứ người xem theo mắt được khi narration nói "rồi dữ liệu đi
    sang...". Mỗi khối được `.nodes[i]` trỏ tới để script `emphasize` riêng.
    """

    def __init__(self, steps: list[str], direction: str = "right", theme: Theme | None = None) -> None:
        super().__init__(theme=theme)
        if not steps:
            raise ValueError("FlowDiagram cần ít nhất một bước")
        t = self.theme
        arrange_dir, out_edge, in_edge = DIRECTIONS.get(direction, DIRECTIONS["right"])

        self.nodes = VGroup(*(self._node(step, t.series_color(i)) for i, step in enumerate(steps)))
        self.nodes.arrange(arrange_dir, buff=0.9)

        self.arrows = VGroup(
            *(
                Arrow(
                    a.get_critical_point(out_edge),
                    b.get_critical_point(in_edge),
                    buff=0.1,
                    color=t.muted,
                    stroke_width=3,
                )
                for a, b in zip(self.nodes, self.nodes[1:])
            )
        )

        self.add(self.nodes, self.arrows)
        self.finish()

    def _node(self, text: str, color: str) -> VGroup:
        t = self.theme
        label = Text(text, font=t.fonts.body, font_size=t.scale.body, color=t.ink)
        box = RoundedRectangle(
            width=label.width + 0.6,
            height=label.height + 0.5,
            corner_radius=0.15,
            stroke_color=color,
            stroke_width=2.5,
            fill_color=t.surface,
            fill_opacity=0.55,
        ).move_to(label)
        return VGroup(box, label)
