"""`ConceptFlowScene` — nền chung của mọi video trên kênh (CR-017 FR45.1).

Script của Creator kế thừa class này thay vì `Scene`, và nhờ đó nhận được nền,
font, bảng màu và nhịp chuyển cảnh mà không phải khai báo gì. Đó là toàn bộ ý
tưởng của CR-017: bản sắc nằm trong code, không nằm trong prompt.

Các factory chữ (`title`, `body`, ...) tồn tại để script **không bao giờ gọi
`Text(...)` trực tiếp** — mỗi lần gọi trực tiếp là một lần cỡ chữ và màu có cơ
hội trôi khỏi chuẩn. Lint (FR46.4) cảnh báo đúng việc đó.
"""

from __future__ import annotations

import numpy as np
from manim import (
    DOWN,
    PI,
    RIGHT,
    ArcBetweenPoints,
    Arrow,
    Brace,
    Circle,
    Code,
    CurvedArrow,
    Dot,
    Line,
    MathTex,
    Mobject,
    MovingCameraScene,
    Polygon,
    Rectangle,
    Square,
    SurroundingRectangle,
    Text,
    VGroup,
    VMobject,
)

from . import narration as narration_runtime
from . import theme as theme_module
from .components import Readout, Recap, TitleCard
from .layout import Box, fit_scale
from .theme import TONES, Theme
from .transitions import (
    dismiss_animation,
    emphasize_animation,
    reveal_animation,
    swap_animation,
    travel_animation,
)


#: Ngưỡng chồng lấn (tỉ lệ diện tích giao nhau trên diện tích vật NHỎ HƠN
#: trong cặp). Bug report (2026-09-12): một caption không định vị chồng khít
#: lên một bảng/table đang hiện — cả hai không đọc được. 25% là ngưỡng cho
#: một cặp mobject cố ý đặt gần nhau (ví dụ nhãn sát mép một hình) mà không
#: báo động giả, trong khi vẫn bắt được trường hợp "chồng gần như hoàn toàn"
#: là bug thật. Hằng số riêng, dễ chỉnh nếu thực tế cho thấy cần khác.
OVERLAP_AREA_RATIO_THRESHOLD = 0.25

#: Độ cong của `connect(style="curved")`, tính bằng radian. Một phần ba PI đủ để
#: mũi tên vòng qua một vật nằm chắn giữa mà không thành vòng cung điệu đà.
CURVED_ARROW_ANGLE = -PI / 3

#: `focus()` zoom sao cho vật chiếm khoảng 1/1.6 bề ngang khung: đủ gần để thấy
#: chi tiết, vẫn còn chỗ cho nhãn đặt cạnh vật.
CAMERA_FOCUS_PADDING = 1.6


