"""Unit tests for the SRT caption-track writer (CR-015 FR38)."""

from __future__ import annotations

from adapters.assembly.srt_file import write_srt_file
from domain.models import SubtitleCue


def read_written(tmp_path, cues) -> str:
    path = str(tmp_path / "captions.srt")
    write_srt_file(cues, path)
    with open(path, encoding="utf-8") as f:
        return f.read()


def test_single_cue_format(tmp_path):
    content = read_written(
        tmp_path, [SubtitleCue(scene_index=0, text="hello", start_time=0.0, end_time=2.5)]
    )

    assert content == "1\n00:00:00,000 --> 00:00:02,500\nhello\n"


def test_cues_are_numbered_in_order_given(tmp_path):
    """SRT numbering is positional, not scene_index — a caller who wants
    cues sorted has to sort them before calling this."""
    content = read_written(
        tmp_path,
        [
            SubtitleCue(scene_index=5, text="first", start_time=0.0, end_time=1.0),
            SubtitleCue(scene_index=2, text="second", start_time=1.0, end_time=2.0),
        ],
    )

    assert content.startswith("1\n")
    assert "\n2\n" in content
    assert content.index("first") < content.index("second")


def test_hour_boundary_is_padded(tmp_path):
    content = read_written(
        tmp_path, [SubtitleCue(scene_index=0, text="hi", start_time=3661.25, end_time=3662.0)]
    )

    assert "01:01:01,250" in content


def test_carries_no_styling(tmp_path):
    """SRT has no style fields — unlike the .ass writer, this module never
    takes a SubtitleStyle (ADR-0027)."""
    content = read_written(
        tmp_path, [SubtitleCue(scene_index=0, text="hello", start_time=0.0, end_time=1.0)]
    )

    assert "Style" not in content
    assert "PlayRes" not in content


def test_empty_cues_writes_empty_file(tmp_path):
    content = read_written(tmp_path, [])

    assert content == ""
