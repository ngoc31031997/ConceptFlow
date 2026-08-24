"""Unit tests for AssembleVideoUseCase (business-rules.md Rule 1, 8)."""

from __future__ import annotations

import os

import pytest

from application.assemble_video import AssembleVideoUseCase
from domain.errors import MissingArtifactError
from domain.models import SceneAssemblyInput, VideoAssemblyRequest
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


def _scene(index: int, clip: str, audio: str) -> SceneAssemblyInput:
    return SceneAssemblyInput(scene_index=index, clip_path=clip, audio_path=audio)


def test_assembles_video_and_returns_result(shared_volume_root):
    clip_path = str(shared_volume_root / "clip0.mp4")
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(clip_path)
    _touch(audio_path)
    assembler = FakeVideoAssembler()
    use_case = AssembleVideoUseCase(assembler)
    request = VideoAssemblyRequest(project_id="proj-1", scenes=[_scene(0, clip_path, audio_path)])

    result = use_case.assemble(request)

    assert result.video_path == str(shared_volume_root / "proj-1" / "video" / "final.mp4")
    assert len(assembler.calls) == 1


def test_idempotent_call_does_not_reassemble(shared_volume_root):
    clip_path = str(shared_volume_root / "clip0.mp4")
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(clip_path)
    _touch(audio_path)
    assembler = FakeVideoAssembler()
    use_case = AssembleVideoUseCase(assembler)
    request = VideoAssemblyRequest(project_id="proj-1", scenes=[_scene(0, clip_path, audio_path)])

    first = use_case.assemble(request)
    second = use_case.assemble(request)

    assert len(assembler.calls) == 1
    assert second.video_path == first.video_path


def test_empty_scenes_raises_missing_artifact_error(shared_volume_root):
    use_case = AssembleVideoUseCase(FakeVideoAssembler())
    request = VideoAssemblyRequest(project_id="proj-1", scenes=[])

    with pytest.raises(MissingArtifactError):
        use_case.assemble(request)


def test_missing_clip_file_raises_missing_artifact_error(shared_volume_root):
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(audio_path)
    use_case = AssembleVideoUseCase(FakeVideoAssembler())
    request = VideoAssemblyRequest(
        project_id="proj-1", scenes=[_scene(0, str(shared_volume_root / "missing.mp4"), audio_path)]
    )

    with pytest.raises(MissingArtifactError):
        use_case.assemble(request)


def test_missing_background_music_raises_missing_artifact_error(shared_volume_root):
    clip_path = str(shared_volume_root / "clip0.mp4")
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(clip_path)
    _touch(audio_path)
    use_case = AssembleVideoUseCase(FakeVideoAssembler())
    request = VideoAssemblyRequest(
        project_id="proj-1",
        scenes=[_scene(0, clip_path, audio_path)],
        background_music_path=str(shared_volume_root / "missing_bg.mp3"),
    )

    with pytest.raises(MissingArtifactError):
        use_case.assemble(request)