class ConceptFlowScene(MovingCameraScene):
    """Base scene mang theme của kênh.

    Đặt `theme_name` ở class con để dùng theme khác; bỏ trống thì lấy mặc định.

    Kế thừa `MovingCameraScene` (không phải `Scene`) để camera di chuyển được:
    script gọi `focus`/`restore_view` chứ không chạm thẳng vào
    `self.camera.frame`, nên độ zoom và nhịp vẫn nằm trong design system.
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
        kwargs.setdefault("corner_radius", self.theme.shapes.corner_radius)
        return Code(code=source, language=language, **kwargs)

    def _text(self, text: str, size: int, font: str, **kwargs) -> Text:
        kwargs.setdefault("color", self.theme.ink)
        return Text(text, font=font, font_size=size, **kwargs)

    # --- Bố cục ---------------------------------------------------------------

    def stack(self, *mobjects: Mobject, buff: float | None = None) -> VGroup:
        """Xếp dọc, canh giữa, rồi co cho vừa khung an toàn."""
        group = VGroup(*mobjects).arrange(
            DOWN, buff=self.theme.spacing.normal if buff is None else buff
        )
        return self.fit(group)

    def row(self, *mobjects: Mobject, buff: float | None = None) -> VGroup:
        group = VGroup(*mobjects).arrange(
            RIGHT, buff=self.theme.spacing.loose if buff is None else buff
        )
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

    # --- Nối và khoanh (thay cho Arrow/SurroundingRectangle thô) ---------------

    def connect(
        self,
        source: Mobject,
        target: Mobject,
        label: str | None = None,
        style: str = "straight",
        tone: str = "muted",
    ) -> VGroup:
        """Mũi tên từ `source` tới `target`, kèm nhãn nhỏ ở giữa nếu có.

        `style="curved"` vòng cung sang một bên — dùng khi đường thẳng sẽ cắt
        ngang một vật thứ ba, hoặc khi cần một mũi tên đi ngược lại (A→B thẳng,
        B→A cong) mà hai mũi tên không chồng lên nhau.

        Trả về mobject chứ không tự hiện: script quyết định lúc nào `reveal`.
        Đây là lý do phổ biến nhất khiến script phải `from manim import Arrow`.
        """
        color = self._tone_color(tone)
        if style == "curved":
            start, end = _edge_points(source, target)
            arrow = CurvedArrow(start, end, angle=CURVED_ARROW_ANGLE, color=color)
            arrow.set_stroke(width=self.theme.strokes.normal)
        else:
            arrow = Arrow(
                source, target, buff=self.theme.spacing.tight, color=color,
                stroke_width=self.theme.strokes.normal,
            )

        group = VGroup(arrow)
        if label:
            # Đặt nhãn vuông góc với trục đầu-cuối để không đè lên thân mũi tên.
            # Tính từ hai đầu chứ không từ `get_unit_vector()`: mũi tên cong là
            # một `Arc`, không phải `Line`, nên không có method đó.
            dx, dy, _ = _unit_vector(arrow.get_start(), arrow.get_end())
            mid = arrow.point_from_proportion(0.5)
            group.add(self.caption(label).next_to(
                mid, np.array([-dy, dx, 0.0]), buff=self.theme.spacing.tight
            ))
        return group

    def outline(self, mobject: Mobject, tone: str = "accent") -> Mobject:
        """Khung bao quanh một đối tượng để chỉ vào nó, màu theo sắc thái."""
        return SurroundingRectangle(
            mobject,
            color=self._tone_color(tone),
            buff=self.theme.spacing.tight,
            corner_radius=self.theme.shapes.corner_radius,
            stroke_width=self.theme.strokes.normal,
        )

    def brace(self, mobject: Mobject, label: str | None = None, direction=DOWN) -> VGroup:
        """Dấu ngoặc ôm lấy một vật, kèm nhãn — "đoạn này là ...".

        Khác `outline` ở chỗ nó chỉ vào một **chiều**: độ dài một đoạn, chiều
        cao một cột, một khoảng trên trục. Đó là lý do phổ biến thứ hai (sau
        `Arrow`) khiến script phải import API thô của Manim.
        """
        brace = Brace(mobject, direction=direction, color=self.theme.muted)
        group = VGroup(brace)
        if label:
            group.add(self.caption(label).next_to(brace, direction, buff=self.theme.spacing.tight))
        return group

    # --- Hình cơ bản (thay cho Rectangle/Circle/Line... thô) ------------------

    def shape(
        self,
        kind: str = "rect",
        tone: str = "accent",
        filled: bool = False,
        width: float = 2.4,
        height: float = 1.4,
        radius: float = 0.7,
        points: list | None = None,
    ) -> VMobject:
        """Một khối hình mang màu và độ dày nét của theme.

        `kind`: "rect", "square", "circle", "dot", "polygon" (cần `points`).
        `filled=True` thêm nền `surface` mờ — cùng chất liệu với `Callout` và
        các khối của `FlowDiagram`, nên hình tự dựng không lạc khỏi phần còn lại.

        Kích thước nhận bằng tham số chứ không bằng `.scale()` sau đó, để một
        hình vuông cạnh 1.4 ở video này bằng đúng hình vuông cạnh 1.4 ở video kia.
        """
        color = self._tone_color(tone)
        builders = {
            "rect": lambda: Rectangle(width=width, height=height),
            "square": lambda: Square(side_length=height),
            "circle": lambda: Circle(radius=radius),
            "dot": lambda: Dot(radius=0.1),
            "polygon": lambda: Polygon(*(_as_point(p) for p in (points or []))),
        }
        if kind == "polygon" and len(points or []) < 3:
            raise ValueError("shape('polygon') cần ít nhất 3 điểm trong `points`")
        mobject = builders.get(kind, builders["rect"])()

        if kind == "dot":
            return mobject.set_color(color)
        mobject.set_stroke(color=color, width=self.theme.strokes.normal)
        mobject.set_fill(
            color=self.theme.surface if filled else color,
            opacity=self.theme.shapes.fill_opacity if filled else 0.0,
        )
        return mobject

    def path(self, *points, tone: str = "muted", curve: float = 0.0) -> VMobject:
        """Đường đi qua các điểm (hoặc qua tâm các vật) đã cho.

        Hai điểm và `curve != 0` thì thành cung; còn lại là đường gấp khúc. Trả
        về một đường thật (không phải mũi tên) nên dùng được làm quỹ đạo cho
        `travel()` — đó là lý do nó nhận nhiều hơn hai điểm.
        """
        anchors = [_as_point(p) for p in points]
        if len(anchors) < 2:
            raise ValueError("path() cần ít nhất hai điểm")

        if len(anchors) == 2 and curve:
            line: VMobject = ArcBetweenPoints(anchors[0], anchors[1], angle=curve)
        elif len(anchors) == 2:
            line = Line(anchors[0], anchors[1])
        else:
            line = VMobject().set_points_as_corners(anchors)
        line.set_stroke(color=self._tone_color(tone), width=self.theme.strokes.normal)
        return line

    # --- Số chạy --------------------------------------------------------------

    def readout(
        self,
        value: float = 0,
        label: str | None = None,
        unit: str = "",
        decimals: int = 0,
        tone: str = "accent",
    ) -> Readout:
        """Một con số lớn mà `count()` animate được. Xem `components.readout`."""
        return Readout(
            value, label=label, unit=unit, decimals=decimals, tone=tone, theme=self.theme
        )

    def count(self, readout: Readout, to: float, speed: str = "slow") -> None:
        """Chạy con số từ giá trị hiện tại tới `to`, không nhảy cóc.

        Mặc định `slow`: người xem cần đủ thời gian đọc được các chữ số đang
        đổi, nếu không thì hiệu ứng chỉ còn là một vệt nhoè.
        """
        self.play(readout.tracker.animate.set_value(to), run_time=self._run_time(speed))

    def _tone_color(self, tone: str) -> str:
        return getattr(self.theme, tone if tone in TONES else "accent")

    # --- Lời thoại (CR-018) ---------------------------------------------------

    # Ambient drift (CR-042 FR123.3): tắt mặc định. Bật cho cả scene bằng
    # `ambient_drift = True` hoặc từng câu bằng `narrate(..., drift=True)`.
    ambient_drift = False

    def narrate(self, text: str, *animations, drift: bool | None = None) -> None:
        """Phát một đoạn lời thoại ngay tại đây.

        Animation truyền kèm chạy TRONG lúc đọc, `run_time` = thời lượng câu
        (CR-042): `self.narrate("Một nửa khả năng biến mất.", half.animate.fade(0.9))`.
        Không kèm animation thì khung đứng yên, trừ khi bật `drift`.

        Dùng được bên trong vòng lặp, nhánh điều kiện và hàm helper — đó là
        điểm khác biệt với `# NARRATION` + `self.wait(AUTO)` mà nó thay thế, và
        là thứ cho phép hook/CTA trở thành component thật (CR-019).
        """
        narration_runtime.narrate(
            self, text, *animations, drift=self.ambient_drift if drift is None else drift
        )

    def beat(self, beat_id: str) -> None:
        """Mở một beat của beat sheet (CR-019). Gắn vào lời thoại kế tiếp."""
        narration_runtime.beat(self, beat_id)

    def chapter(self, title: str) -> None:
        """Mở một chapter YouTube (CR-006 FR15). Gắn vào lời thoại kế tiếp."""
        narration_runtime.chapter(self, title)

    def clip(self, name: str):
        """Đánh dấu một đoạn của scene là clip dọc phái sinh (CR-007 FR19.2).

        Dùng như context manager: `with self.clip("tên"): ...`. Hoạt động
        được cả khi bên trong gọi `self.narrate()` — dùng chung `_recorder`
        singleton của `narration.py` nên không phá cơ chế đếm `index`. Không
        cho lồng nhau: `NarrationError` nếu Creator mở một `clip()` khác
        trong lúc clip trước chưa đóng.
        """
        return narration_runtime.clip(self, name)

    # --- Beat dựng sẵn (CR-019 FR53) ------------------------------------------
    #
    # Ba beat này là khuôn hình lặp lại ở MỌI video, nên chúng là method chứ
    # không phải thứ Creator dựng lại mỗi lần. Chúng chỉ khả thi sau CR-018:
    # trước đó lời thoại là comment phải đếm khớp theo thứ tự dòng, nên không
    # thể nằm trong một hàm — đúng lý do CR-006 §Quyết định #2 phải lùi FR17
    # xuống thành snippet Creator tự chép.

    def hook(self, question: str, *animations) -> None:
        """Mở đầu bằng hình: frame đầu đã có thứ chuyển động, không phải thẻ tiêu đề.

        Dựng cảnh mở màn (vật, nhân vật) TRƯỚC khi gọi, rồi truyền animation
        kèm theo để chúng chạy trong lúc câu hỏi được đọc. Muốn thẻ tiêu đề thì
        gọi `hook_card()` (CR-042 FR124).
        """
        self.beat("hook")
        self.restore_view()
        self.narrate(question, *[a for a in animations if not isinstance(a, str)])

    def hook_card(self, question: str, subtitle: str | None = None) -> None:
        """Mở đầu bằng thẻ tiêu đề chứa câu hỏi (hành vi cũ của `hook`).

        Nội dung do Creator truyền vào, KHÔNG tự sinh từ tiêu đề video: tiêu đề
        được soạn ở bước publish, sau khi render, nên tại đây nó chưa tồn tại
        (cùng lý do khiến thumbnail tự động không burn chữ — CR-006 §Quyết định #3).
        """
        self.beat("hook")
        self.restore_view()
        card = TitleCard(question, subtitle, theme=self.theme)
        self.reveal(card)
        self.narrate(question)
        self.dismiss(card)

    def recap(self, points: list[str] | None = None, title: str = "Tóm lại",
              narration: str | None = None, *animations) -> None:
        """Tóm tắt bằng cách quay lại toàn cảnh với nhân vật chính ở trạng thái cuối.

        Không hiện bảng gạch đầu dòng: những gì đang trên màn hình LÀ phần tóm
        tắt, khung chỉ lùi ra toàn cảnh và đẩy vào chậm trong lúc đọc. `points`
        chỉ còn để ghép thành lời thoại khi không truyền `narration`; `title`
        giữ cho tương thích. Muốn bảng thì gọi `recap_card()` (CR-042 FR124.2).
        """
        self.beat("recap")
        self.restore_view()
        text = narration or ". ".join(points or [])
        self.narrate(text, *animations, drift=None if animations else True)

    def recap_card(self, points: list[str], title: str = "Tóm lại", narration: str | None = None) -> None:
        """Màn tóm tắt dạng bảng gạch đầu dòng (hành vi cũ của `recap`)."""
        self.beat("recap")
        self.restore_view()
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
        self.restore_view()
        card = TitleCard(message, subtitle, theme=self.theme)
        self.reveal(card)
        self.narrate(message)
        self.wait(hold_seconds)

    # --- Phát hiện chồng lấn (bug report 2026-09-12) ---------------------------

    def play(self, *args, **kwargs):
        """Bọc `Scene.play` để soi chồng lấn hình ảnh SAU mỗi animation.

        Chỉ chạy ở lượt dry (`narration_runtime.is_dry_run()`): đây là cổng
        kiểm tra trước TTS (CR-020), không phải thứ đáng trả thêm thời gian ở
        lượt render thật — lượt đó chạy đúng lại animation y hệt nên chồng lấn
        (nếu có) đã được báo ở lượt dry rồi.
        """
        result = super().play(*args, **kwargs)
        if narration_runtime.is_dry_run():
            self._check_overlaps()
        return result

    def _check_overlaps(self) -> None:
        """So từng cặp mobject top-level đang hiện, báo cặp chồng > ngưỡng.

        "Top-level" nghĩa là `self.mobjects` — danh sách Manim tự giữ, đúng
        những gì `self.add`/`self.play` đã đưa vào khung ở cấp cao nhất (một
        `VGroup` tính là một mobject, không tách con ra so riêng — chồng lấn
        bên trong một component do design system tự canh, không phải lỗi
        script). Chỉ lọc camera frame (xem `stage_mobjects`); ngoài nó không có
        phần tử nền/trang trí nào được thêm vào `self.mobjects`.

        Best-effort tuyệt đối, giống `narration._describe_layout`: một mobject
        lạ không đọc được bbox không được làm hỏng cả lượt dry.
        """
        mobjects = self.stage_mobjects()
        boxes: list[tuple[object, tuple[float, float, float, float]] | None] = []
        for mobject in mobjects:
            try:
                boxes.append((mobject, _bbox_of(mobject)))
            except Exception:  # noqa: BLE001 — xem docstring
                boxes.append(None)

        for i in range(len(boxes)):
            entry_i = boxes[i]
            if entry_i is None:
                continue
            for j in range(i + 1, len(boxes)):
                entry_j = boxes[j]
                if entry_j is None:
                    continue
                ratio = _overlap_ratio(entry_i[1], entry_j[1])
                if ratio > OVERLAP_AREA_RATIO_THRESHOLD:
                    narration_runtime.record_overlap(
                        self,
                        f"{type(entry_i[0]).__name__} và {type(entry_j[0]).__name__} "
                        f"chồng lấn {ratio:.0%} diện tích vật nhỏ hơn",
                    )

    # --- Chuyển cảnh ----------------------------------------------------------

    def reveal(self, *mobjects: Mobject, speed: str = "normal") -> None:
        run_time = self._run_time(speed)
        self.play(*(reveal_animation(m, run_time) for m in mobjects))

    def dismiss(self, *mobjects: Mobject, speed: str = "fast") -> None:
        run_time = self._run_time(speed)
        self.play(*(dismiss_animation(m, run_time) for m in mobjects))

    def swap(self, old: Mobject, new: Mobject, speed: str = "normal") -> None:
        self.play(swap_animation(old, new, self._run_time(speed)))

    def emphasize(self, *mobjects: Mobject, style: str = "pulse", speed: str = "normal") -> None:
        """Nhấn vào vật đã có trên màn hình. `style="circle"` khoanh thay vì phóng.

        Dùng `circle` khi vật nằm lọt trong một hình lớn (một ô của bảng, một
        khối của sơ đồ): phóng to nó ở đó sẽ đè lên hàng xóm.
        """
        run_time = self._run_time(speed)
        color = self.theme.accent
        self.play(*(emphasize_animation(m, run_time, style, color) for m in mobjects))

    def travel(self, mobject: Mobject, path: VMobject, speed: str = "slow") -> None:
        """Cho một vật chạy dọc `path` (dựng bằng `self.path(...)`).

        Chậm mặc định: quỹ đạo là thứ người xem phải theo mắt được, và một
        chuyển động nhanh dọc đường cong đọc thành một cú nhảy.
        """
        self.play(travel_animation(mobject, path, self._run_time(speed)))

    def clear_stage(self, speed: str = "fast") -> None:
        """Dọn sạch khung. Gọi giữa hai beat để không tích tụ rác thị giác.

        Trả camera về toàn cảnh trước: beat kế tiếp dựng vật quanh tâm khung
        mặc định, nên nếu camera còn đang zoom thì chúng sẽ lệch hoặc ra ngoài.
        """
        self.restore_view(speed=speed)
        mobjects = self.stage_mobjects()
        if mobjects:
            self.dismiss(*mobjects, speed=speed)

    # --- Camera ---------------------------------------------------------------

    def focus(self, *mobjects: Mobject, speed: str = "normal") -> None:
        """Zoom camera vào một vật (hoặc cả nhóm vật), chừa một khoảng thở.

        Độ zoom suy ra từ kích thước vật, không phải con số script tự chọn —
        cùng lý do cỡ chữ và toạ độ tuyệt đối bị cấm.
        """
        target = mobjects[0] if len(mobjects) == 1 else VGroup(*mobjects)
        frame = self.camera.frame
        self._remember_home_view()
        if not self.is_zoomed():
            # Vật có sẵn trước lần zoom này bị camera cắt đi là CHỦ Ý; QC chỉ
            # soi tràn khung với vật xuất hiện sau đó (xem `added_during_zoom`).
            self._pre_zoom_ids = {id(m) for m in self.stage_mobjects()}
        width = max(target.width, target.height * frame.width / frame.height)
        width = min(width * CAMERA_FOCUS_PADDING, self._home_view[1])
        self.play(
            frame.animate.move_to(target.get_center()).set(width=width),
            run_time=self._run_time(speed),
        )

    def restore_view(self, speed: str = "normal") -> None:
        """Đưa camera về toàn cảnh ban đầu. Không làm gì nếu chưa từng zoom."""
        if getattr(self, "_home_view", None) is None:
            return
        center, width = self._home_view
        frame = self.camera.frame
        self._pre_zoom_ids = None
        if frame.width == width and (frame.get_center() == center).all():
            return
        self.play(frame.animate.move_to(center).set(width=width), run_time=self._run_time(speed))

    def is_zoomed(self) -> bool:
        return getattr(self, "_pre_zoom_ids", None) is not None

    def added_during_zoom(self, mobject: Mobject) -> bool:
        """True nếu đang zoom và vật này xuất hiện SAU lần `focus` đầu tiên."""
        pre = getattr(self, "_pre_zoom_ids", None)
        return pre is not None and id(mobject) not in pre

    def stage_mobjects(self) -> list[Mobject]:
        """Các vật đang hiện, trừ camera frame.

        `self.play(frame.animate...)` khiến Manim thêm frame vào `self.mobjects`;
        nó không phải hình trên màn hình, nên dọn cảnh, dò chồng lấn và QC
        đều phải bỏ qua nó.
        """
        frame = self.camera.frame
        return [m for m in self.mobjects if m is not frame]

    def _remember_home_view(self) -> None:
        if getattr(self, "_home_view", None) is None:
            frame = self.camera.frame
            self._home_view = (frame.get_center().copy(), frame.width)

    def pace(self, speed: str = "normal") -> float:
        """Số giây của một tốc độ theme, cho animation script tự `self.play(...)`.

        Đường thoát hiểm cuối cùng: nếu phải gọi một animation thô của Manim thì
        ít nhất nhịp của nó vẫn lấy từ theme. Các ca thường gặp đã có method
        riêng (`travel`, `count`, `emphasize`) nên không cần tới đây."""
        return self._run_time(speed)

    def _run_time(self, speed: str) -> float:
        pacing = self.theme.pacing
        return {"fast": pacing.fast, "normal": pacing.normal, "slow": pacing.slow}.get(
            speed, pacing.normal
        )


def _as_point(value) -> np.ndarray:
    """Toạ độ của một đối số `shape`/`path`: tâm của mobject, hoặc điểm cho sẵn."""
    if isinstance(value, Mobject):
        return value.get_center()
    point = np.asarray(value, dtype=float)
    # Toạ độ Manim luôn 3 chiều; script viết (x, y) là chuyện thường gặp.
    return np.array([point[0], point[1], point[2] if len(point) > 2 else 0.0])


def _unit_vector(start: np.ndarray, end: np.ndarray) -> np.ndarray:
    delta = np.asarray(end, dtype=float) - np.asarray(start, dtype=float)
    norm = float(np.linalg.norm(delta))
    return delta / norm if norm else np.array([1.0, 0.0, 0.0])


def _edge_points(source: Mobject, target: Mobject, buff: float = 0.15):
    """Hai điểm trên mép của hai vật, nhìn về phía nhau.

    `Arrow(source, target)` tự làm việc này; `CurvedArrow` chỉ nhận toạ độ, nên
    nếu đưa thẳng tâm vào thì mũi tên cắm vào giữa vật.
    """
    unit = _unit_vector(source.get_center(), target.get_center())
    return (
        source.get_boundary_point(unit) + unit * buff,
        target.get_boundary_point(-unit) - unit * buff,
    )


def _bbox_of(mobject: Mobject) -> tuple[float, float, float, float]:
    """Hộp bao trục-song-song (left, right, top, bottom) theo toạ độ Manim.

    Cùng bốn accessor `narration._describe_layout` đã dùng cho QC (FR58.1) —
    giữ một nguồn sự thật duy nhất cho "hộp bao của một mobject là gì".
    """
    return (
        float(mobject.get_left()[0]),
        float(mobject.get_right()[0]),
        float(mobject.get_top()[1]),
        float(mobject.get_bottom()[1]),
    )


def _overlap_ratio(
    a: tuple[float, float, float, float], b: tuple[float, float, float, float]
) -> float:
    """Diện tích giao của hai hộp bao, chia cho diện tích hộp NHỎ hơn.

    Chia cho hộp nhỏ hơn (không phải hợp, không phải hộp lớn hơn) vì đó là
    điều một Creator thực sự muốn biết: "vật nhỏ có bị vật kia nuốt phần lớn
    diện tích của nó không" — một nhãn nhỏ nằm sát mép một hình lớn không đáng
    báo, nhưng một nhãn nhỏ NẰM GIỮA một hình lớn (diện tích giao gần bằng cả
    nhãn) chính là kiểu bug này tồn tại để bắt.
    """
    left_a, right_a, top_a, bottom_a = a
    left_b, right_b, top_b, bottom_b = b

    inter_width = min(right_a, right_b) - max(left_a, left_b)
    inter_height = min(top_a, top_b) - max(bottom_a, bottom_b)
    if inter_width <= 0 or inter_height <= 0:
        return 0.0
    inter_area = inter_width * inter_height

    area_a = (right_a - left_a) * (top_a - bottom_a)
    area_b = (right_b - left_b) * (top_b - bottom_b)
    smaller_area = min(area_a, area_b)
    if smaller_area <= 0:
        return 0.0
    return inter_area / smaller_area

