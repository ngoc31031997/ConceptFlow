"""Sinh clip dọc 9:16 từ video 16:9 đã ghép (CR-007 FR19, LLD D5/D6).

Hàm thuần về mặt điều khiển (validate trước, không im lặng cắt cụt — D4),
nhưng CÓ side effect: gọi ffmpeg qua `FfmpegVideoAssembler._run_ffmpeg` để cắt
đoạn thật. Tái dùng đúng `VIDEO_ENCODE_ARGS`/`AUDIO_ENCODE_ARGS`/
`CONTAINER_ARGS` của `adapters/assembly/ffmpeg_assembler.py` — clip cũng là
file đem đăng, không có lý do để nó kém hơn video chính.

Mỗi (request, preset) độc lập: một clip lỗi validate hoặc lỗi ffmpeg không
được làm hỏng các clip khác trong cùng lần chạy `generate_clips` (D1).
"""

from __future__ import annotations

import logging
import os
import re

from adapters.assembly.ffmpeg_assembler import (
    AUDIO_ENCODE_ARGS,
    CONTAINER_ARGS,
    VIDEO_ENCODE_ARGS,
    FfmpegVideoAssembler,
)
from adapters.assembly.subtitle_file import write_subtitle_file
from adapters.storage.artifact_paths import clip_output_path, ensure_parent_dir
from domain.clip_rules import ClipThresholds, validate_clip_duration
from domain.errors import AssemblyEngineError
from domain.models import SubtitleCue, SubtitleStyle

logger = logging.getLogger(__name__)

# D5 — nền là chính khung gốc phóng to rồi làm mờ, video gốc căn giữa. Manim
# hay đặt công thức ở rìa khung nên crop là mất nội dung.
VERTICAL_WIDTH = 1080
VERTICAL_HEIGHT = 1920
BOXBLUR_STRENGTH = "20:2"

# D6 — style riêng cho clip dọc: xem trên điện thoại, phụ đề phải to hơn hẳn
# và đặt giữa khung, khác hẳn style clip ngang.
VERTICAL_SUBTITLE_STYLE = SubtitleStyle(font_size="large", position="center")
VERTICAL_PLAY_RES = (VERTICAL_WIDTH, VERTICAL_HEIGHT)

_SLUG_NON_ALNUM = re.compile(r"[^a-z0-9]+")


def slugify(name: str) -> str:
    """lowercase, khoảng trắng/ký tự đặc biệt -> '-'. Không cần unicode-aware
    phức tạp: tên clip do Creator đặt, output path chỉ cần ổn định và đọc
    được, không cần đẹp."""
    slug = _SLUG_NON_ALNUM.sub("-", name.strip().lower()).strip("-")
    return slug or "clip"


class ClipRequest:
    """Đối chiếu D3's `ClipRequest{ name, start_seconds, end_seconds,
    presets }` — dựng ở handler từ payload["requests"], một instance ở đây
    ứng với một `(request, preset)` đã tách phẳng."""

    __slots__ = ("name", "start_seconds", "end_seconds")

    def __init__(self, name: str, start_seconds: float, end_seconds: float) -> None:
        self.name = name
        self.start_seconds = start_seconds
        self.end_seconds = end_seconds


def _shift_and_clamp_cues(
    cues: list[SubtitleCue], effective_start: float, effective_end: float
) -> list[SubtitleCue]:
    """D6 rủi ro: dịch cue về mốc 0 của clip, bỏ cue ngoài khoảng, clamp cue
    vắt qua biên. Đây là chỗ dễ sai nhất của CR này — test khoá bằng số."""
    duration = effective_end - effective_start
    result: list[SubtitleCue] = []
    for cue in cues:
        # Bỏ hẳn cue không giao với [effective_start, effective_end).
        if cue.end_time <= effective_start or cue.start_time >= effective_end:
            continue
        start = max(cue.start_time, effective_start) - effective_start
        end = min(cue.end_time, effective_end) - effective_start
        end = min(end, duration)
        start = max(start, 0.0)
        if end <= start:
            continue
        result.append(
            SubtitleCue(
                scene_index=cue.scene_index,
                text=cue.text,
                start_time=start,
                end_time=end,
            )
        )
    return result


