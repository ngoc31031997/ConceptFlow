"""Renders subtitle cues into a plain SRT file, for upload as a YouTube
caption track rather than being burned into the video frames (CR-015 FR38).

Deliberately carries no `SubtitleStyle` — SRT has no styling fields, and
YouTube decides how the track is displayed. That is exactly why the burn-in
path (subtitle_file.py, ASS) still exists alongside this one rather than
being replaced by it (ADR-0027).

Cues passed in here MUST already be shifted by any lead-in the caller adds
(SubtitleCue.shifted_by) — this module never touches a timestamp, on
purpose: a serializer that could shift time is a serializer that could
shift it differently from the other one.
"""

from __future__ import annotations

import os

from domain.models import SubtitleCue


def write_srt_file(cues: list[SubtitleCue], path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(_render(cues))


def _render(cues: list[SubtitleCue]) -> str:
    blocks = [
        f"{index}\n{_to_srt_time(cue.start_time)} --> {_to_srt_time(cue.end_time)}\n{cue.text}\n"
        for index, cue in enumerate(cues, start=1)
    ]
    return "\n".join(blocks)


def _to_srt_time(seconds: float) -> str:
    """SRT timestamps are HH:MM:SS,mmm — comma decimal separator, unlike ASS's
    period, and milliseconds rather than centiseconds."""
    if seconds < 0:
        seconds = 0.0
    hours, remainder = divmod(seconds, 3600)
    minutes, secs = divmod(remainder, 60)
    milliseconds = round((secs - int(secs)) * 1000)
    if milliseconds == 1000:
        secs += 1
        milliseconds = 0
    return f"{int(hours):02d}:{int(minutes):02d}:{int(secs):02d},{milliseconds:03d}"
