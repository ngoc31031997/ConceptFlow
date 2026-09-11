"""Luật validate độ dài clip dọc theo preset (CR-007 FR19.6/19.7) — HÀM THUẦN.

Không đọc file, không gọi ffmpeg. Ngưỡng preset là config (C2b của CR-007 —
YouTube/TikTok đổi ngưỡng theo thời gian), đọc từ biến môi trường theo đúng
convention của `domain/qc_rules.py::QCThresholds.from_env()`.
"""

from __future__ import annotations

import os
from dataclasses import dataclass

CLIP_PRESET_SHORT_MAX_SECONDS = 60.0
CLIP_PRESET_LONG_MIN_SECONDS = 60.0
CLIP_PRESET_LONG_MAX_SECONDS = 180.0

PRESET_SHORT = "short"
PRESET_LONG = "long"


def _as_float(raw: str | None, default: float) -> float:
    """Giá trị không parse được rơi về mặc định thay vì ném — cùng lý do
    `qc_rules._as_float`: một biến môi trường gõ sai không được làm sập
    video-assembly."""
    if raw is None or raw.strip() == "":
        return default
    try:
        return float(raw)
    except ValueError:
        return default


@dataclass(frozen=True)
class ClipThresholds:
    """FR19.6/C2b — ngưỡng độ dài mỗi preset, cấu hình được bằng biến môi
    trường vì bên thứ ba (YouTube/TikTok) đổi ngưỡng theo thời gian."""

    short_max_seconds: float = CLIP_PRESET_SHORT_MAX_SECONDS
    long_min_seconds: float = CLIP_PRESET_LONG_MIN_SECONDS
    long_max_seconds: float = CLIP_PRESET_LONG_MAX_SECONDS

    @classmethod
    def from_env(cls, env: dict | None = None) -> "ClipThresholds":
        source = os.environ if env is None else env
        get = source.get
        return cls(
            short_max_seconds=_as_float(
                get("CLIP_PRESET_SHORT_MAX_SECONDS"), CLIP_PRESET_SHORT_MAX_SECONDS
            ),
            long_min_seconds=_as_float(
                get("CLIP_PRESET_LONG_MIN_SECONDS"), CLIP_PRESET_LONG_MIN_SECONDS
            ),
            long_max_seconds=_as_float(
                get("CLIP_PRESET_LONG_MAX_SECONDS"), CLIP_PRESET_LONG_MAX_SECONDS
            ),
        )


def validate_clip_duration(
    duration_seconds: float, preset: str, thresholds: ClipThresholds
) -> str | None:
    """Trả `None` nếu `duration_seconds` hợp lệ cho `preset`, hoặc thông báo
    lỗi rõ ràng nếu không (FR19.6 — không im lặng cắt cụt).

    FR19.7: mỗi preset được validate độc lập — một đoạn không hợp `short`
    vẫn có thể hợp `long`, gọi hàm này riêng cho từng preset và không đánh
    đổ preset kia.
    """
    if preset == PRESET_SHORT:
        if duration_seconds <= 0:
            return f"đoạn {duration_seconds:.1f}s không hợp lệ cho preset short"
        if duration_seconds > thresholds.short_max_seconds:
            return (
                f"đoạn {duration_seconds:.1f}s không đủ ngắn cho preset short "
                f"(tối đa {thresholds.short_max_seconds:.0f}s)"
            )
        return None
    if preset == PRESET_LONG:
        if duration_seconds < thresholds.long_min_seconds:
            return (
                f"đoạn {duration_seconds:.1f}s không đủ dài cho preset long "
                f"({thresholds.long_min_seconds:.0f}-{thresholds.long_max_seconds:.0f}s)"
            )
        if duration_seconds > thresholds.long_max_seconds:
            return (
                f"đoạn {duration_seconds:.1f}s không đủ dài cho preset long "
                f"({thresholds.long_min_seconds:.0f}-{thresholds.long_max_seconds:.0f}s)"
            )
        return None
    return f"preset không xác định: {preset!r}"
