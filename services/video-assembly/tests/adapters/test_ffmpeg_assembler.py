"""Unit tests for FfmpegVideoAssembler + MediaFormatInspector
(module-structure.md, business-rules.md Rule 2, 3, 9).

Mocks subprocess.run so no real ffmpeg/ffprobe binary is needed to run
the test suite (Code Generation Plan note).
"""

from __future__ import annotations

import json
from unittest.mock import patch

import pytest

from adapters.assembly.ffmpeg_assembler import FfmpegVideoAssembler
from adapters.assembly.ffprobe_inspector import MediaFormatInspector
from domain.errors import AssemblyEngineError, InconsistentMediaFormatError, InvalidSceneIndexError
from domain.models import SceneAssemblyInput, VideoAssemblyRequest


@pytest.fixture
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def _ffprobe_stdout(
    codec: str = "h264", width: int = 1920, height: int = 1080, framerate: str = "30/1"
) -> str:
    stream = {"codec_name": codec, "width": width, "height": height, "r_frame_rate": framerate}
    return json.dumps({"streams": [stream]})


class FakeCompletedProcess:
    def __init__(self, returncode: int = 0, stdout: str = "", stderr: str = "") -> None:
        self.returncode = returncode
        self.stdout = stdout
        self.stderr = stderr


def test_validate_consistent_passes_for_matching_clips():
    inspector = MediaFormatInspector()
    with patch("subprocess.run", return_value=FakeCompletedProcess(stdout=_ffprobe_stdout())):
        inspector.validate_consistent(["clip0.mp4", "clip1.mp4"])


def test_validate_consistent_raises_on_mismatch():
    inspector = MediaFormatInspector()
    responses = [
        FakeCompletedProcess(stdout=_ffprobe_stdout(width=1920, height=1080)),
        FakeCompletedProcess(stdout=_ffprobe_stdout(width=1280, height=720)),
    ]
    with patch("subprocess.run", side_effect=responses):
        with pytest.raises(InconsistentMediaFormatError):
            inspector.validate_consistent(["clip0.mp4", "clip1.mp4"])


def test_assemble_raises_invalid_scene_index_on_gap(shared_volume_root):
    assembler = FfmpegVideoAssembler(MediaFormatInspector())
    request = VideoAssemblyRequest(
        project_id="proj-1",
        scenes=[
            SceneAssemblyInput(scene_index=0, clip_path="c0.mp4", audio_path="a0.wav"),
            SceneAssemblyInput(scene_index=2, clip_path="c2.mp4", audio_path="a2.wav"),
        ],
    )

    with pytest.raises(InvalidSceneIndexError):
        assembler.assemble(request, str(shared_volume_root / "proj-1" / "video" / "final.mp4"))


def test_sorted_validated_scenes_raises_directly():
    scenes = [
        SceneAssemblyInput(scene_index=1, clip_path="c1.mp4", audio_path="a1.wav"),
        SceneAssemblyInput(scene_index=1, clip_path="c1dup.mp4", audio_path="a1dup.wav"),
    ]

    with pytest.raises(InvalidSceneIndexError):
        FfmpegVideoAssembler._sorted_validated_scenes(scenes)


def test_run_ffmpeg_raises_assembly_engine_error_on_nonzero_exit():
    with patch("subprocess.run", return_value=FakeCompletedProcess(returncode=1, stderr="boom")):
        with pytest.raises(AssemblyEngineError):
            FfmpegVideoAssembler._run_ffmpeg(["-y"])
