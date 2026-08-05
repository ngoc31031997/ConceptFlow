"""Sanity tests for domain value objects (domain-entities.md)."""

from __future__ import annotations

from domain.models import SpeechRequest, SpeechResult


def test_speech_request_is_immutable():
    request = SpeechRequest("proj-1", 0, "hello", "en")
    assert request.project_id == "proj-1"
    assert request.scene_index == 0


def test_speech_result_fields():
    result = SpeechResult(audio_path="/shared/proj-1/audio/0_en.wav", duration_seconds=1.23)
    assert result.audio_path == "/shared/proj-1/audio/0_en.wav"
    assert result.duration_seconds == 1.23
