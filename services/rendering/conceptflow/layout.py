"""Hình học của khung an toàn (CR-017 FR45.3).

Cũng **không import manim**, vì lý do giống `theme.py`: đây là số học thuần trên
bounding box, và nó là phần duy nhất của bố cục thực sự cần kiểm thử. Component
chỉ việc đọc bbox của mobject rồi đưa vào đây.

CR-021 FR59.1 (phát hiện tràn khung khi chấm QC) dùng lại đúng các hàm này, nên
luật "thế nào là tràn" chỉ tồn tại ở một nơi — nếu QC và component tự định nghĩa
riêng, sẽ có lúc component dựng ra thứ mà chính QC báo lỗi.
"""

from __future__ import annotations

from dataclasses import dataclass

from .theme import FRAME_HEIGHT, FRAME_WIDTH, SAFE_MARGIN


@dataclass(frozen=True)
class Box:
    """Bounding box theo hệ toạ độ Manim: gốc ở tâm khung, y hướng lên."""

    left: float
    right: float
    bottom: float
    top: float

    @property
    def width(self) -> float:
        return self.right - self.left

    @property
    def height(self) -> float:
        return self.top - self.bottom

    @classmethod
    def from_mobject(cls, mobject: object) -> "Box":
        """Đọc bbox từ một mobject Manim mà không import manim.

        Chỉ dựa vào bốn method mà mọi Mobject đều có.
        """
        return cls(
            left=float(mobject.get_left()[0]),      # type: ignore[attr-defined]
            right=float(mobject.get_right()[0]),    # type: ignore[attr-defined]
            bottom=float(mobject.get_bottom()[1]),  # type: ignore[attr-defined]
            top=float(mobject.get_top()[1]),        # type: ignore[attr-defined]
        )


def safe_area(margin: float = SAFE_MARGIN) -> Box:
    """Vùng mà chữ và hình được phép chiếm."""
    return Box(
        left=-FRAME_WIDTH / 2 + margin,
        right=FRAME_WIDTH / 2 - margin,
        bottom=-FRAME_HEIGHT / 2 + margin,
        top=FRAME_HEIGHT / 2 - margin,
    )


def overflow(box: Box, margin: float = SAFE_MARGIN) -> tuple[str, ...]:
    """Các cạnh mà `box` vượt ra ngoài vùng an toàn.

    Trả về tuple rỗng khi nằm gọn. Trả về tên cạnh chứ không phải True/False vì
    thông báo lỗi của QC cần nói rõ tràn phía nào thì Creator mới sửa được.
    """
    area = safe_area(margin)
    sides = []
    if box.left < area.left:
        sides.append("trái")
    if box.right > area.right:
        sides.append("phải")
    if box.bottom < area.bottom:
        sides.append("dưới")
    if box.top > area.top:
        sides.append("trên")
    return tuple(sides)


#: Co thêm một chút so với mức vừa khít.
#:
#: Không có biên này, một nhóm cao đúng bằng vùng an toàn sẽ được co về đúng
#: mép, rồi sai số dấu phẩy động đẩy nó ra ngoài vài phần nghìn đơn vị — và
#: chính `overflow()` báo tràn thứ mà `fit_scale()` vừa bảo là vừa. Hai hàm này
#: phải nhất quán với nhau, vì CR-021 dùng `overflow()` để chấm QC những khung
#: hình do component ở đây dựng ra.
FIT_EPSILON = 0.995


def fit_scale(box: Box, margin: float = SAFE_MARGIN) -> float:
    """Hệ số cần nhân để `box` vừa vùng an toàn.

    Trả về 1.0 khi đã vừa — component không bao giờ phóng to thứ vốn đã lọt,
    vì làm vậy sẽ khiến cùng một đoạn chữ hiện ở cỡ khác nhau tuỳ độ dài.
    """
    area = safe_area(margin)
    if box.width <= 0 or box.height <= 0:
        return 1.0
    factor = min(area.width / box.width, area.height / box.height) * FIT_EPSILON
    return min(1.0, factor)


def overlaps(a: Box, b: Box, tolerance: float = 0.0) -> bool:
    """Hai box có giao nhau không (CR-021 FR59.2).

    `tolerance` nới lỏng phép so: bbox của Manim thường rộng hơn phần mực thật
    (đuôi chữ, khoảng đệm của font), nên hai dòng chữ sát nhau có thể "giao" về
    mặt số học mà mắt nhìn vẫn tách bạch.
    """
    return not (
        a.right - tolerance <= b.left
        or b.right - tolerance <= a.left
        or a.top - tolerance <= b.bottom
        or b.top - tolerance <= a.bottom
    )


def _channel(hex_color: str, index: int) -> float:
    value = int(hex_color.lstrip("#")[index * 2 : index * 2 + 2], 16) / 255.0
    return value / 12.92 if value <= 0.04045 else ((value + 0.055) / 1.055) ** 2.4


def relative_luminance(hex_color: str) -> float:
    """Độ sáng tương đối theo WCAG."""
    r, g, b = (_channel(hex_color, i) for i in range(3))
    return 0.2126 * r + 0.7152 * g + 0.0722 * b


def contrast_ratio(foreground: str, background: str) -> float:
    """Tỉ lệ tương phản WCAG, từ 1.0 (trùng màu) tới 21.0 (đen trên trắng).

    Dùng cho CR-021 FR59.4. Ngưỡng 4.5 là mức AA của WCAG cho chữ thường; chữ
    trên video nên cao hơn vì còn bị nén và bị xem trên màn hình kém.
    """
    light = relative_luminance(foreground)
    dark = relative_luminance(background)
    if light < dark:
        light, dark = dark, light
    return (light + 0.05) / (dark + 0.05)
