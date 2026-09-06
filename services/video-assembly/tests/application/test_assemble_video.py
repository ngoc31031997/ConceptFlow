"""Unit tests for AssembleVideoUseCase (business-rules.md Rule 1, 8)."""

from __future__ import annotations

import os

import pytest

from application.assemble_video import AssembleVideoUseCase
from domain.errors import MissingArtifactError
from domain.models import VideoAssemblyRequest
from domain.ports import VideoAssemblerPort


def _touch(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write("data")


class FakeVideoAssembler(VideoAssemblerPort):
    """Records assemble() calls and writes a placeholder output file."""

    def __init__(self) -> None:
        self.calls: list[VideoAssemblyRequest] = []

    def assemble(self, request: VideoAssemblyRequest, output_path: str) -> None:
        self.calls.append(request)
        _touch(output_path)


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
    request = VideoAssemblyRequest(project_id="proj-1", video_path=video_path, audio_segments=[audio_path])

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
    request = VideoAssemblyRequest(project_id="proj-1", video_path=video_path, audio_segments=[audio_path])

    first = use_case.assemble(request)
    second = use_case.assemble(request)

    assert len(assembler.calls) == 1
    assert second.video_path == first.video_path


def test_empty_audio_segments_assembles_a_silent_video(shared_volume_root):
    # CR-001: narration is optional, so no audio segments is a valid request
    # rather than a missing artifact.
    video_path = str(shared_volume_root / "rendered.mp4")
    _touch(video_path)
    assembler = FakeVideoAssembler()
    use_case = AssembleVideoUseCase(assembler)
    request = VideoAssemblyRequest(project_id="proj-1", video_path=video_path, audio_segments=[])

    result = use_case.assemble(request)

    assert len(assembler.calls) == 1
    assert result.video_path.endswith(".mp4")


def test_missing_video_file_raises_missing_artifact_error(shared_volume_root):
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(audio_path)
    use_case = AssembleVideoUseCase(FakeVideoAssembler())
    request = VideoAssemblyRequest(
        project_id="proj-1", video_path=str(shared_volume_root / "missing.mp4"), audio_segments=[audio_path]
    )

    with pytest.raises(MissingArtifactError):
        use_case.assemble(request)


def test_missing_audio_segment_raises_missing_artifact_error(shared_volume_root):
    video_path = str(shared_volume_root / "rendered.mp4")
    _touch(video_path)
    use_case = AssembleVideoUseCase(FakeVideoAssembler())
    request = VideoAssemblyRequest(
        project_id="proj-1", video_path=video_path, audio_segments=[str(shared_volume_root / "missing.wav")]
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
        audio_segments=[audio_path],
        background_music_path=str(shared_volume_root / "missing_bg.mp3"),
    )

    with pytest.raises(MissingArtifactError):
        use_case.assemble(request)
