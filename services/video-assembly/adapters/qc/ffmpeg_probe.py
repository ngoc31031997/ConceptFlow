"""Đo đạc cho QC (CR-021 LLD D4) — TOÀN BỘ I/O nằm ở đây.

`domain/qc_rules.py` là hàm thuần và không được biết ffmpeg tồn tại; module này
là chỗ duy nhất chạy tiến trình con để lấy số đo, rồi trả về dict thuần đưa
thẳng vào luật.

Mọi hàm ở đây **không ném**: một phép đo hỏng trả None/`{}` và luật tương ứng
im lặng bỏ qua. FR61.4 — cổng QC hỏng không được biến thành cổng khoá. Việc
phân biệt "chấm được nhưng không có finding" với "không chấm được" là của
handler, dựa trên `probe_video()['ok']`.
"""

from __future__ import annotations

import json
import logging
import os
import re
import subprocess

from adapters.assembly.ffmpeg_assembler import FFMPEG_BINARY, _probe_audio_duration

logger = logging.getLogger(__name__)

#: `-af loudnorm ... -f null -` đọc hết file một lượt. Trần thời gian riêng để
#: một file hỏng không treo consumer.
DEFAULT_QC_TIMEOUT_SECONDS = 600

LOUDNORM_MEASURE_FILTER = "loudnorm=I=-14:TP=-1.5:LRA=11:print_format=json"


def _run(cmd: list[str], timeout_seconds: int) -> subprocess.CompletedProcess | None:
    try:
        return subprocess.run(cmd, capture_output=True, text=True, timeout=timeout_seconds)
    except (OSError, subprocess.SubprocessError) as exc:
        logger.warning("QC probe %s failed: %s", cmd[0], exc)
        return None


def measure_loudness(
    video_path: str, timeout_seconds: int = DEFAULT_QC_TIMEOUT_SECONDS
) -> float | None:
    """LUFS tích hợp thật của file cuối (FR60.1).

    `loudnorm ... print_format=json` in khối JSON ra STDERR sau mọi log khác —
    lấy khối `{...}` cuối cùng trong stderr chứ không parse cả stderr.
    """
    result = _run(
        [
            FFMPEG_BINARY, "-hide_banner", "-nostats",
            "-i", video_path,
            "-af", LOUDNORM_MEASURE_FILTER,
            "-f", "null", "-",
        ],
        timeout_seconds,
    )
    if result is None:
        return None
    payload = _last_json_object(result.stderr or "")
    if payload is None:
        logger.warning("loudnorm printed no JSON for %s", video_path)
        return None
    try:
        value = float(payload["input_i"])
    except (KeyError, TypeError, ValueError):
        return None
    # loudnorm báo -inf/-70 cho file câm; không có gì để chấm.
    return None if value <= -70.0 else value


def _last_json_object(text: str) -> dict | None:
    depth = 0
    start = None
    best = None
    for index, char in enumerate(text):
        if char == "{":
            if depth == 0:
                start = index
            depth += 1
        elif char == "}" and depth > 0:
            depth -= 1
            if depth == 0 and start is not None:
                best = text[start : index + 1]
    if best is None:
        return None
    try:
        parsed = json.loads(best)
    except json.JSONDecodeError:
        return None
    return parsed if isinstance(parsed, dict) else None


_PEAK_RE = re.compile(r"Peak_level=\s*(-?[\d.]+|-?inf)", re.IGNORECASE)


def measure_peak_dbfs(
    video_path: str, timeout_seconds: int = DEFAULT_QC_TIMEOUT_SECONDS
) -> float | None:
    """Peak level (dBFS) qua `astats` (FR60.3). Lấy giá trị LỚN NHẤT trong mọi
    kênh/khối astats in ra — clipping ở một kênh vẫn là clipping."""
    result = _run(
        [
            FFMPEG_BINARY, "-hide_banner", "-nostats",
            "-i", video_path,
            "-af", "astats=metadata=1:reset=0",
            "-f", "null", "-",
        ],
        timeout_seconds,
    )
    if result is None:
        return None
    peaks = []
    for match in _PEAK_RE.finditer(result.stderr or ""):
        raw = match.group(1)
        try:
            peaks.append(float(raw))
        except ValueError:
            continue  # "-inf" — kênh câm
    return max(peaks) if peaks else None


