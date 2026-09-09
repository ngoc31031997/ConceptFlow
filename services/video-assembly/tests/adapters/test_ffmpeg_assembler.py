"""Unit tests for FfmpegVideoAssembler (business-rules.md Rule 9).

Mocks subprocess.run so no real ffmpeg binary is needed to run the test
suite.
"""

from __future__ import annotations

import re
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


def fake_run_factory(audio_duration: str = "5.0", width=1920, height=1080, fps="60/1"):
    """subprocess.run stand-in answering each ffprobe query and letting every
    ffmpeg call succeed.

    The queries are told apart by what they select, since the assembler asks
    ffprobe for a duration, a resolution and a frame rate.
    """

    def fake_run(cmd, *_args, **_kwargs):
        if not cmd or cmd[0] != "ffprobe":
            return FakeCompletedProcess()
        entries = cmd[cmd.index("-show_entries") + 1]
        if "width" in entries:
            return FakeCompletedProcess(stdout=f"{width},{height}\n")
        if "r_frame_rate" in entries:
            return FakeCompletedProcess(stdout=f"{fps}\n")
        return FakeCompletedProcess(stdout=f"{audio_duration}\n")

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
    joined = " ".join(args)
    # With no narration there is nothing to mix or duck the music against...
    assert "amix" not in joined
    assert "sidechaincompress" not in joined
    # ...but a music-only track still gets normalised to YouTube's target.
    maps = [args[i + 1] for i, a in enumerate(args) if a == "-map"]
    assert maps[-1] == "[anorm]"
    assert "[bg]loudnorm" in joined


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
    assert not (tmp_path / "proj-1.srt").exists()


def test_subtitle_mode_track_writes_only_srt_and_stream_copies(tmp_path):
    """CR-015 FR38 / ADR-0027: 'track' delivers a caption file for Publisher
    to upload, and does NOT paint anything into the video — so, unlike
    burn-in, the video stream stays stream-copyable."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=30.0,
        subtitle_cues=[SubtitleCue(scene_index=0, text="hello", start_time=0.0, end_time=2.0)],
        subtitle_mode="track",
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        caption_path = assembler.assemble(request, output_path)

    args = ffmpeg_args(mock_run)
    joined = " ".join(args)
    assert "subtitles=" not in joined
    assert "copy" in args
    assert not (tmp_path / "proj-1.ass").exists()
    assert (tmp_path / "proj-1.srt").exists()
    assert caption_path == str(tmp_path / "proj-1.srt")


def test_subtitle_mode_both_writes_both_files(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=30.0,
        subtitle_cues=[SubtitleCue(scene_index=0, text="hello", start_time=0.0, end_time=2.0)],
        subtitle_mode="both",
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        caption_path = assembler.assemble(request, output_path)

    args = ffmpeg_args(mock_run)
    assert "subtitles=" in " ".join(args)
    assert (tmp_path / "proj-1.ass").exists()
    assert (tmp_path / "proj-1.srt").exists()
    assert caption_path == str(tmp_path / "proj-1.srt")


def test_subtitle_mode_off_ignores_cues_entirely(tmp_path):
    """A cue list left over from before the Creator turned subtitles off
    must not produce either file (CR-001 FR9.1 still holds)."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=30.0,
        subtitle_cues=[SubtitleCue(scene_index=0, text="hello", start_time=0.0, end_time=2.0)],
        subtitle_mode="off",
    )
    output_path = str(tmp_path / "final.mp4")

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        caption_path = assembler.assemble(request, output_path)

    args = ffmpeg_args(mock_run)
    assert "subtitles=" not in " ".join(args)
    assert caption_path is None
    assert not (tmp_path / "proj-1.ass").exists()
    assert not (tmp_path / "proj-1.srt").exists()


def test_lead_in_shifts_the_caption_track_too(tmp_path):
    """Same guarantee as the .ass case (test_lead_in_shifts_picture_audio_and_
    subtitles_together below), but for the .srt path: both serializers must
    receive the same already-shifted cues (ADR-0027)."""
    assembler = FfmpegVideoAssembler(lead_in_seconds=0.5, tail_seconds=1.0)
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=10.0)],
        video_duration_seconds=60.0,
        subtitle_cues=[SubtitleCue(scene_index=0, text="hi", start_time=10.0, end_time=12.0)],
        subtitle_mode="track",
    )

    with patch("subprocess.run", side_effect=fake_run_factory()):
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    with open(tmp_path / "proj-1.srt", encoding="utf-8") as f:
        assert "00:00:10,500" in f.read()


