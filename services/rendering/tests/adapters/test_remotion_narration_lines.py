"""Narration lines grouped into shots: one TTS clip per line, several lines per shot."""

from __future__ import annotations

import pytest

from adapters.rendering.remotion_renderer import (
    FPS,
    _extract_shot_line_counts,
    _segments_from,
)
from domain.errors import AnimationEngineError
from domain.models import NarrationSegment


def _lines(*seconds: float) -> list[NarrationSegment]:
    return [NarrationSegment(scene_index=i, duration_seconds=s) for i, s in enumerate(seconds)]


class TestExtractShotLineCounts:
    def test_reads_the_exported_counts(self):
        script = (
            "export const narrations: string[] = ['a', 'b', 'c'];\n"
            "export const shotLineCounts: number[] = [2, 1];"
        )
        assert _extract_shot_line_counts(script) == [2, 1]

    def test_a_script_without_counts_has_none(self):
        assert _extract_shot_line_counts("export const narrations = ['a'];") is None

    def test_ignores_counts_left_in_a_comment(self):
        assert _extract_shot_line_counts("// export const shotLineCounts = [9];\nconst x = 1;") is None

    @pytest.mark.parametrize("value", ["0", "1.5", "two"])
    def test_rejects_anything_but_whole_numbers_from_one(self, value):
        with pytest.raises(AnimationEngineError, match="whole numbers"):
            _extract_shot_line_counts(f"export const shotLineCounts: number[] = [1, {value}];")


class TestSegmentsFrom:
    def test_without_counts_every_line_is_a_shot_as_before(self):
        segments, offsets = _segments_from(_lines(2.0, 3.0))
        assert segments == [
            {"startFrame": 0, "durationInFrames": 60},
            {"startFrame": 69, "durationInFrames": 90},
        ]
        assert offsets == [0.0, 2.3]

    def test_lines_of_a_shot_are_laid_end_to_end_with_a_short_breath(self):
        # shot 1: 2.0 s + 0.1 s breath + 1.0 s; 0.3 s between shots; shot 2: 1.5 s.
        segments, offsets = _segments_from(_lines(2.0, 1.0, 1.5), [2, 1])
        assert segments == [
            {"startFrame": 0, "durationInFrames": 93, "lines": [0, 63]},
            {"startFrame": 102, "durationInFrames": 45, "lines": [0]},
        ]
        assert offsets == [0.0, 63 / FPS, 102 / FPS]

    def test_one_wait_offset_per_line_and_each_at_its_lines_frame(self):
        segments, offsets = _segments_from(_lines(1.0, 1.0, 1.0, 1.0), [3, 1])
        assert len(offsets) == 4
        first = segments[0]
        assert [round((first["startFrame"] + start) / FPS, 6) for start in first["lines"]] == [
            round(o, 6) for o in offsets[:3]
        ]

    def test_counts_that_do_not_add_up_to_the_lines_are_refused(self):
        with pytest.raises(AnimationEngineError, match="adds up to 3 lines but the script has 2"):
            _segments_from(_lines(1.0, 1.0), [2, 1])
