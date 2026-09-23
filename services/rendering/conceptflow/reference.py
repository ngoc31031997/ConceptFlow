"""Khối "theme có gì" nhét vào prompt Manim Engineer — sinh từ `Theme`, không viết tay.

Prompt từng liệt kê theme bằng tay ("`self.theme.accent`, `ink`, `muted`…"), nên
mỗi lần theme thêm hay đổi một thứ là prompt lỗi thời trong im lặng, và model
tự bịa phần còn thiếu (`CENTER`, `self.theme.blue`…). Module này đọc thẳng
`Theme`/`api` để dựng khối văn bản; `tools/gen_theme_reference.py` ghi nó ra file
mà Orchestrator embed, và `tests/conceptflow/test_reference.py` khoá file khớp
với code — sửa theme mà quên sinh lại thì test đỏ.

Không import manim (cùng lý do với `theme.py` và `api.py`).
"""

from __future__ import annotations

from dataclasses import fields, is_dataclass

from .api import REEXPORTED_MANIM_NAMES
from .theme import DEFAULT, TONES, Theme

#: Vai trò của từng màu — (vi, en). Test buộc mọi trường màu của `Theme` phải có
#: mặt ở đây, nên thêm màu mà không mô tả cho model là không qua được CI.
COLOR_ROLES: dict[str, tuple[str, str]] = {
    "background": ("nền video (đã được vẽ sẵn, KHÔNG tự tô)", "video background (already painted, do NOT paint it)"),
    "ink": ("chữ chính, nét chính", "primary text and strokes"),
    "muted": ("chữ phụ, đường trục, thứ ít quan trọng", "secondary text, axes, low-priority elements"),
    "surface": ("nền của khung/hộp đặc", "fill of filled boxes"),
    "accent": ("màu nhấn chính của kênh — thứ cần người xem nhìn vào", "the channel's main highlight — what the viewer should look at"),
    "accent_alt": ("màu nhấn phụ, đi cặp với accent", "secondary highlight paired with accent"),
    "success": ("đúng, đạt, kết quả tốt", "correct, passing, good result"),
    "warning": ("cẩn thận, chú ý, trường hợp biên", "caution, edge case"),
    "danger": ("sai, lỗi, kết quả xấu", "wrong, error, bad result"),
}

#: Mô tả các bậc của từng nhóm — (vi, en).
GROUP_ROLES: dict[str, tuple[str, str]] = {
    "spacing": ("khoảng cách, dùng cho `buff=`", "gaps, for `buff=`"),
    "strokes": ("độ dày nét, dùng cho `stroke_width=`", "line thickness, for `stroke_width=`"),
    "shapes": ("hình dạng chung của khung/hộp", "shared shape of frames/boxes"),
    "pacing": ("thời lượng chuyển cảnh, dùng qua `self.pace(\"fast\"|\"normal\"|\"slow\")` hoặc `speed=`", "transition duration, via `self.pace(\"fast\"|\"normal\"|\"slow\")` or `speed=`"),
    "scale": ("cỡ chữ; chọn bằng method self.title/heading/body/caption, không đặt tay", "font sizes; pick via self.title/heading/body/caption, never set by hand"),
}

_GROUPS = ("spacing", "strokes", "shapes", "pacing", "scale")


def color_fields(theme: Theme = DEFAULT) -> list[str]:
    """Trường màu (chuỗi hex) của theme, theo thứ tự khai báo."""
    return [
        f.name for f in fields(theme)
        if isinstance(getattr(theme, f.name), str)
        and getattr(theme, f.name).startswith("#")
    ]


def group_fields(theme: Theme = DEFAULT) -> list[str]:
    """Trường là một nhóm số (dataclass con), theo thứ tự khai báo."""
    return [
        f.name for f in fields(theme)
        if is_dataclass(getattr(theme, f.name)) and f.name not in ("fonts",)
    ]


def theme_attribute_paths(theme: Theme = DEFAULT) -> dict[str, frozenset[str] | None]:
    """Mọi thứ `self.theme.<x>` hợp lệ. Giá trị là tập con của `<x>` nếu `<x>` là nhóm."""
    paths: dict[str, frozenset[str] | None] = {}
    for f in fields(theme):
        value = getattr(theme, f.name)
        paths[f.name] = (
            frozenset(g.name for g in fields(value)) if is_dataclass(value) else None
        )
    # Method công khai của Theme (series_color, derive) — gọi được, không có con.
    for name in ("series_color", "derive"):
        paths[name] = None
    return paths


def render_theme_reference(language: str, theme: Theme = DEFAULT) -> str:
    vi = language == "vi"
    idx = 0 if vi else 1
    out: list[str] = []
    add = out.append

    add("### Theme của kênh — CHỈ được dùng đúng những gì liệt kê dưới đây" if vi
        else "### The channel theme — use ONLY what is listed below")
    add("")
    add("Truy cập qua `self.theme.<tên>`. KHÔNG có tên nào ngoài danh sách; đừng đoán "
        "(`self.theme.blue`, `self.theme.primary`… đều không tồn tại)." if vi
        else "Access via `self.theme.<name>`. There is NO name outside this list; do not guess "
        "(`self.theme.blue`, `self.theme.primary`… do not exist).")
    add("")
    add("Màu (giá trị chỉ để bạn hình dung, KHÔNG viết mã hex vào code):" if vi
        else "Colors (values shown only so you can picture them — NEVER write hex into code):")
    for name in color_fields(theme):
        add(f"- `self.theme.{name}` {getattr(theme, name)} — {COLOR_ROLES[name][idx]}")
    series = ", ".join(theme.series)
    add(f"- `self.theme.series_color(i)` — màu thứ i trong dải {len(theme.series)} màu ({series}), "
        "quay vòng; dùng để phân biệt các nhánh/cột cùng loại" if vi
        else f"- `self.theme.series_color(i)` — i-th of a {len(theme.series)}-color series ({series}), "
        "cycling; use to tell same-kind branches/columns apart")
    add("")
    add(f"Sắc thái `tone=` (cho shape/connect/outline/readout): {', '.join(f'`{t}`' for t in TONES)}. "
        "Đây là tên màu ở trên; truyền tên, không truyền màu." if vi
        else f"`tone=` values (for shape/connect/outline/readout): {', '.join(f'`{t}`' for t in TONES)}. "
        "They are the color names above; pass the name, not a color.")
    add("")
    add("Số đo (dùng nếu cần đặt tay; mặc định các method của scene đã tự dùng):" if vi
        else "Measures (use when placing by hand; scene methods already use them by default):")
    for group in _GROUPS:
        value = getattr(theme, group)
        items = ", ".join(f"`{f.name}` = {getattr(value, f.name)}" for f in fields(value))
        add(f"- `self.theme.{group}.<…>` — {GROUP_ROLES[group][idx]}: {items}")
    add("")
    consts = ", ".join(f"`{n}`" for n in sorted(REEXPORTED_MANIM_NAMES))
    add(f"Hằng số có sẵn sau `from conceptflow import *`: {consts}. KHÔNG có `CENTER` — muốn căn giữa dùng "
        "`ORIGIN` hoặc `.move_to(ORIGIN)`. Cần hằng số/class khác của Manim thì `from manim import X` "
        "đích danh." if vi
        else f"Constants available after `from conceptflow import *`: {consts}. There is NO `CENTER` — to "
        "center use `ORIGIN` or `.move_to(ORIGIN)`. Need any other Manim constant/class? "
        "`from manim import X` by name.")
    return "\n".join(out) + "\n"
