"""Bộ kiểm tra style cho hình của thư viện minh hoạ.

Luật đầy đủ là văn bản luật vẽ hình của ngôn ngữ hình đang ghim (authoring-service);
mã [S..], [B..], [R..], [M..], [T..] ở đây là mã luật trong văn bản đó. Vi phạm làm
hỏng lối vẽ chung (chữ, ảnh, bộ lọc tự viết, ngẫu nhiên theo đồng hồ, thiếu xương
của rig) là LỖI và chặn lưu; phần còn lại là CẢNH BÁO — Creator quyết định. Màu
không bị kiểm tra: màu đến từ bảng màu của video.

Bộ lọc `fe*` và `<filter>` do kit lo (vật liệu, bóng, hạt, quầng sáng), hình không
tự viết. Số hình con và số gradient tự viết giới hạn theo HẠNG của hình trong video
này: chính (`hero`), phụ (`support`), phông (`background`).

Kiểm tra trên mã nguồn TSX, không trên ảnh: rẻ, tất định, và chỉ được đúng dòng.
"""

from __future__ import annotations

import re
from dataclasses import dataclass

ERROR_TAGS = {"pattern": "S1", "image": "S1", "text": "S1", "foreignObject": "S1"}
GRADIENT_TAGS = {"linearGradient", "radialGradient"}
ID_TAGS = GRADIENT_TAGS | {"clipPath", "mask"}
FORBIDDEN_CALLS = {"Math.random(": "S22", "Date.now(": "S22", "setTimeout(": "S22", "setInterval(": "S22"}

TAG_RE = re.compile(r"<([A-Za-z][\w.]*)\b([^<>]*?)/?>", re.S)
RAW_SHAPES = {"rect", "circle", "ellipse", "path", "polygon"}

HERO, SUPPORT, BACKGROUND = "hero", "support", "background"
TIERS = (HERO, SUPPORT, BACKGROUND)
#: Hạng -> (số hình con tối thiểu, tối đa) trước khi cảnh báo S4.
SHAPE_RANGE = {HERO: (80, 400), SUPPORT: (20, 80), BACKGROUND: (3, 15)}
#: Hạng -> số gradient tự viết tối đa trước khi báo lỗi.
MAX_GRADIENTS = {HERO: 8, SUPPORT: 4, BACKGROUND: 0}
#: Từ số khối thô này trở lên mà không có <Shape> nào thì cảnh báo M1.
RAW_SHAPES_WITHOUT_MATERIAL = 3


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


