"""Đồ thị hàm số — trục toạ độ cộng một đường cong."""

from __future__ import annotations

from typing import Callable

from manim import DOWN, RIGHT, UR, Axes, Text

from ..theme import Theme
from .base import Component

SAMPLES = 200


class FunctionPlot(Component):
    """Vẽ `func` trên `x_range`, trục y tự tính từ giá trị của hàm.

    Trục không in số (`include_numbers=False`): số trên trục của Manim đi qua
    LaTeX, và một video giải thích khái niệm hiếm khi cần đọc toạ độ chính xác —
    hình dạng đường cong mới là thông điệp. Cần mốc cụ thể thì gắn `Callout`.

    `.axes` và `.graph` để script `emphasize` đường cong hoặc lấy điểm qua
    `plot.axes.c2p(x, y)` khi muốn đặt nhãn.
    """

    def __init__(
        self,
        func: Callable[[float], float],
        x_range: tuple[float, float] = (-3.0, 3.0),
        label: str | None = None,
        x_label: str = "x",
        y_label: str = "y",
        theme: Theme | None = None,
    ) -> None:
        super().__init__(theme=theme)
        t = self.theme
        x_min, x_max = float(x_range[0]), float(x_range[1])
        if x_max <= x_min:
            raise ValueError("FunctionPlot cần x_range tăng dần")

        y_min, y_max = _y_bounds(func, x_min, x_max)
        self.axes = Axes(
            x_range=[x_min, x_max, (x_max - x_min) / 6],
            y_range=[y_min, y_max, (y_max - y_min) / 4],
            x_length=9,
            y_length=5,
            axis_config={"color": t.muted, "stroke_width": 2, "include_numbers": False},
            tips=True,
        )
        self.graph = self.axes.plot(func, x_range=[x_min, x_max], color=t.accent, stroke_width=4)

        x_text = Text(x_label, font=t.fonts.body, font_size=t.scale.caption, color=t.muted)
        y_text = Text(y_label, font=t.fonts.body, font_size=t.scale.caption, color=t.muted)
        x_text.next_to(self.axes.x_axis.get_end(), DOWN, buff=0.2)
        y_text.next_to(self.axes.y_axis.get_end(), RIGHT, buff=0.2)

        self.add(self.axes, self.graph, x_text, y_text)
        if label:
            name = Text(label, font=t.fonts.body, font_size=t.scale.body, color=t.accent)
            name.next_to(self.graph.get_end(), UR, buff=0.15)
            self.add(name)
        self.finish()


def _y_bounds(func: Callable[[float], float], x_min: float, x_max: float) -> tuple[float, float]:
    """Khoảng y chứa đường cong, có đệm 10%, và luôn gồm y=0 để trục x hiện ra."""
    step = (x_max - x_min) / SAMPLES
    ys = []
    for i in range(SAMPLES + 1):
        try:
            y = float(func(x_min + i * step))
        except (ArithmeticError, ValueError):
            continue
        if y == y and abs(y) != float("inf"):
            ys.append(y)
    if not ys:
        raise ValueError("FunctionPlot: hàm không có giá trị hữu hạn nào trên x_range")
    low, high = min(min(ys), 0.0), max(max(ys), 0.0)
    if high - low < 1e-9:
        low, high = low - 1.0, high + 1.0
    pad = (high - low) * 0.1
    return low - pad, high + pad
