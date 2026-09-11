"""Unit tests cho từng luật QC (CR-021 FR59/FR60).

Mỗi luật có ít nhất một case ĐẠT và một case KHÔNG ĐẠT, dữ liệu dựng sẵn — đúng
mục Kiểm chứng của CR. Luật là hàm thuần nên không cần file, ffmpeg hay pool.
"""

from __future__ import annotations

import pytest

from domain.qc_rules import (
    FRAME_HEIGHT,
    FRAME_WIDTH,
    SEVERITY_BLOCKING,
    SEVERITY_WARNING,
    QCThresholds,
    check_clipping,
    check_frame_overflow,
    check_loudness,
    check_low_contrast,
    check_narration_overlap,
    check_publish_attributes,
    check_static_frame,
    check_subtitle_cue_overlap,
    check_text_overlap,
    check_text_too_small,
    contrast_ratio,
    evaluate_all,
    font_size_to_pixels,
)

T = QCThresholds()


def mark(*mobjects, t: float = 12.5, index: int = 0) -> dict:
    return {"index": index, "t": t, "mobjects": list(mobjects)}


def text(
    left: float,
    right: float,
    top: float,
    bottom: float,
    *,
    cls: str = "Text",
    color: str = "#F2F7FF",
    font_size: float | None = 36.0,
    **extra,
) -> dict:
    return {"cls": cls, "bbox": [left, right, top, bottom], "color": color,
            "font_size": font_size, **extra}


# --- Khung hình đúng như Manim ------------------------------------------------


def test_frame_constants_match_manim_defaults() -> None:
    """Manim 0.18: frame_height 8.0, tỉ lệ 16:9 → frame_width 14.222…"""
    assert FRAME_HEIGHT == 8.0
    assert FRAME_WIDTH == pytest.approx(8.0 * 16 / 9)


# --- FR59.1 tràn khung --------------------------------------------------------


def test_frame_overflow_passes_for_a_box_inside_the_safe_area() -> None:
    assert check_frame_overflow([mark(text(-3.0, 3.0, 1.0, 0.0))], T) == []


def test_frame_overflow_flags_a_box_outside_the_manim_frame() -> None:
    findings = check_frame_overflow([mark(text(-9.0, -7.5, 1.0, 0.0))], T)
    assert [f.rule for f in findings] == ["frame_overflow"]
    assert "tràn ra ngoài khung" in findings[0].message
    assert findings[0].timestamp_seconds == 12.5  # FR59.6


def test_frame_overflow_flags_a_box_intruding_on_the_safe_margin() -> None:
    """Bên trong khung nhưng lấn vào safe margin 0.6 của theme."""
    findings = check_frame_overflow([mark(text(-6.9, -6.7, 1.0, 0.0))], T)
    assert len(findings) == 1
    assert "safe margin" in findings[0].message


def test_frame_overflow_ignores_transparent_mobjects() -> None:
    """Đã FadeOut nhưng chưa remove — bbox còn, mắt không thấy (LLD Rủi ro)."""
    faded = text(-9.0, -7.5, 1.0, 0.0, opacity=0.0)
    assert check_frame_overflow([mark(faded)], T) == []


def test_frame_overflow_is_always_reported_as_blocking() -> None:
    """Quyết định #1: tràn khung là blocking. Severity không đổi theo chế độ —
    chế độ chỉ-báo tồn tại để ĐẾM xem luật này kêu đúng bao nhiêu, nên nó phải
    nhìn thấy được severity thật trong báo cáo (LLD D5)."""
    boxes = [mark(text(-9.0, -7.5, 1.0, 0.0))]
    assert check_frame_overflow(boxes, QCThresholds())[0].severity == SEVERITY_BLOCKING


# --- FR59.2 chồng lấn ---------------------------------------------------------


def test_text_overlap_passes_for_boxes_that_do_not_touch() -> None:
    scene = mark(text(-4.0, -1.0, 1.0, 0.0), text(1.0, 4.0, 1.0, 0.0))
    assert check_text_overlap([scene], T) == []


def test_text_overlap_flags_two_overlapping_text_mobjects() -> None:
    scene = mark(text(-2.0, 1.0, 1.0, 0.0), text(0.0, 3.0, 1.0, 0.0))
    findings = check_text_overlap([scene], T)
    assert [f.rule for f in findings] == ["text_overlap"]


def test_text_overlap_ignores_vgroups() -> None:
    """Bbox của VGroup bao trùm con nó, nên nhóm nào cũng 'chồng' con mình."""
    scene = mark(
        {"cls": "VGroup", "bbox": [-2.0, 2.0, 1.0, 0.0], "color": None, "font_size": None},
        text(-2.0, 2.0, 1.0, 0.0),
    )
    assert check_text_overlap([scene], T) == []


def test_text_overlap_ignores_transparent_text() -> None:
    scene = mark(text(-2.0, 1.0, 1.0, 0.0, opacity=0.0), text(0.0, 3.0, 1.0, 0.0))
    assert check_text_overlap([scene], T) == []