def check_style(
    code: str, kind: str = "figure", tier: str = SUPPORT, skeletons: dict[str, dict[str, str | None]] | None = None,
) -> tuple[list[StyleFinding], list[StyleFinding]]:
    """(lỗi, cảnh báo) của một hình (`kind` "figure") hay một nền ("backdrop").

    `tier` là hạng của hình trong video này. `skeletons` là bảng bộ xương của kit
    (domain.kits.load_skeletons); có thì kiểm rig thiếu xương (R1).

    Nền theo luật B..: vẽ kín khung theo `useLayerBox()`/`useVideoConfig()`
    thay vì bọc trong Figure, nhận prop `layer`, không có mặt người."""
    if tier not in TIERS:
        raise ValueError(f"hạng {tier!r} không hợp lệ ({', '.join(TIERS)})")
    errors: list[StyleFinding] = []
    warnings: list[StyleFinding] = []

    raw_shapes = 0
    gradients = 0
    for m in TAG_RE.finditer(code):
        tag, attrs = m.group(1), m.group(2)
        line = _line(code, m.start())
        if tag in ERROR_TAGS:
            msg = f"không dùng <{tag}> — hình không có hoạ tiết, chữ hay ảnh"
            errors.append(StyleFinding(ERROR_TAGS[tag], msg, line))
            continue
        if tag == "filter" or (tag.startswith("fe") and tag[2:3].isupper()):
            msg = f"không tự viết <{tag}> — bóng, hạt, quầng sáng do kit tính qua <Shape material>"
            errors.append(StyleFinding("S1", msg, line))
            continue
        if tag in GRADIENT_TAGS:
            gradients += 1
        if tag in ID_TAGS and re.search(r"\bid\s*=\s*['\"]", attrs):
            msg = f"<{tag}> có id viết cứng — dùng id={{useSvgId('...')}} để hai bản của hình không trùng id"
            warnings.append(StyleFinding("S25", msg, line))
        if tag not in RAW_SHAPES:
            continue
        raw_shapes += 1
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

    limit = MAX_GRADIENTS[tier]
    if gradients > limit:
        errors.append(StyleFinding(
            "S1", f"{gradients} gradient tự viết — hạng {tier} tối đa {limit}; ánh sáng do kit tính, tô màu phẳng"))

    shapes = raw_shapes + len(re.findall(r"<Shape\b", code))
    low, high = SHAPE_RANGE[tier]
    if kind != BACKDROP and not low <= shapes <= high:
        warnings.append(StyleFinding(
            "S4", f"{shapes} hình con — hạng {tier} nên có {low}–{high}"))

    black_fill = re.search(r"fill\s*=\s*['\"{]*['\"]#0{6}['\"]", code)
    if black_fill and not re.search(r"#000000['\"][^>]*opacity", code):
        warnings.append(StyleFinding("S11", "đen thuần làm mảng màu — dùng màu tối của vật hoặc bóng opacity thấp"))

    if kind != BACKDROP and "color" not in re.split(r"\)\s*\{", code, maxsplit=1)[0]:
        warnings.append(StyleFinding("S12", "màu chính không đổi được qua prop color"))
    if "useCurrentFrame" in code and "still" not in code:
        warnings.append(StyleFinding("S20", "chuyển động không tắt được bằng prop still"))

    if kind != BACKDROP:
        warnings += _material_warnings(code, raw_shapes)
        if tier == HERO and not re.search(r"<(Part|Bone|Rig|Mechanism)\b", code):
            warnings.append(StyleFinding(
                "T1", "hình hạng chính nên chia bộ phận: <Part> trong <Mechanism>, hoặc <Bone> trong <Rig>"))
        if skeletons is not None:
            errors += _rig_errors(code, skeletons)
    return errors, warnings


def _material_warnings(code: str, raw_shapes: int) -> list[StyleFinding]:
    if raw_shapes >= RAW_SHAPES_WITHOUT_MATERIAL and "<Shape" not in code:
        return [StyleFinding("M1", "khối lớn nên là <Shape material=...> để kit tính ánh sáng, bóng và viền sáng")]
    return []


RIG_RE = re.compile(r"<Rig\b([^<>]*?)>", re.S)
BONE_RE = re.compile(r"<Bone\b[^<>]*?\bname\s*=\s*(?:\"([^\"]+)\"|'([^']+)'|\{\s*['\"]([^'\"]+)['\"]\s*\})", re.S)


def _rig_errors(code: str, skeletons: dict[str, dict[str, str | None]]) -> list[StyleFinding]:
    """R1: một <Rig> phải khai báo đủ xương của bộ xương nó chọn."""
    errors: list[StyleFinding] = []
    for m in RIG_RE.finditer(code):
        skeleton = _attr(m.group(1), "skeleton")
        line = _line(code, m.start())
        if skeleton is None:
            errors.append(StyleFinding("R1", "<Rig> thiếu prop skeleton", line))
            continue
        if skeleton not in skeletons:
            errors.append(StyleFinding(
                "R1", f"bộ xương {skeleton!r} không có (có: {', '.join(sorted(skeletons))})", line))
            continue
        declared = {a or b or c for a, b, c in BONE_RE.findall(code)}
        missing = [bone for bone in skeletons[skeleton] if bone not in declared]
        if missing:
            errors.append(StyleFinding("R1", f"rig {skeleton} thiếu xương: {', '.join(missing)}", line))
    return errors


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
    for face in ("<Eyes", "<Mouth", "<Brows"):
        idx = code.find(face)
        if idx >= 0:
            errors.append(StyleFinding("B5", "nền là nơi chốn, không có mặt người", _line(code, idx)))
            break
    return errors
