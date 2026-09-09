"""Unit tests for RoutingTTSEngine (CR-005 FR13.3 / FR13.5, ADR-0023, ADR-0024, CR-011)."""

from __future__ import annotations

import json

import pytest

from adapters.tts_engines.routing_engine import FREE_TIER_CHARACTERS, RoutingTTSEngine
from adapters.tts_engines.voice_registry import ENGINE_AZURE, ENGINE_GOOGLE
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


def make_engine(tmp_path, google=None, azure=None, edge=None):
    metered = {}
    if google is not None:
        metered[ENGINE_GOOGLE] = google
    if azure is not None:
        metered[ENGINE_AZURE] = azure
    return RoutingTTSEngine(
        edge=edge or RecordingEngine(),
        metered=metered,
        usage_path=str(tmp_path / "usage.json"),
    )


def test_google_voice_goes_to_google(tmp_path):
    google = RecordingEngine(duration=3.0)
    edge = RecordingEngine()
    engine = make_engine(tmp_path, google=google, edge=edge)

    duration = engine.synthesize("xin chao", "vi-VN-Wavenet-A", "/tmp/out.wav")

    assert duration == 3.0
    assert len(google.calls) == 1
    assert edge.calls == []


def test_edge_voice_bypasses_google_entirely(tmp_path):
    google = RecordingEngine()
    edge = RecordingEngine(duration=1.5)
    engine = make_engine(tmp_path, google=google, edge=edge)

    engine.synthesize("xin chao", "vi-VN-HoaiMyNeural", "/tmp/out.wav")

    assert google.calls == []
    assert len(edge.calls) == 1


def test_missing_credentials_fall_back_within_the_same_language(tmp_path):
    """FR13.3. A Vietnamese project must not suddenly speak English because the
    cloud engine was unavailable."""
    edge = RecordingEngine()
    engine = make_engine(tmp_path, google=None, edge=edge)

    engine.synthesize("xin chao", "vi-VN-Wavenet-A", "/tmp/out.wav")

    assert edge.calls[0][1] == "vi-VN-HoaiMyNeural"


def test_google_failure_falls_back_to_edge(tmp_path):
    """A network blip or expired credential should cost the chosen voice, not
    the whole render the Creator is waiting on."""
    google = RecordingEngine(fail=True)
    edge = RecordingEngine(duration=1.0)
    engine = make_engine(tmp_path, google=google, edge=edge)

    duration = engine.synthesize("hello", "en-US-Wavenet-F", "/tmp/out.wav")

    assert duration == 1.0
    assert edge.calls[0][1] == "en-US-JennyNeural"


def test_fallback_is_logged_not_silent(tmp_path, caplog):
    """Silently shipping a different voice than the one chosen is worse than
    the failure — the Creator has to know before they publish."""
    engine = make_engine(tmp_path, google=RecordingEngine(fail=True))

    with caplog.at_level("WARNING"):
        engine.synthesize("hello", "en-US-Wavenet-F", "/tmp/out.wav")

    assert any("falling back" in record.message.lower() for record in caplog.records)


def test_usage_is_metered_only_for_the_billed_engine(tmp_path):
    usage_path = tmp_path / "usage.json"
    engine = RoutingTTSEngine(
        edge=RecordingEngine(),
        metered={ENGINE_GOOGLE: RecordingEngine()},
        usage_path=str(usage_path),
    )

    engine.synthesize("12345", "en-US-Wavenet-F", "/tmp/a.wav")
    engine.synthesize("ignored-by-meter", "en-US-JennyNeural", "/tmp/b.wav")

    recorded = json.loads(usage_path.read_text())
    assert sum(month[ENGINE_GOOGLE] for month in recorded.values()) == 5


def test_usage_accumulates_across_calls(tmp_path):
    usage_path = tmp_path / "usage.json"
    engine = RoutingTTSEngine(
        edge=RecordingEngine(),
        metered={ENGINE_GOOGLE: RecordingEngine()},
        usage_path=str(usage_path),
    )

    engine.synthesize("abc", "en-US-Wavenet-F", "/tmp/a.wav")
    engine.synthesize("de", "en-US-Wavenet-F", "/tmp/b.wav")

    recorded = json.loads(usage_path.read_text())
    assert sum(month[ENGINE_GOOGLE] for month in recorded.values()) == 5