def test_text_overlap_tolerates_a_brushing_touch_below_the_ratio() -> None:
    """Chồng ~3% diện tích — dưới ngưỡng 15%, không kêu."""
    scene = mark(text(-2.0, 1.0, 1.0, 0.0), text(0.9, 3.9, 1.0, 0.0))
    assert check_text_overlap([scene], T) == []


# --- FR59.3 chữ quá nhỏ -------------------------------------------------------


def test_font_size_to_pixels_matches_the_manim_frame_mapping() -> None:
    # 1080 dòng / 8 đơn vị = 135 px mỗi đơn vị; font_size 20 = 0.20 đơn vị.
    assert font_size_to_pixels(20.0, "1080p60", T) == pytest.approx(27.0)


def test_text_too_small_passes_for_the_theme_caption_size_at_1080p() -> None:
    assert check_text_too_small([mark(text(-1, 1, 1, 0, font_size=20.0))], "1080p60", T) == []


def test_text_too_small_flags_tiny_text() -> None:
    findings = check_text_too_small([mark(text(-1, 1, 1, 0, font_size=12.0))], "1080p60", T)
    assert [f.rule for f in findings] == ["text_too_small"]


def test_text_too_small_accounts_for_the_export_resolution() -> None:
    """Cùng cỡ chữ: đạt ở 1080p, không đạt ở 720p (FR59.3 nói rõ 'ở độ phân
    giải xuất')."""
    marks = [mark(text(-1, 1, 1, 0, font_size=20.0))]
    assert check_text_too_small(marks, "1080p60", T) == []
    assert len(check_text_too_small(marks, "720p30", T)) == 1


def test_text_too_small_skips_mobjects_without_a_font_size() -> None:
    assert check_text_too_small([mark(text(-1, 1, 1, 0, font_size=None))], "1080p60", T) == []


# --- FR59.4 tương phản --------------------------------------------------------


def test_contrast_ratio_extremes() -> None:
    assert contrast_ratio("#FFFFFF", "#000000") == pytest.approx(21.0)
    assert contrast_ratio("#080E1C", "#080E1C") == pytest.approx(1.0)


def test_low_contrast_passes_for_theme_ink_on_theme_background() -> None:
    assert check_low_contrast([mark(text(-1, 1, 1, 0, color="#F2F7FF"))], T) == []


def test_low_contrast_flags_dark_text_on_the_dark_background() -> None:
    findings = check_low_contrast([mark(text(-1, 1, 1, 0, color="#12182A"))], T)
    assert [f.rule for f in findings] == ["low_contrast"]


def test_low_contrast_skips_an_unparseable_colour() -> None:
    assert check_low_contrast([mark(text(-1, 1, 1, 0, color=None))], T) == []


# --- FR59.5 hình chết ---------------------------------------------------------


def test_static_frame_passes_for_closely_spaced_narration() -> None:
    segments = [{"start_time": 0.0}, {"start_time": 5.0}, {"start_time": 11.0}]
    assert check_static_frame(segments, T) == []


def test_static_frame_flags_a_long_gap() -> None:
    findings = check_static_frame([{"start_time": 0.0}, {"start_time": 20.0}], T)
    assert [f.rule for f in findings] == ["static_frame"]
    assert findings[0].timestamp_seconds == 0.0


# --- FR60.2 chồng lấn narration ----------------------------------------------


def test_narration_overlap_passes_when_each_clip_ends_before_the_next_starts() -> None:
    segments = [
        {"start_time": 0.0, "duration_seconds": 4.0},
        {"start_time": 5.0, "duration_seconds": 3.0},
    ]
    assert check_narration_overlap(segments, T) == []


def test_narration_overlap_flags_a_clip_running_into_the_next() -> None:
    segments = [
        {"start_time": 0.0, "duration_seconds": 6.5},
        {"start_time": 5.0, "duration_seconds": 3.0},
    ]
    findings = check_narration_overlap(segments, T)
    assert [f.rule for f in findings] == ["narration_overlap"]
    assert findings[0].timestamp_seconds == 5.0


def test_narration_overlap_skips_segments_with_no_measured_duration() -> None:
    segments = [{"start_time": 0.0}, {"start_time": 1.0, "duration_seconds": 3.0}]
    assert check_narration_overlap(segments, T) == []


def test_narration_overlap_is_always_reported_as_blocking() -> None:
    segments = [
        {"start_time": 0.0, "duration_seconds": 6.5},
        {"start_time": 5.0, "duration_seconds": 3.0},
    ]
    assert check_narration_overlap(segments, T)[0].severity == SEVERITY_BLOCKING


# --- FR60.4 cue phụ đề --------------------------------------------------------