def probe_publish_attributes(
    video_path: str, timeout_seconds: int = DEFAULT_QC_TIMEOUT_SECONDS
) -> dict:
    """Độ phân giải / framerate / pix_fmt / +faststart của file cuối (FR60.5).

    Trả `{}` khi ffprobe không đọc được — handler coi đó là không chấm được.
    """
    result = _run(
        [
            "ffprobe", "-v", "error",
            "-select_streams", "v:0",
            "-show_entries", "stream=width,height,pix_fmt,r_frame_rate",
            "-of", "json",
            video_path,
        ],
        timeout_seconds,
    )
    if result is None or result.returncode != 0:
        logger.warning("ffprobe could not read %s for QC", video_path)
        return {}
    try:
        streams = json.loads(result.stdout or "{}").get("streams") or []
    except json.JSONDecodeError:
        return {}
    if not streams:
        return {}
    stream = streams[0]

    attributes: dict = {
        "width": stream.get("width"),
        "height": stream.get("height"),
        "pix_fmt": stream.get("pix_fmt"),
        "frame_rate": _parse_rational(stream.get("r_frame_rate")),
    }
    faststart = _probe_faststart(video_path)
    if faststart is not None:
        attributes["faststart"] = faststart
    return attributes


def _parse_rational(raw: str | None) -> float | None:
    if not raw:
        return None
    numerator, _, denominator = str(raw).partition("/")
    try:
        value = float(numerator) / float(denominator or 1)
    except (ValueError, ZeroDivisionError):
        return None
    return value if value > 0 else None


def _probe_faststart(video_path: str) -> bool | None:
    """`+faststart` nghĩa là moov atom đứng TRƯỚC mdat. Đọc thẳng bảng atom ở
    đầu file thay vì gọi thêm công cụ ngoài: bốn byte độ dài + bốn byte tên,
    lặp cho tới khi gặp moov hoặc mdat."""
    try:
        with open(video_path, "rb") as handle:
            offset = 0
            size = os.path.getsize(video_path)
            while offset < size:
                handle.seek(offset)
                header = handle.read(8)
                if len(header) < 8:
                    return None
                box_size = int.from_bytes(header[:4], "big")
                box_type = header[4:8]
                if box_type == b"moov":
                    return True
                if box_type == b"mdat":
                    return False
                if box_size == 1:  # 64-bit largesize
                    extended = handle.read(8)
                    if len(extended) < 8:
                        return None
                    box_size = int.from_bytes(extended, "big")
                if box_size <= 0:
                    return None
                offset += box_size
    except OSError as exc:
        logger.warning("could not inspect atoms of %s: %s", video_path, exc)
        return None
    return None


def measure_narration_durations(
    narration_segments: list[dict], timeout_seconds: int = DEFAULT_QC_TIMEOUT_SECONDS
) -> list[dict]:
    """Gắn `duration_seconds` thật vào từng đoạn narration cho FR60.2.

    Tái dùng `_probe_audio_duration` của ffmpeg_assembler (cùng quy ước: file
    không đọc được → 0.0 kèm cảnh báo). 0.0 được bỏ đi thay vì để lại, vì một
    thời lượng bịa bằng 0 sẽ khiến luật chồng lấn im lặng bỏ qua đoạn đó — đúng
    hành vi mong muốn."""
    del timeout_seconds  # _probe_audio_duration tự quản, giữ tham số cho đối xứng
    measured = []
    for segment in narration_segments or []:
        item = dict(segment)
        duration = _probe_audio_duration(segment["audio_path"])
        if duration > 0.0:
            item["duration_seconds"] = duration
        measured.append(item)
    return measured
