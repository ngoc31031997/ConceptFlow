"""Bảng dữ liệu — hàng tiêu đề cộng các hàng giá trị."""

from __future__ import annotations

from manim import DOWN, LEFT, RIGHT, Line, Text, VGroup

from ..theme import Theme
from .base import Component

CELL_PAD_X = 0.5
ROW_GAP = 0.32


class DataTable(Component):
    """Bảng tự dựng bằng `Text` + `Line`, không dùng `manim.Table`.

    `manim.Table` nhận mobject hoặc chuỗi LaTeX cho từng ô và vẽ lưới đủ bốn
    phía; bảng trên video đọc dễ hơn khi chỉ có một đường kẻ dưới tiêu đề —
    lưới dày làm mắt đọc đường kẻ thay vì đọc số. `.rows[i]` trỏ tới từng hàng
    dữ liệu để script `emphasize` hàng đang nói tới.
    """

    def __init__(
        self,
        header: list[str],
        rows: list[list[object]],
        theme: Theme | None = None,
    ) -> None:
        super().__init__(theme=theme)
        if not header:
            raise ValueError("DataTable cần hàng tiêu đề")
        if any(len(row) != len(header) for row in rows):
            raise ValueError("Mỗi hàng của DataTable phải có đúng số cột của header")
        t = self.theme

        head_cells = [
            Text(str(h), font=t.fonts.body, font_size=t.scale.body, color=t.accent, weight="BOLD")
            for h in header
        ]
        body_cells = [
            [Text(str(v), font=t.fonts.body, font_size=t.scale.body, color=t.ink) for v in row]
            for row in rows
        ]

        widths = [
            max(cell.width for cell in [head_cells[c], *(r[c] for r in body_cells)])
            for c in range(len(header))
        ]

        def place(cells: list[Text]) -> VGroup:
            x = 0.0
            for cell, width in zip(cells, widths):
                cell.move_to([x + cell.width / 2, 0, 0])
                x += width + 2 * CELL_PAD_X
            return VGroup(*cells)

        self.header = place(head_cells)
        self.rows = VGroup(*(place(cells) for cells in body_cells))
        table = VGroup(self.header, *self.rows).arrange(DOWN, buff=ROW_GAP, aligned_edge=LEFT)

        rule = Line(
            table.get_left() + LEFT * 0.2, table.get_right() + RIGHT * 0.2,
            color=t.muted, stroke_width=1.5,
        ).next_to(self.header, DOWN, buff=ROW_GAP / 2)
        rule.set_x(table.get_center()[0])

        self.add(table, rule)
        self.finish()
