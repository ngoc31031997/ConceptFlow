"""Bộ kiểm tra style cho hình của thư viện minh hoạ.

Luật đầy đủ nằm ở authoring-service/.../prompts/illustration_style_vi.txt;
mã [S..] ở đây là mã luật trong file đó. Vi phạm làm hỏng style của cả kênh
(gradient, filter, ảnh, chữ, ngẫu nhiên theo đồng hồ) là LỖI và chặn lưu; phần
còn lại là CẢNH BÁO — Creator quyết định. Màu ngoài bảng
màu kênh (S9) không bị kiểm tra.

Gradient (`linearGradient`, `radialGradient`) và làm mờ (`feGaussianBlur` trong
`<filter>`) được dùng cho bóng mềm, điểm sáng và quầng sáng; mọi bộ lọc `fe*`
khác vẫn là lỗi. Id của gradient/filter phải sinh bằng `useSvgId()` (S25): một
id viết cứng bị trùng khi cùng một hình xuất hiện hai lần trên một khung, và
hình thứ hai tô theo gradient của hình thứ nhất.

Kiểm tra trên mã nguồn TSX, không trên ảnh: rẻ, tất định, và chỉ được đúng dòng.
"""

from __future__ import annotations

import re
from dataclasses import dataclass

ERROR_TAGS = {"pattern": "S1", "image": "S1", "text": "S1", "foreignObject": "S1"}
# The one filter primitive a drawing may use: a soft blur for a glow.
ALLOWED_FILTER_PRIMITIVES = {"feGaussianBlur"}
GRADIENT_TAGS = {"linearGradient", "radialGradient"}
ID_TAGS = GRADIENT_TAGS | {"filter", "clipPath", "mask"}
MAX_GRADIENTS = 4
FORBIDDEN_CALLS = {"Math.random(": "S22", "Date.now(": "S22", "setTimeout(": "S22", "setInterval(": "S22"}

TAG_RE = re.compile(r"<([A-Za-z][\w.]*)\b([^<>]*?)/?>", re.S)
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


BACKDROP = "backdrop"


def check_style(code: str, kind: str = "figure") -> tuple[list[StyleFinding], list[StyleFinding]]:
    """(lỗi, cảnh báo) của một hình (`kind` "figure") hay một nền ("backdrop").

    Nền theo luật B..: vẽ kín khung theo `useLayerBox()`/`useVideoConfig()`
    thay vì bọc trong Figure, nhận prop `layer`, không có mặt người."""
    errors: list[StyleFinding] = []
    warnings: list[StyleFinding] = []

    shapes = 0
    gradients = 0
    for m in TAG_RE.finditer(code):
        tag, attrs = m.group(1), m.group(2)
        line = _line(code, m.start())
        if tag in ERROR_TAGS:
            msg = f"không dùng <{tag}> — hình không có hoạ tiết, chữ hay ảnh"
            errors.append(StyleFinding(ERROR_TAGS[tag], msg, line))
            continue
        if tag.startswith("fe") and tag not in ALLOWED_FILTER_PRIMITIVES:
            msg = f"không dùng <{tag}> — bộ lọc duy nhất được dùng là feGaussianBlur (quầng sáng)"
            errors.append(StyleFinding("S1", msg, line))
            continue
        if tag in GRADIENT_TAGS:
            gradients += 1
        if tag in ID_TAGS and re.search(r"\bid\s*=\s*['\"]", attrs):
            msg = f"<{tag}> có id viết cứng — dùng id={{useSvgId('...')}} để hai bản của hình không trùng id"
            warnings.append(StyleFinding("S25", msg, line))
        if tag not in SHAPES:
            continue
        shapes += 1
        fill, stroke = _attr(attrs, "fill"), _attr(attrs, "stroke")
        if stroke and fill and fill != "none":
            warnings.append(StyleFinding("S2", f"<{tag}> vừa tô màu vừa có viền — khối không có viền", line))
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

    if kind == BACKDROP:
        errors += _backdrop_errors(code)
    elif "<Figure" not in code:
        errors.append(StyleFinding("S16",
            "hình phải bọc trong <Figure {...fig} size vw vh> để đặt theo x, y, size"))

    if gradients > MAX_GRADIENTS:
        warnings.append(StyleFinding(
            "S1", f"{gradients} gradient — tối đa {MAX_GRADIENTS}, phần còn lại tô màu phẳng"))
    if shapes > 60:
        warnings.append(StyleFinding("S4", f"{shapes} hình con — quá chi tiết (nên 5–25)"))

    # Bảng màu kênh (S9) và số màu gốc (S10) chỉ là gợi ý cho AI vẽ, không kiểm tra:
    # hình minh hoạ được dùng màu của chính vật.
    black_fill = re.search(r"fill\s*=\s*['\"{]*['\"]#0{6}['\"]", code)
    if black_fill and not re.search(r"#000000['\"][^>]*opacity", code):
        warnings.append(StyleFinding("S11", "đen thuần làm mảng màu — dùng INK hoặc bóng opacity thấp"))

    if kind != BACKDROP and "color" not in re.split(r"\)\s*\{", code, maxsplit=1)[0]:
        warnings.append(StyleFinding("S12", "màu chính không đổi được qua prop color"))
    if "<Face" in code and "useCurrentFrame" not in code:
        warnings.append(StyleFinding("S19", "hình có mặt nhưng không có chuyển động tự thân nào"))
    if "useCurrentFrame" in code and "still" not in code:
        warnings.append(StyleFinding("S20", "chuyển động không tắt được bằng prop still"))
    return errors, warnings


def _backdrop_errors(code: str) -> list[StyleFinding]:
    """Luật chặn lưu riêng của một nền."""
    errors: list[StyleFinding] = []
    head = re.split(r"\)\s*\{", code, maxsplit=1)[0]
    if not re.search(r"\blayer\b", head):
        errors.append(StyleFinding(
            "B1", "nền phải nhận prop layer ('sky' | 'far' | 'mid' | 'near') và vẽ đúng lớp đó"))
    if "useLayerBox" not in code and "useVideoConfig" not in code:
        errors.append(StyleFinding(
            "B2",
            "nền phải vẽ kín khung theo useLayerBox() hoặc useVideoConfig(), không theo kích thước cố định"))
    idx = code.find("<Face")
    if idx >= 0:
        errors.append(StyleFinding("B5", "nền là nơi chốn, không có mặt người", _line(code, idx)))
    return errors
