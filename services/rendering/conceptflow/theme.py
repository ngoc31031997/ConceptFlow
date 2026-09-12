"""Bản sắc thị giác của kênh — nguồn sự thật duy nhất (CR-017 FR44).

Đổi diện mạo của toàn bộ kênh là sửa file này, không sửa script nào.

Module này **không import manim**. Đó là chủ ý: mọi giá trị ở đây là số và
chuỗi thuần, nên bảng màu và thang cỡ chữ kiểm thử được mà không cần cài Manim —
giữ đúng tính chất mà test suite của Rendering đang có (README: test dùng
fake/mock cho toàn bộ tương tác Manim thật).

## Vì sao bảng màu lai

Nền lấy từ bộ nhận diện đã có (`docs/brand/make-banner.py`, cũng là nền của
logo), còn màu nhấn lấy từ Catppuccin Mocha. Creator chọn giữ màu brand cho
intro/outro (CR-023) nhưng dùng Catppuccin cho nội dung; hai nền khác nhau sẽ
giật màu ở đúng điểm nối intro→thân, nên nền dùng chung và chỉ màu nhấn khác
nhau.
"""

from __future__ import annotations

from dataclasses import dataclass, field, replace

# --- Bộ nhận diện (docs/brand) ------------------------------------------------
BRAND_BG = "#080E1C"
BRAND_INK = "#F2F7FF"
BRAND_ACCENT = "#8FC7EF"

# --- Catppuccin Mocha: chỉ lấy phần màu nhấn ---------------------------------
MOCHA_BLUE = "#89B4FA"
MOCHA_LAVENDER = "#B4BEFE"
MOCHA_SKY = "#89DCEB"
MOCHA_TEAL = "#94E2D5"
MOCHA_GREEN = "#A6E3A1"
MOCHA_YELLOW = "#F9E2AF"
MOCHA_PEACH = "#FAB387"
MOCHA_RED = "#F38BA8"
MOCHA_MAUVE = "#CBA6F7"
MOCHA_SUBTEXT = "#A6ADC8"
MOCHA_SURFACE = "#313244"

# --- Khung hình ---------------------------------------------------------------
# Manim 0.18 mặc định: frame cao 8 đơn vị, tỉ lệ 16:9.
FRAME_HEIGHT = 8.0
FRAME_WIDTH = 14.222222222222221

#: Chữ và hình phải nằm trong khung này. Rộng hơn mép an toàn của TV vì video
#: còn bị YouTube crop nhẹ trên một số bề mặt, và vì phụ đề burn-in (CR-001)
#: chiếm phần đáy.
SAFE_MARGIN = 0.6


@dataclass(frozen=True)
class FontScale:
    """Thang cỡ chữ. Chỉ bốn bậc — thêm bậc là mở đường cho style trôi.

    Cỡ tính theo đơn vị `font_size` của Manim ở khung 1080p.
    """

    h1: int = 48
    h2: int = 36
    body: int = 28
    caption: int = 20

    def all_sizes(self) -> tuple[int, ...]:
        return (self.h1, self.h2, self.body, self.caption)


@dataclass(frozen=True)
class Pacing:
    """Nhịp phim. Mọi chuyển cảnh lấy run_time từ đây, không tự đặt số.

    Ba bậc là đủ để có nhịp mà không đủ để mỗi video một kiểu.
    """

    fast: float = 0.5
    normal: float = 0.8
    slow: float = 1.4


@dataclass(frozen=True)
class Fonts:
    """Cormorant Garamond chỉ dùng cho H1/H2.

    Serif tương phản cao mất nét thanh khi chữ nhỏ bị nén qua encode, nên mọi
    thứ từ body trở xuống dùng Be Vietnam Pro — font thiết kế riêng cho tiếng
    Việt, dấu đặt chuẩn ở mọi weight.
    """

    display: str = "Cormorant Garamond"
    body: str = "Be Vietnam Pro"
    # "NL" = No Ligatures. Cùng gia đình font, cùng thiết kế, chỉ khác là
    # JetBrains Mono thường tự nối `++`, `<=`, `==`, `!=`, `->`... thành một
    # glyph duy nhất (programming ligatures qua OpenType "calt", không phải
    # "liga"/"dlig") — Manim's `disable_ligatures=True` chỉ tắt được liga/dlig,
    # không tắt calt, nên `CodePanel` (Code mobject cần tô màu từng ký tự) crash
    # với IndexError bất kỳ lúc nào code mẫu có một trong các chuỗi trên (tức
    # là gần như mọi vòng lặp for/so sánh). Bản NL không có bảng ligature nào
    # cả nên không bao giờ trôi khỏi giả định 1-ký-tự-1-glyph của Manim.
    mono: str = "JetBrains Mono NL"


@dataclass(frozen=True)
class Theme:
    """Một bản sắc hoàn chỉnh.

    Theme mới tạo bằng `derive()` (kế thừa rồi ghi đè), không chép lại toàn bộ —
    CR-017 FR44.3 yêu cầu kiến trúc cho nhiều theme ngay cả khi hiện chỉ dùng một.
    """

    name: str

    background: str = BRAND_BG
    ink: str = BRAND_INK
    muted: str = MOCHA_SUBTEXT
    surface: str = MOCHA_SURFACE

    accent: str = BRAND_ACCENT
    accent_alt: str = MOCHA_LAVENDER
    success: str = MOCHA_GREEN
    warning: str = MOCHA_YELLOW
    danger: str = MOCHA_RED

    #: Màu dùng khi cần phân biệt nhiều phần tử cùng loại (các nhánh, các cột).
    #: Thứ tự cố định để hai video nói về cùng chủ đề tô màu giống nhau.
    series: tuple[str, ...] = (
        MOCHA_BLUE,
        MOCHA_PEACH,
        MOCHA_TEAL,
        MOCHA_MAUVE,
        MOCHA_YELLOW,
        MOCHA_GREEN,
    )

    fonts: Fonts = field(default_factory=Fonts)
    scale: FontScale = field(default_factory=FontScale)
    pacing: Pacing = field(default_factory=Pacing)
    safe_margin: float = SAFE_MARGIN

    def derive(self, name: str, **overrides: object) -> "Theme":
        """Theme con: giữ mọi thứ, chỉ đổi cái được nêu."""
        return replace(self, name=name, **overrides)  # type: ignore[arg-type]

    def series_color(self, index: int) -> str:
        """Màu thứ `index` trong dải, quay vòng khi hết."""
        return self.series[index % len(self.series)]


DEFAULT = Theme(name="conceptflow")

_THEMES: dict[str, Theme] = {DEFAULT.name: DEFAULT}


def register(theme: Theme) -> None:
    """Thêm một theme vào danh mục chọn được theo tên."""
    _THEMES[theme.name] = theme


def get(name: str | None) -> Theme:
    """Theme theo tên; tên lạ hoặc None thì rơi về mặc định.

    Không raise: một video render bằng theme mặc định vẫn là video xem được,
    còn fail cả lượt render vì sai tên theme thì không.
    """
    if not name:
        return DEFAULT
    return _THEMES.get(name, DEFAULT)


def available() -> tuple[str, ...]:
    return tuple(sorted(_THEMES))
