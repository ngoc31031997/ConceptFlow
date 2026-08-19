"""Unit tests for SynthesizeSpeechBatchUseCase (ADR-0014 — fail-fast batch,
mirrors ClassifyScenesBatchUseCase, Unit 2)."""

from __future__ import annotations

import os
import wave

import pytest

from application.synthesize_speech import SynthesizeSpeechUseCase
from application.synthesize_speech_batch import (
    BatchSynthesisFailure,
    BatchSynthesisSuccess,
    SceneSpeechRequest,
    SynthesizeSpeechBatchUseCase,
)
from domain.errors import TTSEngineError
from domain.ports import TTSEnginePort


def _write_silent_wav(path: str, duration_seconds: float = 2.0, framerate: int = 16000) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    n_frames = int(duration_seconds * framerate)
    with wave.open(path, "wb") as wav_file:
        wav_file.setnchannels(1)
        wav_file.setsampwidth(2)
        wav_file.setframerate(framerate)
        wav_file.writeframes(b"\x00\x00" * n_frames)


class FakeTTSEngine(TTSEnginePort):
    def __init__(self, fail_at_language: str | None = None) -> None:
        self.calls: list[tuple[str, str, str]] = []
        self._fail_at_language = fail_at_language

    def synthesize(self, text: str, language: str, output_path: str) -> float:
        if language == self._fail_at_language:
            raise TTSEngineError("engine crashed")
        self.calls.append((text, language, output_path))
        _write_silent_wav(output_path, duration_seconds=3.0)
        return 3.0


@pytest.fixture
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def test_batch_synthesizes_every_scene(shared_volume_root):
    engine = FakeTTSEngine()
    batch_use_case = SynthesizeSpeechBatchUseCase(SynthesizeSpeechUseCase(engine))
    scenes = [
        SceneSpeechRequest(scene_index=0, narration_text="hello", language="en"),
        SceneSpeechRequest(scene_index=1, narration_text="xin chao", language="vi"),
    ]

    outcome = batch_use_case.execute("proj-1", scenes)

    assert isinstance(outcome, BatchSynthesisSuccess)
    assert [r.scene_index for r in outcome.results] == [0, 1]
    assert all(r.duration_seconds == 3.0 for r in outcome.results)


def test_batch_fails_fast_on_first_error(shared_volume_root):
    engine = FakeTTSEngine(fail_at_language="vi")
    batch_use_case = SynthesizeSpeechBatchUseCase(SynthesizeSpeechUseCase(engine))
    scenes = [
        SceneSpeechRequest(scene_index=0, narration_text="hello", language="en"),
        SceneSpeechRequest(scene_index=1, narration_text="xin chao", language="vi"),
        SceneSpeechRequest(scene_index=2, narration_text="never reached", language="en"),
    ]

    outcome = batch_use_case.execute("proj-1", scenes)

    assert isinstance(outcome, BatchSynthesisFailure)
    assert "tts_engine_failure" in outcome.error_message
    assert len(engine.calls) == 1  # scene 0 succeeded, scene 1 failed, scene 2 never attempted


def test_batch_reports_empty_text_error(shared_volume_root):
    engine = FakeTTSEngine()
    batch_use_case = SynthesizeSpeechBatchUseCase(SynthesizeSpeechUseCase(engine))
    scenes = [SceneSpeechRequest(scene_index=0, narration_text="   ", language="en")]

    outcome = batch_use_case.execute("proj-1", scenes)

    assert isinstance(outcome, BatchSynthesisFailure)
    assert "empty_text" in outcome.error_message


def test_batch_reports_unsupported_language_error(shared_volume_root):
    engine = FakeTTSEngine()
    batch_use_case = SynthesizeSpeechBatchUseCase(SynthesizeSpeechUseCase(engine))
    scenes = [SceneSpeechRequest(scene_index=0, narration_text="hello", language="fr")]

    outcome = batch_use_case.execute("proj-1", scenes)

    assert isinstance(outcome, BatchSynthesisFailure)
    assert "unsupported_language" in outcome.error_message
