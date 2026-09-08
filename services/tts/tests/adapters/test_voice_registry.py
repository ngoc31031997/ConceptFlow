"""Unit tests for the CR-001 voice catalog."""

from __future__ import annotations

import json

import pytest

from adapters.tts_engines import voice_registry
from adapters.tts_engines.voice_registry import (
    VOICES,
    export_catalog,
    get_voice_model_path,
    resolve_voice_id,
)


def test_catalog_has_one_voice_per_language_and_gender():
    combos = {(voice.language, voice.gender) for voice in VOICES}
    assert combos == {("vi", "female"), ("vi", "male"), ("en", "female"), ("en", "male")}


def test_model_path_is_named_after_voice_id():
    assert get_voice_model_path("en_US-ryan-high").endswith("/en_US-ryan-high.onnx")


def test_unknown_voice_id_raises():
    with pytest.raises(KeyError):
        get_voice_model_path("nonexistent-voice")


@pytest.mark.parametrize(
    ("voice_id", "language", "expected"),
    [
        ("en_US-ryan-high", "en", "en_US-ryan-high"),
        (None, "vi", "vi-VN-Wavenet-A"),
        ("retired-voice", "en", "en-US-Wavenet-F"),
    ],
)
def test_resolve_voice_id_falls_back_to_language_default(voice_id, language, expected):
    assert resolve_voice_id(voice_id, language) == expected


def test_export_catalog_writes_every_voice(tmp_path, monkeypatch):
    monkeypatch.setattr(voice_registry, "SHARED_VOLUME_ROOT", str(tmp_path))

    export_catalog()

    written = json.loads((tmp_path / "voice_samples" / "catalog.json").read_text(encoding="utf-8"))
    assert [entry["voice_id"] for entry in written] == [voice.voice_id for voice in VOICES]
