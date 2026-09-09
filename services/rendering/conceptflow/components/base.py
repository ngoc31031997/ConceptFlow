"""Nền chung của mọi component (CR-017 FR45.2, FR45.3).

Component là `VGroup` dựng sẵn theo theme, tự giữ mình trong khung an toàn. Điểm
quan trọng: chúng nhận **theme** chứ không nhận toạ độ hay cỡ chữ — script không
có cách nào truyền `font_size=37` vào đây, và đó là chủ ý.
"""

from __future__ import annotations

from manim import VGroup

from ..layout import Box, fit_scale
from ..theme import DEFAULT, Theme


class Component(VGroup):
    """VGroup biết theme của mình và tự co cho vừa khung an toàn."""

    def __init__(self, theme: Theme | None = None, **kwargs) -> None:
        super().__init__(**kwargs)
        self.theme = theme or DEFAULT

    def finish(self) -> "Component":
        """Gọi ở cuối mỗi __init__ của component con.

        Tách thành bước riêng thay vì làm tự động vì component chỉ đo được kích
        thước sau khi đã thêm hết phần tử con — co sớm thì co nhầm.
        """
        factor = fit_scale(Box.from_mobject(self), self.theme.safe_margin)
        if factor < 1.0:
            self.scale(factor)
        return self