def test_subtitle_cue_overlap_passes_for_sequential_cues() -> None:
    cues = [
        {"scene_index": 0, "start_time": 0.0, "end_time": 2.0},
        {"scene_index": 0, "start_time": 2.0, "end_time": 4.0},
    ]
    assert check_subtitle_cue_overlap(cues, T) == []


def test_subtitle_cue_overlap_flags_overlapping_cues() -> None:
    cues = [
        {"scene_index": 0, "start_time": 0.0, "end_time": 3.0},
        {"scene_index": 1, "start_time": 2.0, "end_time": 4.0},
    ]
    findings = check_subtitle_cue_overlap(cues, T)
    assert [f.rule for f in findings] == ["subtitle_cue_overlap"]
    assert findings[0].timestamp_seconds == 2.0


# --- FR60.1 LUFS --------------------------------------------------------------


def test_loudness_passes_within_tolerance() -> None:
    assert check_loudness(-14.8, T) == []


def test_loudness_flags_a_quiet_master() -> None:
    findings = check_loudness(-18.0, T)
    assert [f.rule for f in findings] == ["loudness_off_target"]


def test_loudness_is_skipped_when_nothing_could_be_measured() -> None:
    assert check_loudness(None, T) == []


# --- FR60.3 clipping ----------------------------------------------------------


def test_clipping_passes_below_the_ceiling() -> None:
    assert check_clipping(-1.5, T) == []


def test_clipping_flags_a_peak_at_the_ceiling() -> None:
    assert [f.rule for f in check_clipping(0.0, T)] == ["clipping"]


# --- FR60.5 thuộc tính phát hành ---------------------------------------------


GOOD_ATTRIBUTES = {
    "width": 1920,
    "height": 1080,
    "frame_rate": 60.0,
    "pix_fmt": "yuv420p",
    "faststart": True,
}


def test_publish_attributes_pass_for_a_correctly_encoded_file() -> None:
    assert check_publish_attributes(GOOD_ATTRIBUTES, "1080p60", T) == []


def test_publish_attributes_flag_wrong_resolution_framerate_pixfmt_and_faststart() -> None:
    """Cả bốn thuộc tính FR60.5 liệt kê, cùng một file hỏng."""
    bad = {
        "width": 1280,
        "height": 720,
        "frame_rate": 30.0,
        "pix_fmt": "yuv444p",
        "faststart": False,
    }
    findings = check_publish_attributes(bad, "1080p60", T)
    assert len(findings) == 4
    assert {f.rule for f in findings} == {"publish_attributes"}


def test_publish_attributes_are_skipped_when_ffprobe_read_nothing() -> None:
    assert check_publish_attributes({}, "1080p60", T) == []


# --- Ngưỡng đọc từ môi trường (FR61.5) ---------------------------------------


def test_thresholds_are_read_from_the_environment() -> None:
    thresholds = QCThresholds.from_env(
        {
            "QC_MIN_TEXT_PIXEL_HEIGHT": "40",
            "QC_MAX_STATIC_SECONDS": "5",
            "QC_LOUDNESS_TOLERANCE_LU": "0.5",
        }
    )
    assert thresholds.min_text_pixel_height == 40.0
    assert thresholds.max_static_seconds == 5.0
    assert thresholds.loudness_tolerance_lu == 0.5


def test_an_unparseable_threshold_falls_back_instead_of_breaking_qc() -> None:
    """FR61.4: một biến gõ sai không được biến cổng QC thành cổng khoá."""
    assert QCThresholds.from_env({"QC_MAX_STATIC_SECONDS": "khong-phai-so"}).max_static_seconds == 12.0


# --- evaluate_all -------------------------------------------------------------


def test_evaluate_all_returns_nothing_for_a_clean_video() -> None:
    findings = evaluate_all(
        layout_marks=[mark(text(-3.0, -0.5, 1.0, 0.0), text(0.5, 3.0, 1.0, 0.0))],
        narration_segments=[
            {"start_time": 0.0, "duration_seconds": 4.0},
            {"start_time": 5.0, "duration_seconds": 3.0},
        ],
        subtitle_cues=[{"scene_index": 0, "start_time": 0.0, "end_time": 2.0}],
        render_quality="1080p60",
        measured_lufs=-14.2,
        peak_dbfs=-1.2,
        publish_attributes=GOOD_ATTRIBUTES,
        thresholds=T,
    )
    assert findings == []


def test_evaluate_all_sorts_findings_along_the_timeline() -> None:
    findings = evaluate_all(
        layout_marks=[
            mark(text(-9.0, -7.5, 1.0, 0.0), t=30.0),
            mark(text(-9.0, -7.5, 1.0, 0.0), t=4.0),
        ],
        narration_segments=[],
        subtitle_cues=[],
        render_quality="1080p60",
        measured_lufs=-20.0,
        peak_dbfs=None,
        publish_attributes=None,
        thresholds=T,
    )
    assert [f.timestamp_seconds for f in findings] == [0.0, 4.0, 30.0]
    assert findings[0].rule == "loudness_off_target"
