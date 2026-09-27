"""Bộ kiểm tra style cho hình của thư viện minh hoạ (CR-044).

Luật đầy đủ nằm ở authoring-service/.../prompts/illustration_style_vi.txt;
mã [S..] ở đây là mã luật trong file đó. Vi phạm làm hỏng style của cả kênh
(gradient, filter, ảnh, chữ, ngẫu nhiên theo đồng hồ) là LỖI và chặn lưu; phần
còn lại là CẢNH BÁO — Creator quyết định (đã chốt 2026-09-27). Màu ngoài bảng
màu kênh (S9) không bị kiểm tra (CR-045).

Kiểm tra trên mã nguồn TSX, không trên ảnh: rẻ, tất định, và chỉ được đúng dòng.
"""

from __future__ import annotations

import re
from dataclasses import dataclass

ERROR_TAGS = {
    "linearGradient": "S1", "radialGradient": "S1", "pattern": "S1", "filter": "S1",
    "feGaussianBlur": "S1", "feDropShadow": "S1", "image": "S1", "text": "S1", "foreignObject": "S1",
}
FORBIDDEN_CALLS = {"Math.random(": "S22", "Date.now(": "S22", "setTimeout(": "S22", "setInterval(": "S22"}

TAG_RE = re.compile(r"<([A-Za-z][\w.]*)\b([^<>]*?)/?>", re.S)
HEX_RE = re.compile(r"['\"]#([0-9A-Fa-f]{6}|[0-9A-Fa-f]{3})['\"]")
SHAPES = {"rect", "circle", "ellipse", "path", "polygon"}


@dataclass(frozen=True)
class StyleFinding:
    rule: str
    message: str
    line: int | None = None

    def text(self) -> str:
        return f"[{self.rule}] {self.message}"


def _line(code: str, index: int) -> int:
    return code.count("\n", 0, index) + 1


def _attr(attrs: str, name: str) -> str | None:
    m = re.search(rf"\b{name}\s*=\s*(\{{[^}}]*\}}|\"[^\"]*\"|'[^']*')", attrs)
    return m.group(1).strip("{}\"'").strip() if m else None


def _expand(hex6: str) -> str:
    h = hex6.upper()
    return "".join(c * 2 for c in h) if len(h) == 3 else h


def check_style(code: str) -> tuple[list[StyleFinding], list[StyleFinding]]:
    """(lỗi, cảnh báo) của một hình."""
    errors: list[StyleFinding] = []
    warnings: list[StyleFinding] = []

    shapes = 0
    for m in TAG_RE.finditer(code):
        tag, attrs = m.group(1), m.group(2)
        line = _line(code, m.start())
        if tag in ERROR_TAGS:
            msg = f"không dùng <{tag}> — hình phải phẳng, không chữ, không ảnh"
            errors.append(StyleFinding(ERROR_TAGS[tag], msg, line))
            continue
        if tag not in SHAPES:
            continue
        shapes += 1
        fill, stroke = _attr(attrs, "fill"), _attr(attrs, "stroke")
        if stroke and fill and fill != "none":
            warnings.append(StyleFinding("S2", f"<{tag}> vừa tô màu vừa có viền — khối không có viền", line))
        if tag == "rect" and _attr(attrs, "rx") is None:
            w, h = _attr(attrs, "width"), _attr(attrs, "height")
            if not (w and h and w.isdigit() and h.isdigit() and min(int(w), int(h)) <= 6):
                warnings.append(StyleFinding("S3", "<rect> không bo góc (thêm rx)", line))
        if stroke and _attr(attrs, "strokeLinecap") is None and tag == "path":
            warnings.append(StyleFinding("S3", "nét thiếu strokeLinecap=\"round\"", line))

    for call, rule in FORBIDDEN_CALLS.items():
        idx = code.find(call)
        if idx >= 0:
            msg = f"không dùng {call.rstrip('(')} — chuyển động phải theo frame"
            errors.append(StyleFinding(rule, msg, _line(code, idx)))
    for bad in ("@keyframes", "animation:", "transition:"):
        idx = code.find(bad)
        if idx >= 0:
            msg = f"không dùng CSS '{bad.rstrip(':')}' — chuyển động phải theo frame"
            errors.append(StyleFinding("S22", msg, _line(code, idx)))

    if "<Figure" not in code:
        errors.append(StyleFinding("S16", 
            "hình phải bọc trong <Figure {...fig} size vw vh> để đặt theo x, y, size"))

    if shapes > 60:
        warnings.append(StyleFinding("S4", f"{shapes} hình con — quá chi tiết (nên 5–25)"))

    colours: dict[str, int] = {}
    for m in HEX_RE.finditer(code):
        colours.setdefault(_expand(m.group(1)), _line(code, m.start()))
    # S9 (bảng màu kênh) không còn là cảnh báo (CR-045): hình minh hoạ được
    # dùng màu của chính vật, bảng màu chỉ là gợi ý cho AI vẽ.
    base = {c for c in colours if c not in {"FFFFFF", "000000", "3A1F4B"}}
    if len(base) > 6:
        warnings.append(StyleFinding(
            "S10", f"{len(base)} màu gốc — tối đa 6 (dùng shadeOf cho màu tối/sáng)"))
    black_fill = re.search(r"fill\s*=\s*['\"{]*['\"]#0{6}['\"]", code)
    if black_fill and not re.search(r"#000000['\"][^>]*opacity", code):
        warnings.append(StyleFinding("S11", "đen thuần làm mảng màu — dùng INK hoặc bóng opacity thấp"))

    if "color" not in re.split(r"\)\s*\{", code, maxsplit=1)[0]:
        warnings.append(StyleFinding("S12", "màu chính không đổi được qua prop color"))
    if "<Face" in code and "useCurrentFrame" not in code:
        warnings.append(StyleFinding("S19", "hình có mặt nhưng không có chuyển động tự thân nào"))
    if "useCurrentFrame" in code and "still" not in code:
        warnings.append(StyleFinding("S20", "chuyển động không tắt được bằng prop still"))
    return errors, warnings
