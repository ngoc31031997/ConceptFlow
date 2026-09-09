"""Unit tests for the CR-001 voice catalog."""

from __future__ import annotations

import json

import pytest

from adapters.tts_engines import voice_registry
from adapters.tts_engines.voice_registry import (
    ENGINE_EDGE,
    VOICES,
    engine_for,
    export_catalog,
    fallback_voice_for,
    resolve_voice_id,
    voices_for_engine,
)


def test_catalog_covers_every_language_and_gender():
    combos = {(voice.language, voice.gender) for voice in voices_for_engine(ENGINE_EDGE)}
    assert combos == {("vi", "female"), ("vi", "male"), ("en", "female"), ("en", "male")}


def test_unknown_voice_is_treated_as_edge():
    """Edge owns every voice that needs no credential, so an id this build no
    longer knows must not be routed to the engine that can refuse it."""
    assert engine_for("retired-voice") == ENGINE_EDGE


@pytest.mark.parametrize(
    ("voice_id", "language", "expected"),
    [
        ("en-US-GuyNeural", "en", "en-US-GuyNeural"),
        (None, "vi", "vi-VN-HoaiMyNeural"),
        ("retired-voice", "en", "en-US-JennyNeural"),
    ],
)
def test_resolve_voice_id_falls_back_to_language_default(voice_id, language, expected):
    assert resolve_voice_id(voice_id, language) == expected


@pytest.mark.parametrize(
    ("voice_id", "expected"),
    [
        ("vi-VN-Wavenet-A", "vi-VN-HoaiMyNeural"),
        ("en-US-Wavenet-D", "en-US-JennyNeural"),
    ],
)
def test_google_voices_fall_back_within_their_own_language(voice_id, expected):
    assert fallback_voice_for(voice_id) == expected


def test_export_catalog_writes_every_voice(tmp_path, monkeypatch):
    monkeypatch.setattr(voice_registry, "SHARED_VOLUME_ROOT", str(tmp_path))

    export_catalog()

    written = json.loads((tmp_path / "voice_samples" / "catalog.json").read_text(encoding="utf-8"))
    assert [entry["voice_id"] for entry in written] == [voice.voice_id for voice in VOICES]


def test_azure_and_edge_offer_the_same_voices_under_distinct_keys():
    """CR-011 lists both because they differ in guarantees, not in sound. The
    prefix is what keeps the identical voice names from colliding here."""
    from adapters.tts_engines.voice_registry import ENGINE_AZURE, azure_voice_name

    edge_names = {v.voice_id for v in voices_for_engine(ENGINE_EDGE)}
    azure_names = {azure_voice_name(v.voice_id) for v in voices_for_engine(ENGINE_AZURE)}

    assert azure_names == edge_names
    assert len(VOICES) == len({v.voice_id for v in VOICES})


def test_azure_voices_route_to_azure():
    from adapters.tts_engines.voice_registry import ENGINE_AZURE

    assert engine_for("azure:vi-VN-NamMinhNeural") == ENGINE_AZURE
    assert engine_for("vi-VN-NamMinhNeural") == ENGINE_EDGE


def test_azure_voice_name_is_a_no_op_for_other_engines():
    from adapters.tts_engines.voice_registry import azure_voice_name

    assert azure_voice_name("vi-VN-NamMinhNeural") == "vi-VN-NamMinhNeural"
