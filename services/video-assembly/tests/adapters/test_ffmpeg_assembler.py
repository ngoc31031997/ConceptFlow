"""Unit tests for FfmpegVideoAssembler (business-rules.md Rule 9).

Mocks subprocess.run so no real ffmpeg binary is needed to run the test
suite.
"""

from __future__ import annotations

from unittest.mock import patch

import pytest

from adapters.assembly.ffmpeg_assembler import FfmpegVideoAssembler
from domain.errors import AssemblyEngineError
from domain.models import SubtitleCue, SubtitleStyle, VideoAssemblyRequest


class FakeCompletedProcess:
    def __init__(self, returncode: int = 0, stdout: str = "", stderr: str = "") -> None:
        self.returncode = returncode
        self.stdout = stdout
        self.stderr = stderr


def test_assemble_runs_single_ffmpeg_pass_without_background_music(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        audio_segments=["a0.wav", "a1.wav"],
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", return_value=FakeCompletedProcess()) as mock_run:
        assembler.assemble(request, output_path)

    args = mock_run.call_args[0][0]
    assert args[0] == "ffmpeg"
    assert "video.mp4" in args
    assert "a0.wav" in args and "a1.wav" in args
    assert output_path in args
    assert not any("stream_loop" in a for a in args)


def test_assemble_overlays_background_music_when_present(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        audio_segments=["a0.wav"],
        background_music_path="bg.mp3",
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", return_value=FakeCompletedProcess()) as mock_run:
        assembler.assemble(request, output_path)

    args = mock_run.call_args[0][0]
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
    request = VideoAssemblyRequest(project_id="proj-1", video_path="video.mp4", audio_segments=["a0.wav"])

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
    request = VideoAssemblyRequest(project_id="proj-1", video_path="video.mp4", audio_segments=[])
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", return_value=FakeCompletedProcess()) as mock_run:
        assembler.assemble(request, output_path)

    args = mock_run.call_args[0][0]
    assert "-an" in args
    assert "concat" not in " ".join(args)


def test_assemble_without_narration_keeps_background_music(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        audio_segments=[],
        background_music_path="bg.mp3",
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", return_value=FakeCompletedProcess()) as mock_run:
        assembler.assemble(request, output_path)

    args = mock_run.call_args[0][0]
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
        audio_segments=["a0.wav"],
        subtitle_cues=[SubtitleCue(scene_index=0, text="hello", start_time=0.0, end_time=2.0)],
        subtitle_style=SubtitleStyle(font_size="large", position="top"),
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", return_value=FakeCompletedProcess()) as mock_run:
        assembler.assemble(request, output_path)

    args = mock_run.call_args[0][0]
    joined = " ".join(args)
    assert "subtitles=" in joined
    # Burned-in text means the video stream cannot be stream-copied.
    assert "copy" not in args
    assert "libx264" in args
    assert (tmp_path / "proj-1.ass").exists()
