"""Unit tests for RoutingTTSEngine (CR-005 FR13.3 / FR13.5, ADR-0023)."""

from __future__ import annotations

import json

import pytest

from adapters.tts_engines.routing_engine import FREE_TIER_CHARACTERS, RoutingTTSEngine
from domain.errors import TTSEngineError


class RecordingEngine:
    def __init__(self, fail: bool = False, duration: float = 2.0) -> None:
        self.fail = fail
        self.duration = duration
        self.calls: list[tuple[str, str, str]] = []

    def synthesize(self, text: str, voice_id: str, output_path: str) -> float:
        self.calls.append((text, voice_id, output_path))
        if self.fail:
            raise TTSEngineError("simulated engine failure")
        return self.duration


def make_engine(tmp_path, google=None, piper=None):
    return RoutingTTSEngine(
        piper=piper or RecordingEngine(),
        google=google,
        usage_path=str(tmp_path / "usage.json"),
    )


def test_google_voice_goes_to_google(tmp_path):
    google = RecordingEngine(duration=3.0)
    piper = RecordingEngine()
    engine = make_engine(tmp_path, google=google, piper=piper)

    duration = engine.synthesize("xin chao", "vi-VN-Wavenet-A", "/tmp/out.wav")

    assert duration == 3.0
    assert len(google.calls) == 1
    assert piper.calls == []


def test_piper_voice_bypasses_google_entirely(tmp_path):
    google = RecordingEngine()
    piper = RecordingEngine(duration=1.5)
    engine = make_engine(tmp_path, google=google, piper=piper)

    engine.synthesize("xin chao", "vi_VN-vais1000-medium", "/tmp/out.wav")

    assert google.calls == []
    assert len(piper.calls) == 1


def test_missing_credentials_fall_back_within_the_same_language(tmp_path):
    """FR13.3. A Vietnamese project must not suddenly speak English because the
    cloud engine was unavailable."""
    piper = RecordingEngine()
    engine = make_engine(tmp_path, google=None, piper=piper)

    engine.synthesize("xin chao", "vi-VN-Wavenet-A", "/tmp/out.wav")

    assert piper.calls[0][1] == "vi_VN-vais1000-medium"


def test_google_failure_falls_back_to_piper(tmp_path):
    """A network blip or expired credential should cost quality, not the whole
    render the Creator is waiting on."""
    google = RecordingEngine(fail=True)
    piper = RecordingEngine(duration=1.0)
    engine = make_engine(tmp_path, google=google, piper=piper)

    duration = engine.synthesize("hello", "en-US-Wavenet-F", "/tmp/out.wav")

    assert duration == 1.0
    assert piper.calls[0][1] == "en_US-lessac-medium"


def test_fallback_is_logged_not_silent(tmp_path, caplog):
    """Silently shipping lower-quality narration is worse than the failure —
    the Creator has to know before they publish."""
    engine = make_engine(tmp_path, google=RecordingEngine(fail=True))

    with caplog.at_level("WARNING"):
        engine.synthesize("hello", "en-US-Wavenet-F", "/tmp/out.wav")

    assert any("falling back" in record.message.lower() for record in caplog.records)


def test_usage_is_metered_only_for_the_billed_engine(tmp_path):
    usage_path = tmp_path / "usage.json"
    engine = RoutingTTSEngine(
        piper=RecordingEngine(), google=RecordingEngine(), usage_path=str(usage_path)
    )

    engine.synthesize("12345", "en-US-Wavenet-F", "/tmp/a.wav")
    engine.synthesize("ignored-by-meter", "en_US-lessac-medium", "/tmp/b.wav")

    recorded = json.loads(usage_path.read_text())
    assert sum(recorded.values()) == 5


def test_usage_accumulates_across_calls(tmp_path):
    usage_path = tmp_path / "usage.json"
    engine = RoutingTTSEngine(
        piper=RecordingEngine(), google=RecordingEngine(), usage_path=str(usage_path)
    )

    engine.synthesize("abc", "en-US-Wavenet-F", "/tmp/a.wav")
    engine.synthesize("de", "en-US-Wavenet-F", "/tmp/b.wav")

    assert sum(json.loads(usage_path.read_text()).values()) == 5


def test_usage_warns_near_the_free_tier(tmp_path, caplog):
    usage_path = tmp_path / "usage.json"
    engine = RoutingTTSEngine(
        piper=RecordingEngine(), google=RecordingEngine(), usage_path=str(usage_path)
    )

    with caplog.at_level("WARNING"):
        engine.synthesize("x" * (FREE_TIER_CHARACTERS - 1), "en-US-Wavenet-F", "/tmp/a.wav")

    assert any("free-tier" in record.message for record in caplog.records)


def test_unwritable_usage_file_does_not_fail_the_render(tmp_path):
    """Metering is reporting. Losing it must not cost a render."""
    engine = RoutingTTSEngine(
        piper=RecordingEngine(),
        google=RecordingEngine(duration=2.5),
        usage_path="/proc/definitely-not-writable/usage.json",
    )

    assert engine.synthesize("hello", "en-US-Wavenet-F", "/tmp/a.wav") == 2.5


def test_google_failure_propagates_if_piper_also_fails(tmp_path):
    """With both engines down there is nothing to degrade to, so the saga must
    see a real failure rather than a silent empty file."""
    engine = make_engine(
        tmp_path, google=RecordingEngine(fail=True), piper=RecordingEngine(fail=True)
    )

    with pytest.raises(TTSEngineError):
        engine.synthesize("hello", "en-US-Wavenet-F", "/tmp/a.wav")
