"""Luật chấm chất lượng video — HÀM THUẦN.

Không đọc file, không gọi ffmpeg, không async. Mọi số đo (LUFS, peak, độ phân
giải, thời lượng audio) được adapter `adapters/qc/ffmpeg_probe.py` đo trước rồi
đưa vào đây dưới dạng dữ liệu thuần, nên unit test của từng luật chỉ cần dữ
liệu dựng sẵn.

Ngưỡng nằm trọn trong `QCThresholds`, đọc từ biến môi trường — KHÔNG hằng số
rải trong thân luật, vì rủi ro lớn nhất của QC là báo động giả: ngưỡng khởi đầu
nới rộng, siết dần bằng biến môi trường sau khi đo trên video thật. Module này
KHÔNG biết gì về `QC_ENFORCE` — severity là thuộc tính của phát hiện, còn việc
chặn publish là của orchestrator.
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field

#: Khung pixel theo `render_quality` — cùng bảng với
#: adapters/messaging/consumer.py::QUALITY_FRAME_SIZE.
QUALITY_FRAME_SIZE = {
    "480p15": (854, 480),
    "720p30": (1280, 720),
    "1080p60": (1920, 1080),
    "4k60": (3840, 2160),
}
QUALITY_FRAME_RATE = {"480p15": 15.0, "720p30": 30.0, "1080p60": 60.0, "4k60": 60.0}

SEVERITY_BLOCKING = "blocking"
SEVERITY_WARNING = "warning"

#: Luật duy nhất mang severity blocking: chồng lấn narration. Mọi luật còn lại
#: là cảnh báo.
#:
#: Severity ở đây là **thuộc tính của phát hiện**, không phải của chính sách:
#: nó luôn được ghi đúng vào báo cáo, kể cả lúc chạy chế độ chỉ-báo.
#: Việc một finding blocking có chặn publish hay không do `QC_ENFORCE` bên
#: orchestrator quyết định — service này không đọc biến đó. Hạ cấp severity ở
#: đây sẽ phá đúng mục đích của chế độ chỉ-báo: đo xem luật blocking kêu đúng
#: bao nhiêu phần trăm trước khi bật cổng.
BLOCKING_RULES = frozenset({"narration_overlap"})


@dataclass(frozen=True)
class QCFinding:
    rule: str
    severity: str
    message: str
    #: Luôn có, để Creator tua thẳng tới chỗ đó. Luật xét cả file
    #: (LUFS, thuộc tính phát hành) dùng 0.0 = đầu video.
    timestamp_seconds: float


def _as_float(raw: str | None, default: float) -> float:
    """Giá trị không parse được rơi về mặc định thay vì ném — một biến môi
    trường gõ sai không được làm sập QC."""
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
    """Mọi ngưỡng ở một chỗ, cấu hình được bằng biến môi trường."""

    # --- hình chết ---
    max_static_seconds: float = 12.0

    # --- LUFS ---
    loudness_target_lufs: float = -14.0
    loudness_tolerance_lu: float = 1.5

    # --- chồng lấn narration ---
    #: Đoạn audio được phép tràn sang mốc kế tiếp chừng này giây trước khi kêu —
    #: đuôi im lặng của file TTS thường dài cỡ này.
    narration_overlap_tolerance_seconds: float = 0.15

    # --- clipping ---
    #: dBFS. Trên -0.1 dBFS coi như đã chạm trần.
    max_peak_dbfs: float = -0.1

    # --- cue phụ đề ---
    subtitle_overlap_tolerance_seconds: float = 0.05

    # --- thuộc tính phát hành ---
    required_pix_fmt: str = "yuv420p"
    require_faststart: bool = True

    #: Ghi đè bảng khung pixel (chỉ dùng trong test).
    quality_frame_size: dict = field(default_factory=lambda: dict(QUALITY_FRAME_SIZE))

    @classmethod
    def from_env(cls, env: dict | None = None) -> QCThresholds:
        """Mọi ngưỡng đọc từ môi trường, mọi cái vắng mặt lấy mặc định
        ở trên. Giá trị không parse được rơi về mặc định thay vì làm sập QC:
        một cổng hỏng không được biến thành cổng khoá."""
        source = os.environ if env is None else env
        get = source.get
        return cls(
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
        """Severity thật luôn được ghi vào báo cáo. Cổng chặn là việc
        của orchestrator (`QC_ENFORCE`), không phải của người chấm."""
        return SEVERITY_BLOCKING if rule in BLOCKING_RULES else SEVERITY_WARNING


# --- hình chết ---------------------------------------------------------------


def check_static_frame(
    narration_segments: list[dict], thresholds: QCThresholds
) -> list[QCFinding]:
    """Khoảng giữa hai mốc narration dài hơn ngưỡng."""
    severity = thresholds.severity_for("static_frame")
    starts = sorted(float(s.get("start_time") or 0.0) for s in narration_segments or [])
    findings: list[QCFinding] = []
    for previous, current in zip(starts, starts[1:], strict=False):
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


# --- chồng lấn narration -----------------------------------------------------


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
    for current, following in zip(segments, segments[1:], strict=False):
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


# --- cue phụ đề --------------------------------------------------------------


def check_subtitle_cue_overlap(
    subtitle_cues: list[dict], thresholds: QCThresholds
) -> list[QCFinding]:
    severity = thresholds.severity_for("subtitle_cue_overlap")
    cues = sorted(
        (c for c in subtitle_cues or []), key=lambda c: float(c.get("start_time") or 0.0)
    )
    findings: list[QCFinding] = []
    for current, following in zip(cues, cues[1:], strict=False):
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


# --- LUFS --------------------------------------------------------------------


def check_loudness(measured_lufs: float | None, thresholds: QCThresholds) -> list[QCFinding]:
    """Chỉ ĐO VÀ BÁO. Chuyển loudnorm sang hai lượt là thay đổi assembly có
    chi phí, chỉ đáng làm khi số đo chứng minh sai lệch đủ lớn."""
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


# --- clipping ----------------------------------------------------------------


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


# --- thuộc tính phát hành ----------------------------------------------------


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
    narration_segments: list[dict] | None,
    subtitle_cues: list[dict] | None,
    render_quality: str,
    measured_lufs: float | None = None,
    peak_dbfs: float | None = None,
    publish_attributes: dict | None = None,
    thresholds: QCThresholds | None = None,
) -> list[QCFinding]:
    """Chạy toàn bộ luật, trả findings đã sắp theo timestamp (báo cáo đọc từ
    trên xuống là đi dọc video)."""
    t = thresholds or QCThresholds.from_env()
    segments = narration_segments or []

    findings: list[QCFinding] = []
    findings += check_static_frame(segments, t)
    findings += check_narration_overlap(segments, t)
    findings += check_subtitle_cue_overlap(subtitle_cues or [], t)
    findings += check_loudness(measured_lufs, t)
    findings += check_clipping(peak_dbfs, t)
    findings += check_publish_attributes(publish_attributes, render_quality, t)

    return sorted(findings, key=lambda f: (f.timestamp_seconds, f.rule))

