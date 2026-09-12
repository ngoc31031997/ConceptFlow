"""Unit tests for the ASS subtitle writer (CR-004 FR12.5)."""

from __future__ import annotations

from adapters.assembly.subtitle_file import DEFAULT_PLAY_RES_X, FONT_SIZES, write_subtitle_file
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


def font_size_of(content: str) -> int:
    return int(content.split("Style: Default,DejaVu Sans,")[1].split(",")[0])


def test_vertical_frame_scales_by_width_not_height(tmp_path):
    """Bug report (screenshot): a 9:16 clip's subtitles covered nearly the
    whole picture. Root cause was scaling font size by HEIGHT: a 9:16 clip
    (1080x1920) is narrower but much TALLER than the 1920x1080 default, so
    height-based scaling inflated "medium" ~1.78x (56 -> ~99pt) while the
    frame was simultaneously narrower — every line wrapped after 1-2 words.

    Width is what actually governs how many characters fit on one ASS line
    before it wraps, which is exactly what determines whether subtitles
    overflow — so scaling by width is what keeps a vertical clip's text from
    ballooning to cover the screen. A 9:16 clip is narrower than 16:9 at the
    same declared height, so its font must come out SMALLER, not larger."""
    vertical = font_size_of(read_written(tmp_path, play_res=(1080, 1920)))
    baseline = font_size_of(read_written(tmp_path, play_res=(1920, 1080)))

    assert vertical < baseline


def test_width_based_scaling_matches_height_based_for_16_9(tmp_path):
    """Every existing (long-form, always 16:9) resolution must render
    identically after this fix — width and height scale together 1:1 for any
    16:9 frame, so switching the scale basis changes nothing for them."""
    at_720p = font_size_of(read_written(tmp_path, play_res=(1280, 720)))
    at_1080p = font_size_of(read_written(tmp_path, play_res=(1920, 1080)))

    assert at_720p == round(FONT_SIZES["medium"] * 1280 / DEFAULT_PLAY_RES_X)
    assert at_1080p == FONT_SIZES["medium"]


def test_cue_timings_are_preserved(tmp_path):
    content = read_written(tmp_path)

    assert "0:00:00.00,0:00:02.00" in content
    assert "hello" in content