def _build_filtergraph(subtitle_path: str | None) -> tuple[str, str]:
    """Trả (filter_complex, video_map_label)."""
    parts = [
        "[0:v]split=2[bg][fg]",
        f"[bg]scale={VERTICAL_WIDTH}:{VERTICAL_HEIGHT}:force_original_aspect_ratio=increase,"
        f"crop={VERTICAL_WIDTH}:{VERTICAL_HEIGHT},boxblur={BOXBLUR_STRENGTH}[blurred]",
        f"[fg]scale={VERTICAL_WIDTH}:-2[scaled]",
    ]
    if subtitle_path is not None:
        from adapters.assembly.ffmpeg_assembler import _escape_filter_path

        parts.append(
            f"[blurred][scaled]overlay=(W-w)/2:(H-h)/2,"
            f"subtitles={_escape_filter_path(subtitle_path)}[vout]"
        )
    else:
        parts.append("[blurred][scaled]overlay=(W-w)/2:(H-h)/2[vout]")
    return ";".join(parts), "[vout]"


def generate_clip(
    *,
    project_id: str,
    video_path: str,
    request: ClipRequest,
    preset: str,
    intro_duration_seconds: float,
    subtitle_cues: list[SubtitleCue],
    thresholds: ClipThresholds,
) -> dict:
    """Cắt một `(request, preset)` thành một clip dọc. Trả dict theo đúng
    shape một phần tử của `clips_generated.clips`:
    `{"name", "preset", "status": "ok"|"error", ...}`.

    KHÔNG ném ngoại lệ cho lỗi validate/ffmpeg — cả hai được bắt và trả về
    như một clip status="error", để vòng lặp gọi hàm này tiếp tục với các
    (request, preset) khác (D1: lỗi sinh clip không chặn publish).
    """
    name = request.name
    # D5 rủi ro / CR-023: mốc trong clip_marks là giây trong video Manim gốc,
    # chưa cộng intro — cộng offset y hệt effective_lead_in của ffmpeg_assembler.
    effective_start = request.start_seconds + intro_duration_seconds
    effective_end = request.end_seconds + intro_duration_seconds
    duration = effective_end - effective_start

    error = validate_clip_duration(duration, preset, thresholds)
    if error is not None:
        return {"name": name, "preset": preset, "status": "error", "error_message": error}

    slug = slugify(name)
    output_path = clip_output_path(project_id, slug, preset)
    ensure_parent_dir(output_path)

    cues = _shift_and_clamp_cues(subtitle_cues or [], effective_start, effective_end)

    subtitle_path: str | None = None
    try:
        if cues:
            subtitle_path = os.path.join(
                os.path.dirname(output_path), f"{slug}_{preset}.ass"
            )
            write_subtitle_file(
                cues, VERTICAL_SUBTITLE_STYLE, subtitle_path, play_res=VERTICAL_PLAY_RES
            )

        filter_complex, video_map = _build_filtergraph(subtitle_path)

        cmd = [
            "-y",
            "-ss", f"{effective_start:.3f}",
            "-i", video_path,
            "-t", f"{duration:.3f}",
            "-filter_complex", filter_complex,
            "-map", video_map,
            "-map", "0:a?",
            *VIDEO_ENCODE_ARGS,
            *AUDIO_ENCODE_ARGS,
            *CONTAINER_ARGS,
            output_path,
        ]
        FfmpegVideoAssembler._run_ffmpeg(cmd)
    except AssemblyEngineError as exc:
        logger.warning("generate_clips: lỗi sinh clip %r/%s: %s", name, preset, exc)
        return {"name": name, "preset": preset, "status": "error", "error_message": str(exc)}

    return {
        "name": name,
        "preset": preset,
        "status": "ok",
        "output_path": output_path,
        "duration_seconds": duration,
    }
