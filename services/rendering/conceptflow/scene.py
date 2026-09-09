"""`ConceptFlowScene` — nền chung của mọi video trên kênh (CR-017 FR45.1).

Script của Creator kế thừa class này thay vì `Scene`, và nhờ đó nhận được nền,
font, bảng màu và nhịp chuyển cảnh mà không phải khai báo gì. Đó là toàn bộ ý
tưởng của CR-017: bản sắc nằm trong code, không nằm trong prompt.

Các factory chữ (`title`, `body`, ...) tồn tại để script **không bao giờ gọi
`Text(...)` trực tiếp** — mỗi lần gọi trực tiếp là một lần cỡ chữ và màu có cơ
hội trôi khỏi chuẩn. Lint (FR46.4) cảnh báo đúng việc đó.
"""

from __future__ import annotations

from manim import DOWN, RIGHT, Code, MathTex, Mobject, Scene, Text, VGroup

from . import theme as theme_module
from .layout import Box, fit_scale
from .theme import Theme
from .transitions import (
    dismiss_animation,
    emphasize_animation,
    reveal_animation,
    swap_animation,
)


class ConceptFlowScene(Scene):
    """Base scene mang theme của kênh.

    Đặt `theme_name` ở class con để dùng theme khác; bỏ trống thì lấy mặc định.
    """

    theme_name: str | None = None

    def __init__(self, *args, **kwargs) -> None:
        super().__init__(*args, **kwargs)
        self.theme: Theme = theme_module.get(self.theme_name)
        # Đặt ngay trong __init__ chứ không đợi construct(): script con override
        # construct() và gần như chắc chắn sẽ quên gọi super().
        self.camera.background_color = self.theme.background

    # --- Chữ ------------------------------------------------------------------

    def title(self, text: str, **kwargs) -> Text:
        """Tiêu đề lớn. Font display chỉ dùng ở đây và `heading`."""
        return self._text(text, self.theme.scale.h1, self.theme.fonts.display, **kwargs)

    def heading(self, text: str, **kwargs) -> Text:
        return self._text(text, self.theme.scale.h2, self.theme.fonts.display, **kwargs)

    def body(self, text: str, **kwargs) -> Text:
        return self._text(text, self.theme.scale.body, self.theme.fonts.body, **kwargs)

    def caption(self, text: str, **kwargs) -> Text:
        return self._text(
            text, self.theme.scale.caption, self.theme.fonts.body,
            color=kwargs.pop("color", self.theme.muted), **kwargs,
        )

    def formula(self, latex: str, **kwargs) -> MathTex:
        """Công thức dùng LaTeX mặc định của Manim.

        Không ép font display lên đây: `MathTex` đi qua LaTeX chứ không qua
        Pango, nên đổi font là đổi cả gói chữ toán, việc đó vượt phạm vi CR-017.
        """
        kwargs.setdefault("color", self.theme.ink)
        return MathTex(latex, **kwargs)

    def code(self, source: str, language: str = "python", **kwargs) -> Code:
        kwargs.setdefault("style", "monokai")
        kwargs.setdefault("background", "rectangle")
        kwargs.setdefault("background_stroke_color", self.theme.accent)
        kwargs.setdefault("corner_radius", 0.15)
        return Code(code=source, language=language, **kwargs)

    def _text(self, text: str, size: int, font: str, **kwargs) -> Text:
        kwargs.setdefault("color", self.theme.ink)
        return Text(text, font=font, font_size=size, **kwargs)

    # --- Bố cục ---------------------------------------------------------------

    def stack(self, *mobjects: Mobject, buff: float = 0.35) -> VGroup:
        """Xếp dọc, canh giữa, rồi co cho vừa khung an toàn."""
        group = VGroup(*mobjects).arrange(DOWN, buff=buff)
        return self.fit(group)

    def row(self, *mobjects: Mobject, buff: float = 0.6) -> VGroup:
        group = VGroup(*mobjects).arrange(RIGHT, buff=buff)
        return self.fit(group)

    def fit(self, mobject: Mobject) -> Mobject:
        """Co lại nếu tràn khung an toàn (FR45.3).

        Chỉ thu nhỏ, không bao giờ phóng to: nếu phóng to thì cùng một đoạn chữ
        sẽ hiện ở cỡ khác nhau tuỳ độ dài, và thang cỡ chữ mất hết ý nghĩa.
        """
        factor = fit_scale(Box.from_mobject(mobject), self.theme.safe_margin)
        if factor < 1.0:
            mobject.scale(factor)
        return mobject

    # --- Chuyển cảnh ----------------------------------------------------------

    def reveal(self, *mobjects: Mobject, speed: str = "normal") -> None:
        run_time = self._run_time(speed)
        self.play(*(reveal_animation(m, run_time) for m in mobjects))

    def dismiss(self, *mobjects: Mobject, speed: str = "fast") -> None:
        run_time = self._run_time(speed)
        self.play(*(dismiss_animation(m, run_time) for m in mobjects))

    def swap(self, old: Mobject, new: Mobject, speed: str = "normal") -> None:
        self.play(swap_animation(old, new, self._run_time(speed)))

    def emphasize(self, *mobjects: Mobject, speed: str = "normal") -> None:
        run_time = self._run_time(speed)
        self.play(*(emphasize_animation(m, run_time) for m in mobjects))

    def clear_stage(self, speed: str = "fast") -> None:
        """Dọn sạch khung. Gọi giữa hai beat để không tích tụ rác thị giác."""
        if self.mobjects:
            self.dismiss(*self.mobjects, speed=speed)

    def _run_time(self, speed: str) -> float:
        pacing = self.theme.pacing
        return {"fast": pacing.fast, "normal": pacing.normal, "slow": pacing.slow}.get(
            speed, pacing.normal
        )