def test_encoded_output_uses_upload_grade_settings(tmp_path):
    """CR-004 FR12.2. YouTube transcodes whatever it receives, so the source has
    to carry spare quality into that second encode — and yuv420p/+faststart are
    correctness, not polish: without them some players reject the file or must
    download it whole before playing."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=30.0,
        subtitle_cues=[SubtitleCue(scene_index=0, text="hi", start_time=0.0, end_time=2.0)],
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    args = ffmpeg_args(mock_run)
    for expected in ["libx264", "slow", "18", "yuv420p", "high", "+faststart", "aac", "192k", "48000"]:
        assert expected in args, f"{expected} missing from {args}"
    # Keyframe every 2s at the probed 60fps.
    assert args[args.index("-g") + 1] == "120"


def test_keyframe_interval_follows_the_videos_frame_rate(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=30.0,
        subtitle_cues=[SubtitleCue(scene_index=0, text="hi", start_time=0.0, end_time=2.0)],
    )

    with patch("subprocess.run", side_effect=fake_run_factory(fps="30/1")) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    args = ffmpeg_args(mock_run)
    assert args[args.index("-g") + 1] == "60"


def test_stream_copy_skips_the_encode_settings_entirely(tmp_path):
    """FR12.3. Phase 0 measured -c:v copy at 0.1s against 24.8s for a
    re-encode, so not re-encoding when nothing repaints the picture is the
    single highest-value change here."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=60.0,
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    args = ffmpeg_args(mock_run)
    assert "copy" in args
    assert "libx264" not in args
    assert "-g" not in args
    # The container flag still applies — faststart is layout, not codec.
    assert "+faststart" in args


def test_silent_video_gets_no_audio_encode_args(tmp_path):
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1", video_path="video.mp4", narration_segments=[]
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    args = ffmpeg_args(mock_run)
    assert "-an" in args
    assert "aac" not in args


def test_background_music_is_ducked_against_the_narration(tmp_path):
    """CR-005 FR14.1. A flat mix leaves music fighting the voice when it is
    loud and leaves silence when it is not."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=30.0,
        background_music_path="bg.mp3",
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    filter_complex = ffmpeg_args(mock_run)[ffmpeg_args(mock_run).index("-filter_complex") + 1]
    assert "sidechaincompress" in filter_complex
    # The narration is the sidechain input, so the music ducks under the voice.
    # It reaches the compressor through asplit, because the same narration also
    # has to reach the final mix and ffmpeg consumes each label only once.
    assert "[narration]asplit=2[navoice][nakey]" in filter_complex
    assert "[bg][nakey]sidechaincompress" in filter_complex
    assert "[navoice][ducked]amix=inputs=2" in filter_complex


def test_every_filtergraph_label_is_consumed_exactly_once(tmp_path):
    """ffmpeg reads a second use of a label as an input stream specifier and
    aborts with "matches no streams", so reusing one breaks assembly outright."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[
            NarrationSegment(audio_path="a0.wav", start_time=0.0),
            NarrationSegment(audio_path="a1.wav", start_time=12.5),
        ],
        video_duration_seconds=30.0,
        background_music_path="bg.mp3",
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    filter_complex = ffmpeg_args(mock_run)[ffmpeg_args(mock_run).index("-filter_complex") + 1]
    produced: list[str] = []
    consumed: list[str] = []
    label = r"\[([A-Za-z_][A-Za-z0-9_]*)\]"
    for step in filter_complex.split(";"):
        head = re.match(r"^((?:\[[^\]]+\])*)", step).group(1)
        tail = re.search(r"((?:\[[^\]]+\])*)$", step).group(1)
        # "[0:v]" and friends name real input streams, not graph labels; the
        # pattern skips them, so only named links are counted.
        consumed += re.findall(label, head)
        produced += re.findall(label, tail)

    assert sorted(produced) == sorted(set(produced)), "a label is produced twice"
    for label in consumed:
        assert consumed.count(label) == 1, f"label [{label}] is consumed twice"
        assert label in produced, f"label [{label}] is consumed but never produced"


def test_music_volume_is_configurable(tmp_path):
    """FR14.2 — 0.2 was hardcoded."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[],
        background_music_path="bg.mp3",
        background_music_volume=0.35,
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    filter_complex = ffmpeg_args(mock_run)[ffmpeg_args(mock_run).index("-filter_complex") + 1]
    assert "volume=0.35" in filter_complex


def test_audio_is_normalised_to_youtubes_target(tmp_path):
    """FR14.3. YouTube normalizes to about -14 LUFS, so a quieter master is not
    left quiet — it is turned up along with its noise floor."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=30.0,
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    filter_complex = ffmpeg_args(mock_run)[ffmpeg_args(mock_run).index("-filter_complex") + 1]
    assert "loudnorm=I=-14" in filter_complex


