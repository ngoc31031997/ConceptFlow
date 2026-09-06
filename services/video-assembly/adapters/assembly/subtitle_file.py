"""Renders subtitle cues into an ASS file for ffmpeg to burn in (CR-001 FR9.2/FR9.4).

ASS rather than SRT because the Creator chooses the appearance (size, colour,
background box, position) and SRT carries no styling — with SRT those choices
would have to be re-expressed as ffmpeg's `force_style`, which is the same ASS
style syntax anyway, just harder to inspect when a video looks wrong.
"""

from __future__ import annotations

import os

from domain.models import SubtitleCue, SubtitleStyle

# ASS renders at a fixed reference resolution and scales to the real frame, so
# these font sizes stay proportional regardless of the rendered video size.
PLAY_RES_X = 1920
PLAY_RES_Y = 1080

FONT_SIZES = {"small": 42, "medium": 56, "large": 72}

# ASS alignment codes (numpad layout): 2 = bottom-centre, 8 = top-centre.
ALIGNMENT = {"bottom": 2, "top": 8}

MARGIN_VERTICAL = 60


def write_subtitle_file(cues: list[SubtitleCue], style: SubtitleStyle, path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(_render(cues, style))


def _render(cues: list[SubtitleCue], style: SubtitleStyle) -> str:
    font_size = FONT_SIZES.get(style.font_size, FONT_SIZES["medium"])
    alignment = ALIGNMENT.get(style.position, ALIGNMENT["bottom"])
    primary = _to_ass_colour(style.text_color, opacity=1.0)
    back = _to_ass_colour("#000000", opacity=style.background_opacity)
    # BorderStyle 3 draws an opaque box behind the text; with a fully
    # transparent background colour it renders as plain text on the video.
    border_style = 3 if style.background_opacity > 0 else 1

    header = f"""[Script Info]
ScriptType: v4.00+
PlayResX: {PLAY_RES_X}
PlayResY: {PLAY_RES_Y}
WrapStyle: 0

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, OutlineColour, BackColour, Bold, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,DejaVu Sans,{font_size},{primary},&H00000000,{back},0,{border_style},2,0,{alignment},80,80,{MARGIN_VERTICAL},1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
"""

    lines = [
        f"Dialogue: 0,{_to_ass_time(cue.start_time)},{_to_ass_time(cue.end_time)},Default,,0,0,0,,{_escape(cue.text)}"
        for cue in cues
    ]
    return header + "\n".join(lines) + "\n"


def _to_ass_colour(hex_colour: str, opacity: float) -> str:
    """ASS colours are &HAABBGGRR — alpha first, then blue/green/red, and the
    alpha byte is inverted (00 is fully opaque, FF fully transparent)."""
    value = hex_colour.lstrip("#")
    if len(value) != 6:
        value = "FFFFFF"
    red, green, blue = value[0:2], value[2:4], value[4:6]
    alpha = round((1.0 - max(0.0, min(1.0, opacity))) * 255)
    return f"&H{alpha:02X}{blue}{green}{red}".upper()


def _to_ass_time(seconds: float) -> str:
    """ASS timestamps are H:MM:SS.cc — centiseconds, single-digit hour."""
    if seconds < 0:
        seconds = 0.0
    hours, remainder = divmod(seconds, 3600)
    minutes, secs = divmod(remainder, 60)
    centiseconds = round((secs - int(secs)) * 100)
    if centiseconds == 100:
        secs += 1
        centiseconds = 0
    return f"{int(hours)}:{int(minutes):02d}:{int(secs):02d}.{centiseconds:02d}"


def _escape(text: str) -> str:
    """Newlines become ASS line breaks; braces would otherwise open an
    override block and swallow the rest of the line."""
    return text.replace("\\", "\\\\").replace("{", "(").replace("}", ")").replace("\n", "\\N")
