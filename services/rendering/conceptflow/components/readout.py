"""Số chạy — một con số thay đổi được trong lúc quay (CR-017 FR45.2).

Đây là khuôn hình duy nhất trong thư viện có **trạng thái chạy theo thời gian**:
`self.tracker` là một `ValueTracker`, và con số trên màn hình bám theo nó. Script
đổi số bằng `self.count(readout, tới)` chứ không dựng lại mobject mới, nhờ vậy
các chữ số biến đổi liên tục thay vì nháy một cái sang giá trị khác.

## Vì sao có một bản "bóng" vô hình

`always_redraw` dựng lại mobject **từ đầu** mỗi frame, nên mọi thứ script làm
với nó sau đó — `.shift()`, `.next_to()`, hay phép co của `self.stack()` — đều
bị xoá ngay frame kế tiếp. Vì vậy trong group có một bản tĩnh đã ẩn đi
(`_ghost`): nó nhận mọi phép biến hình như một mobject bình thường, còn bản
sống mỗi frame lại bám vào vị trí và cỡ chữ hiện tại của nó. Không có nó thì
`self.readout(...).shift(UP * 2.6)` sẽ đẩy nhãn lên trên còn con số vẫn nằm
giữa khung — đúng lỗi mà bản đầu của component này mắc phải.

## Vì sao đơn vị nằm trong phần dựng lại

Số dài ra khi giá trị tăng (0 → 128). Nếu đơn vị là một mobject tĩnh đặt cạnh
con số lúc khởi tạo thì các chữ số mới mọc ra sẽ bò đè lên nó.

## Vì sao không dùng `unit=` của DecimalNumber, và vì sao chữ số là `Text`

`DecimalNumber(unit=...)` dựng đơn vị bằng `SingleStringMathTex`, còn `mob_class`
mặc định của nó là `MathTex` — cả hai đều đi qua LaTeX và mang font của LaTeX
vào khung hình. Một video mà chữ dùng Be Vietnam Pro còn các con số lại dùng
Computer Modern thì nhìn ra ngay là hai nguồn khác nhau. Cùng lý do khiến
`BarChart` không dùng `manim.BarChart`.
"""

from __future__ import annotations

from functools import partial

from manim import DOWN, RIGHT, DecimalNumber, Text, ValueTracker, VGroup, always_redraw

from ..theme import Theme
from .base import Component
from .callout import TONES


class Readout(Component):
    """Một con số lớn (kèm nhãn và đơn vị) mà script animate được.

    `.tracker` là `ValueTracker` phía sau; `.value` đọc giá trị hiện tại.
    """

    def __init__(
        self,
        value: float = 0,
        label: str | None = None,
        unit: str = "",
        decimals: int = 0,
        tone: str = "accent",
        theme: Theme | None = None,
    ) -> None:
        super().__init__(theme=theme)
        t = self.theme
        self._color = getattr(t, TONES.get(tone, "accent"))
        self._decimals = decimals
        self._unit = unit
        self.tracker = ValueTracker(value)

        self._ghost = self._reading(value, t.scale.h1)
        parts: list = [self._ghost]
        if label:
            parts.append(
                Text(label, font=t.fonts.body, font_size=t.scale.caption, color=t.muted)
                .next_to(self._ghost, DOWN, buff=0.3)
            )
        self.add(*parts)
        # Co cho vừa khung TRƯỚC khi ẩn bóng: sau khi ẩn, `finish()` vẫn đo được
        # nó (mobject trong suốt vẫn có bounding box), nhưng làm trước thì thứ
        # tự đọc đúng thứ tự — đo cái nhìn thấy được, rồi mới giấu nó đi.
        self.finish()
        self._ghost.set_opacity(0)
        self._base_size = self._ghost[0].font_size

        self.number = always_redraw(self._live)
        # Thêm vào đầu để nhãn vẫn nằm trên nếu có chỗ giao nhau.
        self.add_to_back(self.number)

    @property
    def value(self) -> float:
        return float(self.tracker.get_value())

    def _live(self) -> VGroup:
        """Bản vẽ của frame hiện tại, bám theo vị trí và cỡ của bóng.

        `DecimalNumber.font_size` quy đổi theo chiều cao hiện tại, nên nó trả về
        đúng cỡ đã co nếu group từng bị `fit()`/`stack()` thu nhỏ. Giữa một
        animation vẽ dần (`Create`) chiều cao có thể bằng 0 trong vài frame đầu,
        và cỡ chữ 0 thì `DecimalNumber` ném `ValueError` — rơi về cỡ ban đầu ở
        những frame đó thay vì làm hỏng cả lượt render.
        """
        size = self._ghost[0].font_size or self._base_size
        return self._reading(self.tracker.get_value(), size).move_to(self._ghost)

    def _reading(self, value: float, font_size: float) -> VGroup:
        t = self.theme
        digits = DecimalNumber(
            value,
            num_decimal_places=self._decimals,
            mob_class=partial(Text, font=t.fonts.body),
            font_size=font_size,
            color=self._color,
        )
        group = VGroup(digits)
        if self._unit:
            # Cỡ đơn vị bám theo cỡ số (tỉ lệ body/h1 của theme) để cả cụm co
            # cùng nhau khi group bị thu nhỏ.
            unit_size = font_size * t.scale.body / t.scale.h1
            group.add(
                Text(self._unit, font=t.fonts.body, font_size=unit_size, color=self._color)
                .next_to(digits, RIGHT, buff=0.18, aligned_edge=DOWN)
            )
        return group
