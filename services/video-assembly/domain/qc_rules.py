"""Luật chấm chất lượng video (CR-021 FR59/FR60) — HÀM THUẦN.

Không đọc file, không gọi ffmpeg, không async. Mọi số đo (LUFS, peak, độ phân
giải, thời lượng audio) được adapter `adapters/qc/ffmpeg_probe.py` đo trước rồi
đưa vào đây dưới dạng dữ liệu thuần. Đây là chỗ toàn bộ unit test của FR59/FR60
trỏ vào (LLD D4).

Ngưỡng (FR61.5) nằm trọn trong `QCThresholds`, đọc từ biến môi trường —
KHÔNG hằng số rải trong thân luật. Lý do là rủi ro số một của CR: báo động giả.
Ngưỡng khởi đầu nới rộng, siết dần bằng biến môi trường sau khi đo trên video
thật. Module này KHÔNG biết gì về `QC_ENFORCE` — severity là thuộc tính của
phát hiện, còn việc chặn publish là của orchestrator (LLD D5).
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field

# --- Khung hình Manim ---------------------------------------------------------
# Manim 0.18 mặc định: `config.frame_height` = 8.0, tỉ lệ 16:9 nên
# `config.frame_width` = 8 * 16/9 = 14.2222…  → x ∈ [-7.111, 7.111],
# y ∈ [-4, 4]. Chép thành hằng số ở đây vì video-assembly KHÔNG cài Manim.
# Nguồn đối chiếu: services/rendering/conceptflow/theme.py (FRAME_WIDTH,
# FRAME_HEIGHT) — cùng hai con số, cùng lý do.
FRAME_WIDTH = 14.222222222222221
FRAME_HEIGHT = 8.0

# Nguồn: services/rendering/conceptflow/theme.py::SAFE_MARGIN = 0.6.
# Chép (không import) — video-assembly không phụ thuộc gói rendering. Nếu
# theme.py đổi con số này, đổi ở đây, hoặc đặt QC_SAFE_MARGIN.
THEME_SAFE_MARGIN = 0.6

# Nguồn: services/rendering/conceptflow/theme.py::BRAND_BG — nền của mọi cảnh,
# dùng làm màu nền để tính tương phản (FR59.4).
THEME_BACKGROUND = "#080E1C"

#: Chỉ những class này mới bị áp luật chồng lấn/chữ nhỏ/tương phản (FR59.2).
#: `VGroup`/`Group` bị loại vì bbox của nhóm bao trùm con nó nên nhóm nào cũng
#: "chồng" con nó — một trong ba nguồn hiểu nhầm CR nêu đích danh (LLD Rủi ro).
TEXT_CLASSES = frozenset({"Text", "MarkupText", "Tex", "MathTex", "Title", "Paragraph"})

#: Khung pixel theo `render_quality` — cùng bảng với
#: adapters/messaging/consumer.py::QUALITY_FRAME_SIZE.
QUALITY_FRAME_SIZE = {
    "720p30": (1280, 720),
    "1080p60": (1920, 1080),
    "4k60": (3840, 2160),
}
QUALITY_FRAME_RATE = {"720p30": 30.0, "1080p60": 60.0, "4k60": 60.0}
DEFAULT_QUALITY = "1080p60"

SEVERITY_BLOCKING = "blocking"
SEVERITY_WARNING = "warning"

#: Hai luật mang severity blocking (Quyết định #1 của CR): tràn khung và chồng
#: lấn narration. Mọi luật còn lại là cảnh báo.
#:
#: Severity ở đây là **thuộc tính của phát hiện**, không phải của chính sách:
#: nó luôn được ghi đúng vào báo cáo, kể cả lúc chạy chế độ chỉ-báo (LLD D5).
#: Việc một finding blocking có chặn publish hay không do `QC_ENFORCE` bên
#: orchestrator quyết định — service này không đọc biến đó. Hạ cấp severity ở
#: đây sẽ phá đúng mục đích của chế độ chỉ-báo: đo xem luật blocking kêu đúng
#: bao nhiêu phần trăm trước khi bật cổng.
BLOCKING_RULES = frozenset({"frame_overflow", "narration_overlap"})


@dataclass(frozen=True)
class QCFinding:
    rule: str
    severity: str
    message: str
    #: FR59.6 — luôn có, để Creator tua thẳng tới chỗ đó. Luật xét cả file
    #: (LUFS, thuộc tính phát hành) dùng 0.0 = đầu video.
    timestamp_seconds: float


def _as_float(raw: str | None, default: float) -> float:
    """Giá trị không parse được rơi về mặc định thay vì ném — một biến môi
    trường gõ sai không được làm sập QC (FR61.4)."""
    if raw is None or raw.strip() == "":
        return default
    try:
        return float(raw)
    except ValueError:
        return default


def _as_bool(raw: str | None, default: bool) -> bool:
    if raw is None or raw.strip() == "":
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on"}


@dataclass(frozen=True)
class QCThresholds:
    """FR61.5 — mọi ngưỡng ở một chỗ, cấu hình được bằng biến môi trường."""

    # --- FR59.1 tràn khung ---
    safe_margin: float = THEME_SAFE_MARGIN
    #: Dung sai (đơn vị Manim) trước khi kêu — bbox Manim rộng hơn nét chữ thật
    #: vài phần trăm, kêu ở 0.0 là kêu suốt.
    frame_overflow_tolerance: float = 0.05

    # --- FR59.2 chồng lấn ---
    #: Tỉ lệ diện tích giao trên diện tích của mobject NHỎ HƠN. 0.15 nới rộng có
    #: chủ ý: chữ có dấu tiếng Việt hay chạm nhẹ nhau mà mắt không thấy.
    text_overlap_min_ratio: float = 0.15

    # --- FR59.3 chữ quá nhỏ ---
    #: Chiều cao chữ tối thiểu tính bằng pixel Ở ĐỘ PHÂN GIẢI XUẤT.
    #: Căn cứ (Quyết định #2): màn 5,5 inch xem toàn màn hình → khung 16:9 cao
    #: ~68 mm. 24 px trên 1080 dòng ≈ 1,5 mm chiều cao em — sàn đọc thoải mái ở
    #: khoảng cách cầm tay ~30 cm. Ở 720p cùng cỡ chữ chỉ còn 16 px nên bị kêu,
    #: và đó là đúng: chữ nhỏ ở bitrate thấp còn bị encode làm nhoè thêm.
    min_text_pixel_height: float = 24.0

    # --- FR59.4 tương phản ---
    #: Tỉ số tương phản WCAG. 3.0 là ngưỡng WCAG AA cho chữ LỚN — chữ trên video
    #: gần như luôn là chữ lớn, và ngưỡng 4.5 sẽ kêu cả những cặp màu theme đã
    #: chọn có chủ ý.
    min_contrast_ratio: float = 3.0
    background_color: str = THEME_BACKGROUND

    # --- FR59.5 hình chết ---
    max_static_seconds: float = 12.0

    # --- FR60.1 LUFS ---
    loudness_target_lufs: float = -14.0
    loudness_tolerance_lu: float = 1.5

    # --- FR60.2 chồng lấn narration ---
    #: Đoạn audio được phép tràn sang mốc kế tiếp chừng này giây trước khi kêu —
    #: đuôi im lặng của file TTS thường dài cỡ này.
    narration_overlap_tolerance_seconds: float = 0.15

    # --- FR60.3 clipping ---
    #: dBFS. Trên -0.1 dBFS coi như đã chạm trần.
    max_peak_dbfs: float = -0.1

    # --- FR60.4 cue phụ đề ---
    subtitle_overlap_tolerance_seconds: float = 0.05

    # --- FR60.5 thuộc tính phát hành ---
    required_pix_fmt: str = "yuv420p"
    require_faststart: bool = True

    #: Ghi đè bảng khung pixel (chỉ dùng trong test).
    quality_frame_size: dict = field(default_factory=lambda: dict(QUALITY_FRAME_SIZE))

    @classmethod
    def from_env(cls, env: dict | None = None) -> "QCThresholds":
        """FR61.5 — mọi ngưỡng đọc từ môi trường, mọi cái vắng mặt lấy mặc định
        ở trên. Giá trị không parse được rơi về mặc định thay vì làm sập QC:
        một cổng hỏng không được biến thành cổng khoá (FR61.4)."""
        source = os.environ if env is None else env
        get = source.get
        return cls(
            safe_margin=_as_float(get("QC_SAFE_MARGIN"), THEME_SAFE_MARGIN),
            frame_overflow_tolerance=_as_float(get("QC_FRAME_OVERFLOW_TOLERANCE"), 0.05),
            text_overlap_min_ratio=_as_float(get("QC_TEXT_OVERLAP_MIN_RATIO"), 0.15),
            min_text_pixel_height=_as_float(get("QC_MIN_TEXT_PIXEL_HEIGHT"), 24.0),
            min_contrast_ratio=_as_float(get("QC_MIN_CONTRAST_RATIO"), 3.0),
            background_color=get("QC_BACKGROUND_COLOR") or THEME_BACKGROUND,
            max_static_seconds=_as_float(get("QC_MAX_STATIC_SECONDS"), 12.0),
            loudness_target_lufs=_as_float(get("QC_LOUDNESS_TARGET_LUFS"), -14.0),
            loudness_tolerance_lu=_as_float(get("QC_LOUDNESS_TOLERANCE_LU"), 1.5),
            narration_overlap_tolerance_seconds=_as_float(
                get("QC_NARRATION_OVERLAP_TOLERANCE_SECONDS"), 0.15
            ),
            max_peak_dbfs=_as_float(get("QC_MAX_PEAK_DBFS"), -0.1),
            subtitle_overlap_tolerance_seconds=_as_float(
                get("QC_SUBTITLE_OVERLAP_TOLERANCE_SECONDS"), 0.05
            ),
            required_pix_fmt=get("QC_REQUIRED_PIX_FMT") or "yuv420p",
            require_faststart=_as_bool(get("QC_REQUIRE_FASTSTART"), True),
        )

    def severity_for(self, rule: str) -> str:
        """LLD D5: severity thật luôn được ghi vào báo cáo. Cổng chặn là việc
        của orchestrator (`QC_ENFORCE`), không phải của người chấm."""
        return SEVERITY_BLOCKING if rule in BLOCKING_RULES else SEVERITY_WARNING


# --- Tiện ích bố cục ----------------------------------------------------------


def _bbox(mobject: dict) -> tuple[float, float, float, float] | None:
    """(left, right, top, bottom) theo toạ độ Manim, hoặc None nếu bbox hỏng."""
    raw = mobject.get("bbox")
    if not raw or len(raw) != 4:
        return None
    try:
        left, right, top, bottom = (float(v) for v in raw)
    except (TypeError, ValueError):
        return None
    # Chuẩn hoá: một mark ghi ngược thứ tự vẫn dùng được.
    if right < left:
        left, right = right, left
    if top < bottom:
        top, bottom = bottom, top
    return left, right, top, bottom


def _is_text(mobject: dict) -> bool:
    return mobject.get("cls") in TEXT_CLASSES


def _is_visible(mobject: dict) -> bool:
    """Mobject trong suốt bị bỏ qua (LLD Rủi ro) — đã `FadeOut` nhưng chưa
    remove thì bbox vẫn còn đó mà mắt không thấy gì."""
    for key in ("opacity", "fill_opacity", "stroke_opacity"):
        value = mobject.get(key)
        if value is not None:
            try:
                if float(value) > 0.0:
                    return True
            except (TypeError, ValueError):
                return True
            return False
    return True  # không khai báo opacity → coi như hiện


def _mark_time(mark: dict) -> float:
    try:
        return float(mark.get("t") or 0.0)
    except (TypeError, ValueError):
        return 0.0


def _iter_mobjects(layout_marks: list[dict]):
    for mark in layout_marks or []:
        timestamp = _mark_time(mark)
        for mobject in mark.get("mobjects") or []:
            if isinstance(mobject, dict):
                yield timestamp, mobject


def _label(mobject: dict) -> str:
    cls = mobject.get("cls") or "mobject"
    text = (mobject.get("text") or "").strip()
    if text:
        snippet = text if len(text) <= 30 else text[:27] + "…"
        return f'{cls} "{snippet}"'
    return cls


# --- FR59.1 ------------------------------------------------------------------


def check_frame_overflow(layout_marks: list[dict], thresholds: QCThresholds) -> list[QCFinding]:
    """Bbox vượt khung Manim, hoặc lấn vào safe margin của theme."""
    severity = thresholds.severity_for("frame_overflow")
    half_width = FRAME_WIDTH / 2.0
    half_height = FRAME_HEIGHT / 2.0
    tol = thresholds.frame_overflow_tolerance
    margin = thresholds.safe_margin

    findings: list[QCFinding] = []
    for timestamp, mobject in _iter_mobjects(layout_marks):
        if not _is_visible(mobject):
            continue
        box = _bbox(mobject)
        if box is None:
            continue
        left, right, top, bottom = box

        outside = (
            left < -half_width - tol
            or right > half_width + tol
            or bottom < -half_height - tol
            or top > half_height + tol
        )
        if outside:
            findings.append(
                QCFinding(
                    rule="frame_overflow",
                    severity=severity,
                    message=(
                        f"{_label(mobject)} tràn ra ngoài khung "
                        f"(bbox l={left:.2f} r={right:.2f} t={top:.2f} b={bottom:.2f}, "
                        f"khung ±{half_width:.2f} × ±{half_height:.2f})"
                    ),
                    timestamp_seconds=timestamp,
                )
            )
            continue

        inner_x = half_width - margin
        inner_y = half_height - margin
        if (
            left < -inner_x - tol
            or right > inner_x + tol
            or bottom < -inner_y - tol
            or top > inner_y + tol
        ):
            findings.append(
                QCFinding(
                    rule="frame_overflow",
                    severity=severity,
                    message=(
                        f"{_label(mobject)} lấn vào safe margin {margin:.2f} "
                        f"(bbox l={left:.2f} r={right:.2f} t={top:.2f} b={bottom:.2f})"
                    ),
                    timestamp_seconds=timestamp,
                )
            )
    return findings


# --- FR59.2 ------------------------------------------------------------------


def _area(box: tuple[float, float, float, float]) -> float:
    left, right, top, bottom = box
    return max(0.0, right - left) * max(0.0, top - bottom)


def _intersection_area(a, b) -> float:
    left = max(a[0], b[0])
    right = min(a[1], b[1])
    top = min(a[2], b[2])
    bottom = max(a[3], b[3])
    return max(0.0, right - left) * max(0.0, top - bottom)


def check_text_overlap(layout_marks: list[dict], thresholds: QCThresholds) -> list[QCFinding]:
    """Chỉ áp cho mobject dạng chữ, hiện hình. VGroup/Group bị bỏ qua có chủ ý:
    bbox của nhóm bao trùm con nó nên mọi nhóm đều "chồng" con mình."""
    severity = thresholds.severity_for("text_overlap")
    findings: list[QCFinding] = []

    for mark in layout_marks or []:
        timestamp = _mark_time(mark)
        boxes = []
        for mobject in mark.get("mobjects") or []:
            if not isinstance(mobject, dict) or not _is_text(mobject) or not _is_visible(mobject):
                continue
            box = _bbox(mobject)
            if box is None or _area(box) <= 0.0:
                continue
            boxes.append((mobject, box))

        for i in range(len(boxes)):
            for j in range(i + 1, len(boxes)):
                mobject_a, box_a = boxes[i]
                mobject_b, box_b = boxes[j]
                overlap = _intersection_area(box_a, box_b)
                if overlap <= 0.0:
                    continue
                ratio = overlap / min(_area(box_a), _area(box_b))
                if ratio > thresholds.text_overlap_min_ratio:
                    findings.append(
                        QCFinding(
                            rule="text_overlap",
                            severity=severity,
                            message=(
                                f"{_label(mobject_a)} và {_label(mobject_b)} chồng lấn "
                                f"{ratio * 100:.0f}% diện tích"
                            ),
                            timestamp_seconds=timestamp,
                        )
                    )
    return findings


# --- FR59.3 ------------------------------------------------------------------


def font_size_to_pixels(font_size: float, render_quality: str, thresholds: QCThresholds) -> float:
    """Quy đổi `font_size` của Manim ra chiều cao pixel ở độ phân giải xuất.

    Manim đặt chiều cao một em bằng `font_size / 100` đơn vị khung, và khung cao
    `FRAME_HEIGHT` đơn vị được vẽ lên `pixel_height` dòng — nên
    px = font_size / 100 * pixel_height / FRAME_HEIGHT.
    Ví dụ: cỡ `caption` 20 của theme ở 1080p → 0.20 × 135 = 27 px.
    """
    _, pixel_height = thresholds.quality_frame_size.get(
        render_quality, thresholds.quality_frame_size.get(DEFAULT_QUALITY, (1920, 1080))
    )
    return font_size / 100.0 * pixel_height / FRAME_HEIGHT


def check_text_too_small(
    layout_marks: list[dict], render_quality: str, thresholds: QCThresholds
) -> list[QCFinding]:
    severity = thresholds.severity_for("text_too_small")
    findings: list[QCFinding] = []
    for timestamp, mobject in _iter_mobjects(layout_marks):
        if not _is_text(mobject) or not _is_visible(mobject):
            continue
        raw = mobject.get("font_size")
        if raw is None:
            continue  # Manim không phải mobject nào cũng có cỡ chữ
        try:
            font_size = float(raw)
        except (TypeError, ValueError):
            continue
        pixels = font_size_to_pixels(font_size, render_quality, thresholds)
        if pixels < thresholds.min_text_pixel_height:
            findings.append(
                QCFinding(
                    rule="text_too_small",
                    severity=severity,
                    message=(
                        f"{_label(mobject)} cỡ {font_size:.0f} ≈ {pixels:.0f}px ở "
                        f"{render_quality}, dưới ngưỡng đọc được "
                        f"{thresholds.min_text_pixel_height:.0f}px trên màn 5,5 inch"
                    ),
                    timestamp_seconds=timestamp,
                )
            )
    return findings


# --- FR59.4 ------------------------------------------------------------------


def _parse_hex_color(value: str | None) -> tuple[float, float, float] | None:
    if not isinstance(value, str):
        return None
    raw = value.strip().lstrip("#")
    if len(raw) == 3:
        raw = "".join(ch * 2 for ch in raw)
    if len(raw) == 8:  # #RRGGBBAA
        raw = raw[:6]
    if len(raw) != 6:
        return None
    try:
        return tuple(int(raw[i : i + 2], 16) / 255.0 for i in (0, 2, 4))  # type: ignore[return-value]
    except ValueError:
        return None


def _relative_luminance(rgb: tuple[float, float, float]) -> float:
    """WCAG 2.1 relative luminance."""

    def channel(c: float) -> float:
        return c / 12.92 if c <= 0.03928 else ((c + 0.055) / 1.055) ** 2.4

    r, g, b = (channel(c) for c in rgb)
    return 0.2126 * r + 0.7152 * g + 0.0722 * b


def contrast_ratio(foreground: str, background: str) -> float | None:
    """Tỉ số tương phản WCAG, 1.0 (không phân biệt) → 21.0 (đen/trắng)."""
    fg = _parse_hex_color(foreground)
    bg = _parse_hex_color(background)
    if fg is None or bg is None:
        return None
    lighter, darker = sorted((_relative_luminance(fg), _relative_luminance(bg)), reverse=True)
    return (lighter + 0.05) / (darker + 0.05)


def check_low_contrast(layout_marks: list[dict], thresholds: QCThresholds) -> list[QCFinding]:
    severity = thresholds.severity_for("low_contrast")
    findings: list[QCFinding] = []
    for timestamp, mobject in _iter_mobjects(layout_marks):
        if not _is_text(mobject) or not _is_visible(mobject):
            continue
        ratio = contrast_ratio(mobject.get("color"), thresholds.background_color)
        if ratio is None:
            continue
        if ratio < thresholds.min_contrast_ratio:
            findings.append(
                QCFinding(
                    rule="low_contrast",
                    severity=severity,
                    message=(
                        f"{_label(mobject)} màu {mobject.get('color')} trên nền "
                        f"{thresholds.background_color} chỉ đạt tương phản {ratio:.2f}:1, "
                        f"dưới ngưỡng {thresholds.min_contrast_ratio:.1f}:1"
                    ),
                    timestamp_seconds=timestamp,
                )
            )
    return findings


# --- FR59.5 ------------------------------------------------------------------


def check_static_frame(
    narration_segments: list[dict], thresholds: QCThresholds
) -> list[QCFinding]:
    """Khoảng giữa hai mốc narration dài hơn ngưỡng."""
    severity = thresholds.severity_for("static_frame")
    starts = sorted(float(s.get("start_time") or 0.0) for s in narration_segments or [])
    findings: list[QCFinding] = []
    for previous, current in zip(starts, starts[1:]):
        gap = current - previous
        if gap > thresholds.max_static_seconds:
            findings.append(
                QCFinding(
                    rule="static_frame",
                    severity=severity,
                    message=(
                        f"{gap:.1f}s không có narration giữa {previous:.1f}s và "
                        f"{current:.1f}s (ngưỡng {thresholds.max_static_seconds:.0f}s)"
                    ),
                    timestamp_seconds=previous,
                )
            )
    return findings


# --- FR60.2 ------------------------------------------------------------------


def check_narration_overlap(
    narration_segments: list[dict], thresholds: QCThresholds
) -> list[QCFinding]:
    """Một đoạn audio kéo dài quá điểm bắt đầu của đoạn kế tiếp.

    `duration_seconds` là thời lượng thật ffprobe đo được, do adapter gắn vào
    trước khi gọi. Đoạn không đo được thời lượng bị bỏ qua thay vì đoán."""
    severity = thresholds.severity_for("narration_overlap")
    segments = sorted(
        (s for s in narration_segments or []),
        key=lambda s: float(s.get("start_time") or 0.0),
    )
    findings: list[QCFinding] = []
    for current, following in zip(segments, segments[1:]):
        duration = current.get("duration_seconds")
        if duration is None:
            continue
        try:
            end = float(current.get("start_time") or 0.0) + float(duration)
        except (TypeError, ValueError):
            continue
        next_start = float(following.get("start_time") or 0.0)
        overlap = end - next_start
        if overlap > thresholds.narration_overlap_tolerance_seconds:
            findings.append(
                QCFinding(
                    rule="narration_overlap",
                    severity=severity,
                    message=(
                        f"đoạn narration bắt đầu ở {float(current.get('start_time') or 0.0):.2f}s "
                        f"kéo tới {end:.2f}s, chồng {overlap:.2f}s lên đoạn kế tiếp "
                        f"({next_start:.2f}s)"
                    ),
                    timestamp_seconds=next_start,
                )
            )
    return findings


# --- FR60.4 ------------------------------------------------------------------


def check_subtitle_cue_overlap(
    subtitle_cues: list[dict], thresholds: QCThresholds
) -> list[QCFinding]:
    severity = thresholds.severity_for("subtitle_cue_overlap")
    cues = sorted(
        (c for c in subtitle_cues or []), key=lambda c: float(c.get("start_time") or 0.0)
    )
    findings: list[QCFinding] = []
    for current, following in zip(cues, cues[1:]):
        end = float(current.get("end_time") or 0.0)
        next_start = float(following.get("start_time") or 0.0)
        overlap = end - next_start
        if overlap > thresholds.subtitle_overlap_tolerance_seconds:
            findings.append(
                QCFinding(
                    rule="subtitle_cue_overlap",
                    severity=severity,
                    message=(
                        f"cue kết thúc ở {end:.2f}s chồng {overlap:.2f}s lên cue bắt đầu ở "
                        f"{next_start:.2f}s"
                    ),
                    timestamp_seconds=next_start,
                )
            )
    return findings


# --- FR60.1 ------------------------------------------------------------------


def check_loudness(measured_lufs: float | None, thresholds: QCThresholds) -> list[QCFinding]:
    """FR60.1 — CR này chỉ ĐO VÀ BÁO. Chuyển loudnorm sang hai lượt là thay đổi
    assembly có chi phí, chỉ làm khi số đo chứng minh sai lệch đủ lớn (LLD)."""
    if measured_lufs is None:
        return []
    deviation = measured_lufs - thresholds.loudness_target_lufs
    if abs(deviation) <= thresholds.loudness_tolerance_lu:
        return []
    return [
        QCFinding(
            rule="loudness_off_target",
            severity=thresholds.severity_for("loudness_off_target"),
            message=(
                f"đo được {measured_lufs:.1f} LUFS, lệch {deviation:+.1f} LU so với đích "
                f"{thresholds.loudness_target_lufs:.0f} LUFS "
                f"(dung sai ±{thresholds.loudness_tolerance_lu:.1f} LU)"
            ),
            timestamp_seconds=0.0,
        )
    ]


# --- FR60.3 ------------------------------------------------------------------


def check_clipping(peak_dbfs: float | None, thresholds: QCThresholds) -> list[QCFinding]:
    if peak_dbfs is None:
        return []
    if peak_dbfs <= thresholds.max_peak_dbfs:
        return []
    return [
        QCFinding(
            rule="clipping",
            severity=thresholds.severity_for("clipping"),
            message=(
                f"peak {peak_dbfs:.2f} dBFS vượt ngưỡng {thresholds.max_peak_dbfs:.2f} dBFS — "
                f"âm thanh chạm trần và méo"
            ),
            timestamp_seconds=0.0,
        )
    ]


# --- FR60.5 ------------------------------------------------------------------


def check_publish_attributes(
    attributes: dict | None, render_quality: str, thresholds: QCThresholds
) -> list[QCFinding]:
    """Độ phân giải / framerate / pix_fmt / +faststart của FILE CUỐI.

    Tồn tại vì nhánh `-c:v copy` của assembly bỏ qua bước encode, nên không có
    gì đảm bảo file cuối vẫn mang đúng thuộc tính VIDEO_ENCODE_ARGS đặt ra."""
    if not attributes:
        return []
    severity = thresholds.severity_for("publish_attributes")
    findings: list[QCFinding] = []

    expected_size = thresholds.quality_frame_size.get(render_quality)
    width = attributes.get("width")
    height = attributes.get("height")
    if expected_size and width and height and (int(width), int(height)) != expected_size:
        findings.append(
            QCFinding(
                rule="publish_attributes",
                severity=severity,
                message=(
                    f"độ phân giải {int(width)}x{int(height)} không khớp {render_quality} "
                    f"({expected_size[0]}x{expected_size[1]})"
                ),
                timestamp_seconds=0.0,
            )
        )

    expected_fps = QUALITY_FRAME_RATE.get(render_quality)
    fps = attributes.get("frame_rate")
    if expected_fps and fps and abs(float(fps) - expected_fps) > 0.5:
        findings.append(
            QCFinding(
                rule="publish_attributes",
                severity=severity,
                message=f"framerate {float(fps):.2f} không khớp {render_quality} ({expected_fps:.0f})",
                timestamp_seconds=0.0,
            )
        )

    pix_fmt = attributes.get("pix_fmt")
    if pix_fmt and pix_fmt != thresholds.required_pix_fmt:
        findings.append(
            QCFinding(
                rule="publish_attributes",
                severity=severity,
                message=(
                    f"pix_fmt {pix_fmt} không phải {thresholds.required_pix_fmt} — "
                    f"nhiều nền tảng sẽ không phát được"
                ),
                timestamp_seconds=0.0,
            )
        )

    if thresholds.require_faststart and attributes.get("faststart") is False:
        findings.append(
            QCFinding(
                rule="publish_attributes",
                severity=severity,
                message="moov atom không nằm đầu file (+faststart) — video không phát được ngay khi tải",
                timestamp_seconds=0.0,
            )
        )

    return findings


# --- Chạy tất cả --------------------------------------------------------------


def evaluate_all(
    *,
    layout_marks: list[dict] | None,
    narration_segments: list[dict] | None,
    subtitle_cues: list[dict] | None,
    render_quality: str,
    measured_lufs: float | None = None,
    peak_dbfs: float | None = None,
    publish_attributes: dict | None = None,
    thresholds: QCThresholds | None = None,
) -> list[QCFinding]:
    """Chạy toàn bộ luật, trả findings đã sắp theo timestamp (FR59.6 — báo cáo
    đọc từ trên xuống là đi dọc video)."""
    t = thresholds or QCThresholds.from_env()
    marks = layout_marks or []
    segments = narration_segments or []

    findings: list[QCFinding] = []
    findings += check_frame_overflow(marks, t)
    findings += check_text_overlap(marks, t)
    findings += check_text_too_small(marks, render_quality, t)
    findings += check_low_contrast(marks, t)
    findings += check_static_frame(segments, t)
    findings += check_narration_overlap(segments, t)
    findings += check_subtitle_cue_overlap(subtitle_cues or [], t)
    findings += check_loudness(measured_lufs, t)
    findings += check_clipping(peak_dbfs, t)
    findings += check_publish_attributes(publish_attributes, render_quality, t)

    return sorted(findings, key=lambda f: (f.timestamp_seconds, f.rule))