def test_usage_warns_near_the_free_tier(tmp_path, caplog):
    usage_path = tmp_path / "usage.json"
    engine = RoutingTTSEngine(
        edge=RecordingEngine(),
        metered={ENGINE_GOOGLE: RecordingEngine()},
        usage_path=str(usage_path),
    )

    with caplog.at_level("WARNING"):
        engine.synthesize(
            "x" * (FREE_TIER_CHARACTERS[ENGINE_GOOGLE] - 1), "en-US-Wavenet-F", "/tmp/a.wav"
        )

    assert any("free allowance" in record.message for record in caplog.records)


def test_unwritable_usage_file_does_not_fail_the_render(tmp_path):
    """Metering is reporting. Losing it must not cost a render."""
    engine = RoutingTTSEngine(
        edge=RecordingEngine(),
        metered={ENGINE_GOOGLE: RecordingEngine(duration=2.5)},
        usage_path="/proc/definitely-not-writable/usage.json",
    )

    assert engine.synthesize("hello", "en-US-Wavenet-F", "/tmp/a.wav") == 2.5


def test_google_failure_propagates_if_edge_also_fails(tmp_path):
    """With both engines down there is nothing to degrade to, so the saga must
    see a real failure rather than a silent empty file."""
    engine = make_engine(
        tmp_path, google=RecordingEngine(fail=True), edge=RecordingEngine(fail=True)
    )

    with pytest.raises(TTSEngineError):
        engine.synthesize("hello", "en-US-Wavenet-F", "/tmp/a.wav")


def test_azure_voice_goes_to_azure_when_configured(tmp_path):
    azure = RecordingEngine(duration=4.0)
    edge = RecordingEngine()
    engine = make_engine(tmp_path, azure=azure, edge=edge)

    duration = engine.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", "/tmp/out.wav")

    assert duration == 4.0
    assert edge.calls == []


def test_azure_voice_falls_back_to_the_same_language_edge_voice(tmp_path):
    """CR-011: picking an Azure voice with no credentials configured must still
    render, in the language the project asked for."""
    edge = RecordingEngine()
    engine = make_engine(tmp_path, azure=None, edge=edge)

    engine.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", "/tmp/out.wav")

    assert edge.calls[0][1] == "vi-VN-HoaiMyNeural"


def test_azure_failure_falls_back_to_edge(tmp_path):
    edge = RecordingEngine(duration=1.0)
    engine = make_engine(tmp_path, azure=RecordingEngine(fail=True), edge=edge)

    duration = engine.synthesize("hello", "azure:en-US-GuyNeural", "/tmp/out.wav")

    assert duration == 1.0
    assert edge.calls[0][1] == "en-US-JennyNeural"


def test_each_engine_is_metered_against_its_own_allowance(tmp_path, caplog):
    """Azure's free tier is 500k and Google's number means something different
    now, so one shared counter would warn at the wrong time for both."""
    usage_path = tmp_path / "usage.json"
    engine = RoutingTTSEngine(
        edge=RecordingEngine(),
        metered={ENGINE_AZURE: RecordingEngine(), ENGINE_GOOGLE: RecordingEngine()},
        usage_path=str(usage_path),
    )

    engine.synthesize("abc", "azure:en-US-GuyNeural", "/tmp/a.wav")
    engine.synthesize("de", "en-US-Wavenet-F", "/tmp/b.wav")

    recorded = json.loads(usage_path.read_text())
    month = next(iter(recorded.values()))
    assert month == {ENGINE_AZURE: 3, ENGINE_GOOGLE: 2}


def test_azure_warns_at_its_own_free_tier_not_googles(tmp_path, caplog):
    usage_path = tmp_path / "usage.json"
    engine = RoutingTTSEngine(
        edge=RecordingEngine(),
        metered={ENGINE_AZURE: RecordingEngine()},
        usage_path=str(usage_path),
    )
    over_azure_but_under_google = FREE_TIER_CHARACTERS[ENGINE_AZURE] - 1

    with caplog.at_level("WARNING"):
        engine.synthesize("x" * over_azure_but_under_google, "azure:en-US-GuyNeural", "/tmp/a.wav")

    assert any("free allowance" in record.message for record in caplog.records)
