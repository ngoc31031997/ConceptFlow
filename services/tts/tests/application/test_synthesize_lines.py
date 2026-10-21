"""Unit tests for SynthesizeLinesUseCase (ADR-0040): script lines read outside a saga."""

from __future__ import annotations

import os
import wave

import pytest

from application.synthesize_lines import LineSpeechRequest, SynthesizeLinesUseCase
from domain.errors import TTSEngineError
from domain.ports import TTSEnginePort


def _write_silent_wav(path: str, duration_seconds: float = 2.0, framerate: int = 16000) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with wave.open(path, "wb") as wav_file:
        wav_file.setnchannels(1)
        wav_file.setsampwidth(2)
        wav_file.setframerate(framerate)
        wav_file.writeframes(b"\x00\x00" * int(duration_seconds * framerate))


class FakeEngine(TTSEnginePort):
    def __init__(self, fail_on: str | None = None) -> None:
        self.calls: list[tuple[str, str, str]] = []
        self._fail_on = fail_on

    def synthesize(self, text: str, voice_id: str, output_path: str) -> float:
        self.calls.append((text, voice_id, output_path))
        if self._fail_on is not None and self._fail_on in text:
            raise TTSEngineError("boom")
        _write_silent_wav(output_path, 3.0)
        return 3.0


@pytest.fixture
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def _line(line_id="b1.1", text="Quả sung có xác ong.", voice="vi-VN-HoaiMyNeural", ref="sha-1"):
    return LineSpeechRequest(line_id, text, "vi", voice, ref)


def test_reads_a_line_into_the_projects_audio_folder_and_echoes_the_ref(shared_volume_root):
    engine = FakeEngine()
    [result] = SynthesizeLinesUseCase(engine).execute("proj-1", [_line()])

    assert result.error == ""
    assert result.ref == "sha-1"
    assert result.duration_seconds == 3.0
    assert result.cache_hit is False
    assert result.audio_path == str(shared_volume_root / "proj-1" / "audio" / f"b1-1_{result.cache_key}.wav")
    assert os.path.isfile(result.audio_path)
    assert engine.calls == [("Quả sung có xác ong.", "vi-VN-HoaiMyNeural", result.audio_path)]


def test_the_same_wording_and_voice_is_not_read_twice(shared_volume_root):
    engine = FakeEngine()
    use_case = SynthesizeLinesUseCase(engine)
    [first] = use_case.execute("proj-1", [_line()])
    [second] = use_case.execute("proj-1", [_line()])

    assert second.cache_hit is True
    assert second.audio_path == first.audio_path
    assert len(engine.calls) == 1


def test_changed_wording_or_voice_gets_a_new_file(shared_volume_root):
    use_case = SynthesizeLinesUseCase(FakeEngine())
    [a] = use_case.execute("proj-1", [_line()])
    [b] = use_case.execute("proj-1", [_line(text="Quả sung có xác ong thật.")])
    [c] = use_case.execute("proj-1", [_line(voice="vi-VN-NamMinhNeural")])

    assert len({a.cache_key, b.cache_key, c.cache_key}) == 3
    assert len({a.audio_path, b.audio_path, c.audio_path}) == 3


def test_a_failing_line_carries_its_reason_and_the_others_are_still_read(shared_volume_root):
    lines = [_line("b1.1", "Câu hỏng"), _line("b1.2", "Câu tốt")]
    ok, bad = None, None
    results = SynthesizeLinesUseCase(FakeEngine(fail_on="hỏng")).execute("proj-1", lines)
    bad, ok = results

    assert "boom" in bad.error and bad.audio_path == ""
    assert ok.error == "" and ok.duration_seconds == 3.0


@pytest.mark.parametrize("line_id", ["", "1.1", "b1", "b1.1/../x", "../b1.1", "b0.1"])
def test_a_line_id_that_is_not_a_script_line_never_reaches_a_file_name(shared_volume_root, line_id):
    engine = FakeEngine()
    [result] = SynthesizeLinesUseCase(engine).execute("proj-1", [_line(line_id)])

    assert "mã câu" in result.error
    assert engine.calls == []


def test_empty_text_and_unsupported_language_are_per_line_errors(shared_volume_root):
    engine = FakeEngine()
    empty, other = SynthesizeLinesUseCase(engine).execute(
        "proj-1", [_line(text="  "), LineSpeechRequest("b1.2", "Bonjour", "fr", None, "r")]
    )

    assert empty.error == "câu trống"
    assert "Unsupported language" in other.error
    assert engine.calls == []


def test_stopping_marks_the_unread_lines_cancelled_instead_of_silently_skipping_them(shared_volume_root):
    engine = FakeEngine()
    stop = iter([False, True])
    results = SynthesizeLinesUseCase(engine).execute(
        "proj-1", [_line("b1.1"), _line("b1.2", "Câu hai.")], should_stop=lambda: next(stop)
    )

    assert results[0].error == ""
    assert results[1].error == "cancelled"
    assert len(engine.calls) == 1


def test_a_project_id_that_could_leave_the_shared_folder_is_refused(shared_volume_root):
    with pytest.raises(ValueError):
        SynthesizeLinesUseCase(FakeEngine()).execute("../escape", [_line()])
