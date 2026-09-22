"""Từ vựng chuyển cảnh chuẩn (CR-017 FR45.4).

Bốn động tác, run_time lấy từ `Theme.pacing`. Script không tự đặt `run_time`,
nên hai video khác nhau có cùng một nhịp — đó là thứ người xem cảm nhận được
là "cùng một kênh" trước cả khi họ nhận ra bảng màu.

Tách khỏi `scene.py` để `ConceptFlowScene` chỉ còn là nơi ráp các mảnh, và để
mỗi động tác kiểm thử được riêng.
"""

from __future__ import annotations

from manim import (
    DOWN,
    UP,
    Circumscribe,
    Create,
    FadeIn,
    FadeOut,
    Indicate,
    Mobject,
    MoveAlongPath,
    ReplacementTransform,
    VMobject,
)


def reveal_animation(mobject: Mobject, run_time: float):
    """Đưa một đối tượng vào khung.

    `Create` cho hình vẽ được bằng nét (đường, khối hình), `FadeIn` cho phần
    còn lại (chữ, ảnh). Phân biệt như vậy vì `Create` trên một khối chữ dài
    trông như máy đánh chữ hỏng, còn `FadeIn` trên một đường thẳng thì phí mất
    động tác vẽ vốn là thế mạnh của Manim.
    """
    if isinstance(mobject, VMobject) and not _is_texty(mobject):
        return Create(mobject, run_time=run_time)
    return FadeIn(mobject, shift=UP * 0.2, run_time=run_time)


def dismiss_animation(mobject: Mobject, run_time: float):
    return FadeOut(mobject, shift=DOWN * 0.15, run_time=run_time)


def swap_animation(old: Mobject, new: Mobject, run_time: float):
    """Thay chỗ: giữ mạch nhìn thay vì tắt rồi bật."""
    return ReplacementTransform(old, new, run_time=run_time)


def emphasize_animation(
    mobject: Mobject, run_time: float, style: str = "pulse", color: str | None = None
):
    """Nhấn vào một vật đã có trên màn hình.

    `pulse` phóng nhẹ — hợp với một vật nhỏ đứng riêng. `circle` khoanh một
    khung chạy quanh vật rồi biến mất — hợp khi vật nằm trong một hình lớn và
    phóng to nó sẽ đè lên hàng xóm. Màu do scene truyền vào (từ theme), vì
    module này cố tình không biết gì về bảng màu.
    """
    if style == "circle":
        kwargs = {"color": color} if color else {}
        return Circumscribe(mobject, buff=0.15, run_time=run_time, **kwargs)
    return Indicate(mobject, scale_factor=1.12, run_time=run_time)


def travel_animation(mobject: Mobject, path: VMobject, run_time: float):
    """Đưa một vật chạy dọc theo một đường.

    Tách khỏi `reveal`: ở đây vật đã có sẵn trên màn hình và cái người xem theo
    dõi là **quỹ đạo**, không phải sự xuất hiện.
    """
    return MoveAlongPath(mobject, path, run_time=run_time)


def _is_texty(mobject: Mobject) -> bool:
    """Đoán xem đối tượng có phải chữ không, không cần import Text/MathTex.

    Dựa vào tên class thay vì isinstance để module này không phải kéo theo cả
    họ text mobject của Manim (Text, MarkupText, Tex, MathTex, Code...), vốn
    chỉ dùng cho đúng một phép so ở đây.
    """
    names = {type(mobject).__name__, *(c.__name__ for c in type(mobject).__mro__)}
    # `Readout` là một con số: vẽ dần nét của các chữ số trông như lỗi font, và
    # bản sống của nó dựng lại mỗi frame nên `Create` không có gì để vẽ tiếp.
    return bool(
        names
        & {"Text", "MarkupText", "Tex", "MathTex", "SingleStringMathTex", "Code", "Readout"}
    )
