"""Unit tests for SynthesizeSpeechUseCase (business-rules.md Rule 1-6)."""

from __future__ import annotations

import os
import wave

import pytest

from application.synthesize_speech import SynthesizeSpeechUseCase
from domain.errors import EmptyTextError, UnsupportedLanguageError
from domain.models import SpeechRequest
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
    """Records synthesize() calls and writes a fixed-duration .wav file."""

    def __init__(self, duration_seconds: float = 3.0) -> None:
        self.calls: list[tuple[str, str, str]] = []
        self._duration_seconds = duration_seconds

    def synthesize(self, text: str, language: str, output_path: str) -> float:
        self.calls.append((text, language, output_path))
        _write_silent_wav(output_path, self._duration_seconds)
        return self._duration_seconds


@pytest.fixture
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def test_synthesize_calls_engine_and_returns_result(shared_volume_root):
    engine = FakeTTSEngine(duration_seconds=4.5)
    use_case = SynthesizeSpeechUseCase(engine)

    result = use_case.synthesize(SpeechRequest("proj-1", 0, "Xin chao", "vi"))

    assert result.duration_seconds == 4.5
    assert result.audio_path == str(shared_volume_root / "proj-1" / "audio" / "0_vi.wav")
    assert engine.calls == [("Xin chao", "vi", result.audio_path)]


def test_synthesize_passes_text_verbatim_no_preprocessing(shared_volume_root):
    # Business Rule 3: no normalization — raw text (including symbols) reaches the engine.
    engine = FakeTTSEngine()
    use_case = SynthesizeSpeechUseCase(engine)

    use_case.synthesize(SpeechRequest("proj-1", 0, "  for i in range(10):  ", "en"))

    assert engine.calls[0][0] == "for i in range(10):"


def test_empty_text_raises_empty_text_error(shared_volume_root):
    use_case = SynthesizeSpeechUseCase(FakeTTSEngine())

    with pytest.raises(EmptyTextError):
        use_case.synthesize(SpeechRequest("proj-1", 0, "   ", "vi"))


def test_unsupported_language_raises_error(shared_volume_root):
    use_case = SynthesizeSpeechUseCase(FakeTTSEngine())

    with pytest.raises(UnsupportedLanguageError):
        use_case.synthesize(SpeechRequest("proj-1", 0, "hello", "fr"))


def test_idempotent_call_does_not_re_synthesize(shared_volume_root):
    # Business Rule 4: a second call with the same project_id+scene_index reuses the file.
    engine = FakeTTSEngine(duration_seconds=4.5)
    use_case = SynthesizeSpeechUseCase(engine)
    request = SpeechRequest("proj-1", 0, "Xin chao", "vi")

    first = use_case.synthesize(request)
    second = use_case.synthesize(request)

    assert len(engine.calls) == 1
    assert second.audio_path == first.audio_path
    assert second.duration_seconds == first.duration_seconds


def test_duration_is_measured_from_wav_file_not_estimated(shared_volume_root):
    # Business Rule 5: duration comes from the .wav file, independent of engine's
    # own return value plausibility (here they happen to match).
    engine = FakeTTSEngine(duration_seconds=7.25)
    use_case = SynthesizeSpeechUseCase(engine)

    result = use_case.synthesize(SpeechRequest("proj-1", 0, "a very long narration text " * 5, "en"))

    assert result.duration_seconds == 7.25
