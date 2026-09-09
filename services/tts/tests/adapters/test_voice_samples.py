"""Unit tests for the startup preview clips (CR-001 FR4.5, CR-011 follow-up)."""

from __future__ import annotations

import json

import pytest

from adapters.tts_engines import voice_registry, voice_samples
from adapters.tts_engines.voice_registry import ENGINE_AZURE, ENGINE_EDGE


class RecordingEngine:
    """Records what it was asked to synthesize and writes a stand-in file."""

    def __init__(self) -> None:
        self.calls: list[str] = []

    def synthesize(self, text: str, voice_id: str, output_path: str) -> float:
        self.calls.append(voice_id)
        with open(output_path, "wb") as f:
            f.write(b"RIFF")
        return 1.0


@pytest.fixture
def shared(tmp_path, monkeypatch):
    monkeypatch.setattr(voice_registry, "SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path / "voice_samples"


def test_only_offered_voices_get_a_preview(shared):
    engine = RecordingEngine()

    voice_samples.generate_missing_samples(engine, {ENGINE_EDGE})

    assert engine.calls == [
        "vi-VN-HoaiMyNeural",
        "vi-VN-NamMinhNeural",
        "en-US-JennyNeural",
        "en-US-GuyNeural",
    ]
    catalog = json.loads((shared / "catalog.json").read_text(encoding="utf-8"))
    assert {entry["engine"] for entry in catalog} == {ENGINE_EDGE}


def test_previews_for_no_longer_offered_voices_are_deleted(shared):
    """The clip on disk for an unconfigured Azure voice was made by the Edge
    fallback. Keeping it would mean that adding AZURE_SPEECH_KEY later never
    regenerates it — the Azure preview would play Edge audio for good. The same
    sweep clears the Piper clips ADR-0024 left behind."""
    shared.mkdir(parents=True)
    (shared / "azure:vi-VN-HoaiMyNeural.wav").write_bytes(b"edge audio")
    (shared / "vi_VN-vais1000-medium.wav").write_bytes(b"piper audio")

    voice_samples.generate_missing_samples(RecordingEngine(), {ENGINE_EDGE})

    assert not (shared / "azure:vi-VN-HoaiMyNeural.wav").exists()
    assert not (shared / "vi_VN-vais1000-medium.wav").exists()
    assert (shared / "vi-VN-HoaiMyNeural.wav").exists()


def test_a_configured_engine_keeps_its_previews(shared):
    engine = RecordingEngine()

    voice_samples.generate_missing_samples(engine, {ENGINE_EDGE, ENGINE_AZURE})

    assert "azure:vi-VN-HoaiMyNeural" in engine.calls
    assert (shared / "azure:vi-VN-HoaiMyNeural.wav").exists()
    assert (shared / "catalog.json").exists()


def test_an_existing_preview_is_not_regenerated(shared):
    voice_samples.generate_missing_samples(RecordingEngine(), {ENGINE_EDGE})
    second = RecordingEngine()

    voice_samples.generate_missing_samples(second, {ENGINE_EDGE})

    assert second.calls == []


def test_a_failing_voice_does_not_stop_the_others(shared):
    class HalfBrokenEngine(RecordingEngine):
        def synthesize(self, text: str, voice_id: str, output_path: str) -> float:
            if voice_id == "vi-VN-HoaiMyNeural":
                raise RuntimeError("engine refused")
            return super().synthesize(text, voice_id, output_path)

    engine = HalfBrokenEngine()

    voice_samples.generate_missing_samples(engine, {ENGINE_EDGE})

    assert (shared / "vi-VN-NamMinhNeural.wav").exists()
