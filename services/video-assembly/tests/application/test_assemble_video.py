"""Unit tests for AssembleVideoUseCase (business-rules.md Rule 1, 8)."""

from __future__ import annotations

import os

import pytest

from application.assemble_video import AssembleVideoUseCase
from domain.errors import MissingArtifactError
from domain.models import NarrationSegment, VideoAssemblyRequest
from domain.ports import VideoAssemblerPort


def _touch(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write("data")


class FakeVideoAssembler(VideoAssemblerPort):
    """Records assemble() calls and writes a placeholder output file."""

    def __init__(self, caption_path: str | None = None) -> None:
        self.calls: list[VideoAssemblyRequest] = []
        self._caption_path = caption_path

    def assemble(self, request: VideoAssemblyRequest, output_path: str) -> str | None:
        self.calls.append(request)
        _touch(output_path)
        if self._caption_path:
            _touch(self._caption_path)
        return self._caption_path


@pytest.fixture
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def test_assembles_video_and_returns_result(shared_volume_root):
    video_path = str(shared_volume_root / "rendered.mp4")
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(video_path)
    _touch(audio_path)
    assembler = FakeVideoAssembler()
    use_case = AssembleVideoUseCase(assembler)
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path=video_path,
        narration_segments=[NarrationSegment(audio_path=audio_path, start_time=0.0)],
    )

    result = use_case.assemble(request)

    assert result.video_path == str(shared_volume_root / "proj-1" / "video" / "final.mp4")
    assert len(assembler.calls) == 1


def test_idempotent_call_does_not_reassemble(shared_volume_root):
    video_path = str(shared_volume_root / "rendered.mp4")
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(video_path)
    _touch(audio_path)
    assembler = FakeVideoAssembler()
    use_case = AssembleVideoUseCase(assembler)
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path=video_path,
        narration_segments=[NarrationSegment(audio_path=audio_path, start_time=0.0)],
    )

    first = use_case.assemble(request)
    second = use_case.assemble(request)

    assert len(assembler.calls) == 1
    assert second.video_path == first.video_path


def test_empty_narration_segments_assembles_a_silent_video(shared_volume_root):
    # CR-001: narration is optional, so no audio segments is a valid request
    # rather than a missing artifact.
    video_path = str(shared_volume_root / "rendered.mp4")
    _touch(video_path)
    assembler = FakeVideoAssembler()
    use_case = AssembleVideoUseCase(assembler)
    request = VideoAssemblyRequest(project_id="proj-1", video_path=video_path, narration_segments=[])

    result = use_case.assemble(request)

    assert len(assembler.calls) == 1
    assert result.video_path.endswith(".mp4")


def test_missing_video_file_raises_missing_artifact_error(shared_volume_root):
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(audio_path)
    use_case = AssembleVideoUseCase(FakeVideoAssembler())
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path=str(shared_volume_root / "missing.mp4"),
        narration_segments=[NarrationSegment(audio_path=audio_path, start_time=0.0)],
    )

    with pytest.raises(MissingArtifactError):
        use_case.assemble(request)


def test_missing_narration_audio_raises_missing_artifact_error(shared_volume_root):
    video_path = str(shared_volume_root / "rendered.mp4")
    _touch(video_path)
    use_case = AssembleVideoUseCase(FakeVideoAssembler())
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path=video_path,
        narration_segments=[
            NarrationSegment(audio_path=str(shared_volume_root / "missing.wav"), start_time=0.0)
        ],
    )

    with pytest.raises(MissingArtifactError):
        use_case.assemble(request)


def test_missing_background_music_raises_missing_artifact_error(shared_volume_root):
    video_path = str(shared_volume_root / "rendered.mp4")
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(video_path)
    _touch(audio_path)
    use_case = AssembleVideoUseCase(FakeVideoAssembler())
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path=video_path,
        narration_segments=[NarrationSegment(audio_path=audio_path, start_time=0.0)],
        background_music_path=str(shared_volume_root / "missing_bg.mp3"),
    )

    with pytest.raises(MissingArtifactError):
        use_case.assemble(request)


def test_caption_path_flows_from_assembler_into_the_result(shared_volume_root):
    """CR-015 FR38.4: caption_path travels the same way thumbnail_path does —
    through the result, not rediscovered by the caller."""
    video_path = str(shared_volume_root / "rendered.mp4")
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(video_path)
    _touch(audio_path)
    expected_caption_path = str(shared_volume_root / "proj-1" / "video" / "final.srt")
    assembler = FakeVideoAssembler(caption_path=expected_caption_path)
    use_case = AssembleVideoUseCase(assembler)
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path=video_path,
        narration_segments=[NarrationSegment(audio_path=audio_path, start_time=0.0)],
        subtitle_mode="track",
    )

    result = use_case.assemble(request)

    assert result.caption_path == expected_caption_path


def test_no_caption_path_when_the_assembler_produced_none(shared_volume_root):
    video_path = str(shared_volume_root / "rendered.mp4")
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(video_path)
    _touch(audio_path)
    use_case = AssembleVideoUseCase(FakeVideoAssembler(caption_path=None))
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path=video_path,
        narration_segments=[NarrationSegment(audio_path=audio_path, start_time=0.0)],
    )

    result = use_case.assemble(request)

    assert result.caption_path is None


def test_idempotent_call_reuses_caption_file_already_on_disk(shared_volume_root):
    """Mirrors the video-file idempotency check (Business Rule 8): a caption
    file left over from a prior call is reused rather than silently dropped,
    and its absence is not silently invented either."""
    video_path = str(shared_volume_root / "rendered.mp4")
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(video_path)
    _touch(audio_path)
    assembler = FakeVideoAssembler(caption_path=str(shared_volume_root / "proj-1" / "video" / "final.srt"))
    use_case = AssembleVideoUseCase(assembler)
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path=video_path,
        narration_segments=[NarrationSegment(audio_path=audio_path, start_time=0.0)],
        subtitle_mode="track",
    )
    first = use_case.assemble(request)

    second = use_case.assemble(request)

    assert len(assembler.calls) == 1
    assert second.caption_path == first.caption_path


def test_unknown_subtitle_mode_raises_missing_artifact_error(shared_volume_root):
    video_path = str(shared_volume_root / "rendered.mp4")
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(video_path)
    _touch(audio_path)
    use_case = AssembleVideoUseCase(FakeVideoAssembler())
    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path=video_path,
        narration_segments=[NarrationSegment(audio_path=audio_path, start_time=0.0)],
        subtitle_mode="bogus",
    )

    with pytest.raises(MissingArtifactError, match="subtitle_mode"):
        use_case.assemble(request)


def test_negative_start_time_raises_missing_artifact_error(shared_volume_root):
    """Zero-trust: a negative offset would make ffmpeg's adelay silently drop
    the segment, so it is rejected here rather than producing a video missing
    one line of narration."""
    video_path = str(shared_volume_root / "rendered.mp4")
    audio_path = str(shared_volume_root / "a0.wav")
    _touch(video_path)
    _touch(audio_path)
    use_case = AssembleVideoUseCase(FakeVideoAssembler())

    request = VideoAssemblyRequest(
        project_id="proj-1",
        video_path=video_path,
        narration_segments=[NarrationSegment(audio_path=audio_path, start_time=-1.0)],
    )
    with pytest.raises(MissingArtifactError, match="negative start_time"):
        use_case.assemble(request)
