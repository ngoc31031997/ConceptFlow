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

from . import narration as narration_runtime
from . import theme as theme_module
from .components import Recap, TitleCard
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

    # --- Lời thoại (CR-018) ---------------------------------------------------

    def narrate(self, text: str) -> None:
        """Phát một đoạn lời thoại ngay tại đây.

        Dùng được bên trong vòng lặp, nhánh điều kiện và hàm helper — đó là
        điểm khác biệt với `# NARRATION` + `self.wait(AUTO)` mà nó thay thế, và
        là thứ cho phép hook/CTA trở thành component thật (CR-019).
        """
        narration_runtime.narrate(self, text)

    def beat(self, beat_id: str) -> None:
        """Mở một beat của beat sheet (CR-019). Gắn vào lời thoại kế tiếp."""
        narration_runtime.beat(self, beat_id)

    def chapter(self, title: str) -> None:
        """Mở một chapter YouTube (CR-006 FR15). Gắn vào lời thoại kế tiếp."""
        narration_runtime.chapter(self, title)

    # --- Beat dựng sẵn (CR-019 FR53) ------------------------------------------
    #
    # Ba beat này là khuôn hình lặp lại ở MỌI video, nên chúng là method chứ
    # không phải thứ Creator dựng lại mỗi lần. Chúng chỉ khả thi sau CR-018:
    # trước đó lời thoại là comment phải đếm khớp theo thứ tự dòng, nên không
    # thể nằm trong một hàm — đúng lý do CR-006 §Quyết định #2 phải lùi FR17
    # xuống thành snippet Creator tự chép.

    def hook(self, question: str, subtitle: str | None = None) -> None:
        """Mở đầu: một câu hỏi hoặc nghịch lý, hiện bằng hình rồi mới nói.

        Nội dung do Creator truyền vào, KHÔNG tự sinh từ tiêu đề video: tiêu đề
        được soạn ở bước publish, sau khi render, nên tại đây nó chưa tồn tại
        (cùng lý do khiến thumbnail tự động không burn chữ — CR-006 §Quyết định #3).
        """
        self.beat("hook")
        card = TitleCard(question, subtitle, theme=self.theme)
        self.reveal(card)
        self.narrate(question)
        self.dismiss(card)

    def recap(self, points: list[str], title: str = "Tóm lại", narration: str | None = None) -> None:
        """Màn tóm tắt: nhắc lại bằng hình, không phải danh sách gạch đầu dòng."""
        self.beat("recap")
        panel = Recap(points, title=title, theme=self.theme)
        self.reveal(panel)
        self.narrate(narration or ". ".join(points))
        self.dismiss(panel)

    def call_to_action(self, message: str, subtitle: str | None = None, hold_seconds: float = 8.0) -> None:
        """Kêu gọi hành động, rồi giữ khung cuối.

        `hold_seconds` là số cụ thể chứ không phải lời thoại: đây là khoảng lặng
        để YouTube có chỗ hiện end-screen element (CR-006 FR17.1).
        """
        self.beat("cta")
        card = TitleCard(message, subtitle, theme=self.theme)
        self.reveal(card)
        self.narrate(message)
        self.wait(hold_seconds)

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

