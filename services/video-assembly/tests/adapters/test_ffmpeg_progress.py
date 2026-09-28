"""Merge progress from ffmpeg's own `-progress` stream, against a real ffmpeg.

The rest of the assembler suite mocks subprocess.run; this reads a live child's
stdout line by line, which only a real binary exercises. Skipped without ffmpeg.
"""

from __future__ import annotations

import shutil

import pytest

from adapters.assembly.ffmpeg_assembler import FfmpegVideoAssembler
from domain.errors import AssemblyEngineError

pytestmark = pytest.mark.skipif(shutil.which("ffmpeg") is None, reason="needs a real ffmpeg binary")


def _encode_args(out_path: str, seconds: int) -> list[str]:
    return [
        "-y", "-f", "lavfi", "-i", f"testsrc=duration={seconds}:size=320x240:rate=30",
        "-c:v", "libx264", "-preset", "ultrafast", out_path,
    ]


def test_reports_rising_percentages_held_below_the_share(tmp_path):
    percents: list[int] = []
    FfmpegVideoAssembler._run_ffmpeg_with_progress(
        _encode_args(str(tmp_path / "out.mp4"), 3),
        expected_seconds=3.0,
        share=90,
        on_progress=percents.append,
    )

    assert percents, "ffmpeg -progress must yield at least one percentage"
    assert percents == sorted(set(percents)), "percentages only ever rise, one call per change"
    # The share itself is reported by the caller only after a successful exit.
    assert max(percents) <= 89


def test_a_failing_ffmpeg_still_raises_with_its_stderr(tmp_path):
    with pytest.raises(AssemblyEngineError, match="ffmpeg exited"):
        FfmpegVideoAssembler._run_ffmpeg_with_progress(
            ["-y", "-i", str(tmp_path / "missing.mp4"), str(tmp_path / "out.mp4")],
            expected_seconds=3.0, share=100, on_progress=lambda _p: None,
        )
