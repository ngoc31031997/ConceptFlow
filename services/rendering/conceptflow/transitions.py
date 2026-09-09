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
    Create,
    FadeIn,
    FadeOut,
    Indicate,
    Mobject,
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


def emphasize_animation(mobject: Mobject, run_time: float):
    return Indicate(mobject, scale_factor=1.12, run_time=run_time)


def _is_texty(mobject: Mobject) -> bool:
    """Đoán xem đối tượng có phải chữ không, không cần import Text/MathTex.

    Dựa vào tên class thay vì isinstance để module này không phải kéo theo cả
    họ text mobject của Manim (Text, MarkupText, Tex, MathTex, Code...), vốn
    chỉ dùng cho đúng một phép so ở đây.
    """
    names = {type(mobject).__name__, *(c.__name__ for c in type(mobject).__mro__)}
    return bool(names & {"Text", "MarkupText", "Tex", "MathTex", "SingleStringMathTex", "Code"})
