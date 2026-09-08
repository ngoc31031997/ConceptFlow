"""Unit tests for the ASS subtitle writer (CR-004 FR12.5)."""

from __future__ import annotations

from adapters.assembly.subtitle_file import write_subtitle_file
from domain.models import SubtitleCue, SubtitleStyle


def read_written(tmp_path, **kwargs) -> str:
    path = str(tmp_path / "subs.ass")
    write_subtitle_file(
        [SubtitleCue(scene_index=0, text="hello", start_time=0.0, end_time=2.0)],
        SubtitleStyle(),
        path,
        **kwargs,
    )
    with open(path, encoding="utf-8") as f:
        return f.read()


def test_play_res_defaults_to_1080p(tmp_path):
    content = read_written(tmp_path)

    assert "PlayResX: 1920" in content
    assert "PlayResY: 1080" in content


def test_play_res_follows_the_actual_video(tmp_path):
    """ASS scales its layout from the declared reference resolution to the real
    frame. Declaring 1080p for a 4K video would shrink every subtitle to half
    its intended size."""
    content = read_written(tmp_path, play_res=(3840, 2160))

    assert "PlayResX: 3840" in content
    assert "PlayResY: 2160" in content


def test_font_size_scales_with_the_frame(tmp_path):
    """A "medium" subtitle should occupy the same fraction of the picture at
    any resolution, so the declared size has to scale with it."""
    at_1080 = read_written(tmp_path, play_res=(1920, 1080))
    at_2160 = read_written(tmp_path, play_res=(3840, 2160))

    size_1080 = int(at_1080.split("Style: Default,DejaVu Sans,")[1].split(",")[0])
    size_2160 = int(at_2160.split("Style: Default,DejaVu Sans,")[1].split(",")[0])

    assert size_2160 == size_1080 * 2


def test_vertical_frame_scales_by_height_not_width(tmp_path):
    """A 9:16 clip (CR-007) is narrower but taller than 1080p. Scaling off
    width would shrink the text on exactly the format that needs it largest."""
    content = read_written(tmp_path, play_res=(1080, 1920))

    size = int(content.split("Style: Default,DejaVu Sans,")[1].split(",")[0])
    baseline = int(
        read_written(tmp_path, play_res=(1920, 1080))
        .split("Style: Default,DejaVu Sans,")[1]
        .split(",")[0]
    )

    assert size > baseline


def test_cue_timings_are_preserved(tmp_path):
    content = read_written(tmp_path)

    assert "0:00:00.00,0:00:02.00" in content
    assert "hello" in content
