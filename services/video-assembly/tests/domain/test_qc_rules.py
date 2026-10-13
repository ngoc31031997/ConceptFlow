"""Unit tests cho từng luật QC.

Mỗi luật có ít nhất một case ĐẠT và một case KHÔNG ĐẠT, dữ liệu dựng sẵn — đúng
mục Kiểm chứng của CR. Luật là hàm thuần nên không cần file, ffmpeg hay pool.
"""

from __future__ import annotations

from domain.qc_rules import (
    BLOCKING_RULES,
    SEVERITY_BLOCKING,
    QCThresholds,
    check_clipping,
    check_loudness,
    check_narration_overlap,
    check_publish_attributes,
    check_static_frame,
    check_subtitle_cue_overlap,
    evaluate_all,
)

T = QCThresholds()


def test_narration_overlap_is_the_only_blocking_rule() -> None:
    assert BLOCKING_RULES == frozenset({"narration_overlap"})


# --- hình chết ----------------------------------------------------------------


def test_static_frame_passes_for_closely_spaced_narration() -> None:
    segments = [{"start_time": 0.0}, {"start_time": 5.0}, {"start_time": 11.0}]
    assert check_static_frame(segments, T) == []


def test_static_frame_flags_a_long_gap() -> None:
    findings = check_static_frame([{"start_time": 0.0}, {"start_time": 20.0}], T)
    assert [f.rule for f in findings] == ["static_frame"]
    assert findings[0].timestamp_seconds == 0.0


# --- chồng lấn narration -----------------------------------------------------


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


# --- cue phụ đề ---------------------------------------------------------------


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


# --- LUFS ---------------------------------------------------------------------


def test_loudness_passes_within_tolerance() -> None:
    assert check_loudness(-14.8, T) == []


def test_loudness_flags_a_quiet_master() -> None:
    findings = check_loudness(-18.0, T)
    assert [f.rule for f in findings] == ["loudness_off_target"]


def test_loudness_is_skipped_when_nothing_could_be_measured() -> None:
    assert check_loudness(None, T) == []


# --- clipping -----------------------------------------------------------------


def test_clipping_passes_below_the_ceiling() -> None:
    assert check_clipping(-1.5, T) == []


def test_clipping_flags_a_peak_at_the_ceiling() -> None:
    assert [f.rule for f in check_clipping(0.0, T)] == ["clipping"]


# --- thuộc tính phát hành ----------------------------------------------------


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
    """Cả bốn thuộc tính phát hành, cùng một file hỏng."""
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


# --- Ngưỡng đọc từ môi trường -------------------------------------------------


def test_thresholds_are_read_from_the_environment() -> None:
    thresholds = QCThresholds.from_env(
        {
            "QC_MAX_STATIC_SECONDS": "5",
            "QC_LOUDNESS_TOLERANCE_LU": "0.5",
        }
    )
    assert thresholds.max_static_seconds == 5.0
    assert thresholds.loudness_tolerance_lu == 0.5


def test_an_unparseable_threshold_falls_back_instead_of_breaking_qc() -> None:
    """Một biến gõ sai không được biến cổng QC thành cổng khoá."""
    assert QCThresholds.from_env({"QC_MAX_STATIC_SECONDS": "khong-phai-so"}).max_static_seconds == 12.0


# --- evaluate_all -------------------------------------------------------------


def test_evaluate_all_returns_nothing_for_a_clean_video() -> None:
    findings = evaluate_all(
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
        narration_segments=[
            {"start_time": 0.0, "duration_seconds": 1.0},
            {"start_time": 30.0, "duration_seconds": 1.0},
            {"start_time": 31.0, "duration_seconds": 1.0},
            {"start_time": 31.5, "duration_seconds": 1.0},
        ],
        subtitle_cues=[],
        render_quality="1080p60",
        measured_lufs=-20.0,
        peak_dbfs=None,
        publish_attributes=None,
        thresholds=T,
    )
    assert [(f.timestamp_seconds, f.rule) for f in findings] == [
        (0.0, "loudness_off_target"),
        (0.0, "static_frame"),
        (31.5, "narration_overlap"),
    ]
