"""Unit tests for FfmpegVideoAssembler (business-rules.md Rule 9).

Mocks subprocess.run so no real ffmpeg binary is needed to run the test
suite.
"""

from __future__ import annotations

from unittest.mock import patch

import pytest

from adapters.assembly.ffmpeg_assembler import FfmpegVideoAssembler
from domain.errors import AssemblyEngineError
from domain.models import (
    NarrationSegment,
    SubtitleCue,
    SubtitleStyle,
    VideoAssemblyRequest,
)


class FakeCompletedProcess:
    def __init__(self, returncode: int = 0, stdout: str = "", stderr: str = "") -> None:
        self.returncode = returncode
        self.stdout = stdout
        self.stderr = stderr


def fake_run_factory(audio_duration: str = "5.0"):
    """subprocess.run stand-in that answers ffprobe with a duration and every
    other call (i.e. ffmpeg) with success."""

    def fake_run(cmd, *_args, **_kwargs):
        if cmd and cmd[0] == "ffprobe":
            return FakeCompletedProcess(stdout=f"{audio_duration}\n")
        return FakeCompletedProcess()

    return fake_run


def ffmpeg_args(mock_run):
    """The ffmpeg invocation, ignoring any ffprobe calls made alongside it."""
    for call in mock_run.call_args_list:
        cmd = call[0][0]
        if cmd and cmd[0] == "ffmpeg":
            return cmd
    raise AssertionError("ffmpeg was never invoked")


def test_assemble_runs_single_ffmpeg_pass_without_background_music(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[
            NarrationSegment(audio_path="a0.wav", start_time=0.0),
            NarrationSegment(audio_path="a1.wav", start_time=12.5),
        ],
        video_duration_seconds=30.0,
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, output_path)

    args = ffmpeg_args(mock_run)
    assert args[0] == "ffmpeg"
    assert "video.mp4" in args
    assert "a0.wav" in args and "a1.wav" in args
    assert output_path in args
    assert not any("stream_loop" in a for a in args)


def test_each_narration_segment_is_delayed_to_its_own_offset(tmp_path):
    """CR-002 core regression: segments must be placed at their measured
    offsets, never concatenated end to end."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[
            NarrationSegment(audio_path="a0.wav", start_time=0.0),
            NarrationSegment(audio_path="a1.wav", start_time=12.5),
            NarrationSegment(audio_path="a2.wav", start_time=41.25),
        ],
        video_duration_seconds=60.0,
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    filter_complex = ffmpeg_args(mock_run)[
        ffmpeg_args(mock_run).index("-filter_complex") + 1
    ]
    assert "adelay=0:all=1" in filter_complex
    assert "adelay=12500:all=1" in filter_complex
    assert "adelay=41250:all=1" in filter_complex
    # normalize=0 or a 3-segment narration comes out a third of its volume.
    assert "amix=inputs=3:normalize=0" in filter_complex
    assert "concat" not in filter_complex


def test_segments_are_placed_by_offset_not_input_order(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[
            NarrationSegment(audio_path="late.wav", start_time=30.0),
            NarrationSegment(audio_path="early.wav", start_time=1.0),
        ],
        video_duration_seconds=60.0,
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    args = ffmpeg_args(mock_run)
    inputs = [args[i + 1] for i, a in enumerate(args) if a == "-i"]
    assert inputs == ["video.mp4", "early.wav", "late.wav"]


def test_video_is_padded_when_narration_outlasts_the_animation(tmp_path):
    """CR-002 FR10.6: hold the last frame rather than cutting the closing line."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=18.0)],
        video_duration_seconds=20.0,
    )

    # A 5s clip starting at 18s ends at 23s, i.e. 3s past the 20s video.
    with patch("subprocess.run", side_effect=fake_run_factory("5.0")) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    args = ffmpeg_args(mock_run)
    joined = " ".join(args)
    assert "tpad=stop_mode=clone:stop_duration=3.000" in joined
    assert args[args.index("-t") + 1] == "23.000"
    # -shortest is what used to truncate the closing narration.
    assert "-shortest" not in args


def test_video_is_not_padded_when_narration_fits(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=2.0)],
        video_duration_seconds=60.0,
    )

    with patch("subprocess.run", side_effect=fake_run_factory("5.0")) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    args = ffmpeg_args(mock_run)
    assert "tpad" not in " ".join(args)
    assert args[args.index("-t") + 1] == "60.000"
    # Nothing repaints the picture, so it can still be stream-copied.
    assert "copy" in args


def test_falls_back_to_shortest_without_a_known_video_duration(tmp_path):
    """A pre-CR-002 project reports no duration; keep the old behaviour rather
    than guessing a target length."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=0.0,
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    args = ffmpeg_args(mock_run)
    assert "-shortest" in args
    assert "-t" not in args


def test_assemble_overlays_background_music_when_present(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=30.0,
        background_music_path="bg.mp3",
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, output_path)

    args = ffmpeg_args(mock_run)
    assert "bg.mp3" in args
    assert "-stream_loop" in args
    filter_complex = args[args.index("-filter_complex") + 1]
    assert "amix" in filter_complex


def test_run_ffmpeg_raises_assembly_engine_error_on_nonzero_exit():
    with patch("subprocess.run", return_value=FakeCompletedProcess(returncode=1, stderr="boom")):
        with pytest.raises(AssemblyEngineError):
            FfmpegVideoAssembler._run_ffmpeg(["-y"])


def test_assemble_wraps_timeout_as_assembly_engine_error(tmp_path):
    assembler = FfmpegVideoAssembler(timeout_seconds=0)
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
    )

    import time

    def slow_run(*_args, **_kwargs):
        time.sleep(0.5)
        return FakeCompletedProcess()

    with patch("subprocess.run", side_effect=slow_run):
        with pytest.raises(AssemblyEngineError, match="timed out"):
            assembler.assemble(request, str(tmp_path / "final.mp4"))


def test_assemble_without_narration_produces_a_silent_video(tmp_path):
    # CR-001: no audio_segments means nothing to mux — the output must be
    # explicitly silent rather than inheriting a stray stream.
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1", video_path="video.mp4", narration_segments=[]
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, output_path)

    args = ffmpeg_args(mock_run)
    assert "-an" in args
    assert "concat" not in " ".join(args)


def test_assemble_without_narration_keeps_background_music(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[],
        background_music_path="bg.mp3",
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, output_path)

    args = ffmpeg_args(mock_run)
    assert "-an" not in args
    maps = [args[i + 1] for i, a in enumerate(args) if a == "-map"]
    assert maps[-1] == "[bg]"
    # With no narration there is nothing to mix the music against.
    assert "amix" not in " ".join(args)


def test_assemble_with_subtitles_burns_them_in_and_reencodes(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=30.0,
        subtitle_cues=[SubtitleCue(scene_index=0, text="hello", start_time=0.0, end_time=2.0)],
        subtitle_style=SubtitleStyle(font_size="large", position="top"),
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, output_path)

    args = ffmpeg_args(mock_run)
    joined = " ".join(args)
    assert "subtitles=" in joined
    # Burned-in text means the video stream cannot be stream-copied.
    assert "copy" not in args
    assert "libx264" in args
    assert (tmp_path / "proj-1.ass").exists()