def test_silent_video_is_not_normalised(tmp_path):
    """There is no audio stream to normalise, and asking ffmpeg to build one
    would just add an empty track."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1", video_path="video.mp4", narration_segments=[]
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    assert "loudnorm" not in " ".join(ffmpeg_args(mock_run))


def test_padding_is_off_by_default_so_stream_copy_survives(tmp_path):
    """FR14.5's trade-off, asserted so it is not silently reversed: padding
    either end means tpad, tpad repaints frames, and that rules out -c:v copy —
    measured at 250x faster in Phase 0."""
    assembler = FfmpegVideoAssembler()
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=1.0)],
        video_duration_seconds=60.0,
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    args = ffmpeg_args(mock_run)
    assert "tpad" not in " ".join(args)
    assert "copy" in args


def test_lead_in_shifts_picture_audio_and_subtitles_together(tmp_path):
    """The lead-in must not reintroduce CR-002's desync: narration i still has
    to land on wait i, just later in the file. Shifting only the audio would
    put every line ahead of its animation by the lead-in."""
    assembler = FfmpegVideoAssembler(lead_in_seconds=0.5, tail_seconds=1.0)
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=10.0)],
        video_duration_seconds=60.0,
        subtitle_cues=[SubtitleCue(scene_index=0, text="hi", start_time=10.0, end_time=12.0)],
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(tmp_path / "final.mp4"))

    args = ffmpeg_args(mock_run)
    joined = " ".join(args)
    # Audio delayed by 10.0 + 0.5.
    assert "adelay=10500:all=1" in joined
    # Picture delayed by the same 0.5 at the head.
    assert "tpad=start_mode=clone:start_duration=0.500" in joined
    # And the subtitle moved with them.
    with open(tmp_path / "proj-1.ass", encoding="utf-8") as f:
        assert "0:00:10.50" in f.read()
    # 60s video + 0.5 lead-in + 1.0 tail.
    assert args[args.index("-t") + 1] == "61.500"


def ffmpeg_calls(mock_run):
    return [c[0][0] for c in mock_run.call_args_list if c[0][0] and c[0][0][0] == "ffmpeg"]


def test_a_thumbnail_candidate_is_extracted(tmp_path):
    """CR-006 FR16.1. No text is burned in: the title does not exist yet at
    assembly time, so anything overlaid here would be guesswork."""
    assembler = FfmpegVideoAssembler()
    video_dir = tmp_path / "proj-1" / "video"
    video_dir.mkdir(parents=True)
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=200.0,
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(video_dir / "final.mp4"))

    thumbnail_call = ffmpeg_calls(mock_run)[-1]
    joined = " ".join(thumbnail_call)
    assert "1280:720" in joined
    assert "-frames:v" in thumbnail_call
    # A quarter of the way in — the opening frames of a Manim video are a title
    # card or an empty stage.
    assert thumbnail_call[thumbnail_call.index("-ss") + 1] == "50.000"
    assert thumbnail_call[-1].endswith("thumbnail/auto.jpg")


def test_thumbnail_failure_does_not_fail_the_assembly(tmp_path):
    """A missing thumbnail is a small inconvenience; losing an assembled video
    over one would not be."""
    assembler = FfmpegVideoAssembler()
    video_dir = tmp_path / "proj-1" / "video"
    video_dir.mkdir(parents=True)
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=200.0,
    )

    calls = {"n": 0}

    def fail_on_thumbnail(cmd, *args, **kwargs):
        if cmd and cmd[0] == "ffmpeg":
            calls["n"] += 1
            if calls["n"] > 1:  # the thumbnail pass
                return FakeCompletedProcess(returncode=1, stderr="boom")
        return fake_run_factory()(cmd, *args, **kwargs)

    with patch("subprocess.run", side_effect=fail_on_thumbnail):
        assembler.assemble(request, str(video_dir / "final.mp4"))


def test_no_thumbnail_without_a_known_duration(tmp_path):
    """Without a duration there is no sensible frame to pick."""
    assembler = FfmpegVideoAssembler()
    video_dir = tmp_path / "proj-1" / "video"
    video_dir.mkdir(parents=True)
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path="video.mp4",
        narration_segments=[NarrationSegment(audio_path="a0.wav", start_time=0.0)],
        video_duration_seconds=0.0,
    )

    with patch("subprocess.run", side_effect=fake_run_factory()) as mock_run:
        assembler.assemble(request, str(video_dir / "final.mp4"))

    assert len(ffmpeg_calls(mock_run)) == 1
