"""API layer tests using FastAPI TestClient (interface-contracts.md)."""

from __future__ import annotations

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from adapters.api.router import create_health_router, create_v1_router
from application.synthesize_speech import SynthesizeSpeechUseCase
from domain.ports import TTSEnginePort


class FakeTTSEngine(TTSEnginePort):
    def __init__(self, fail_with: Exception | None = None) -> None:
        self._fail_with = fail_with

    def synthesize(self, text: str, language: str, output_path: str) -> float:
        if self._fail_with is not None:
            raise self._fail_with
        import os
        import wave

        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with wave.open(output_path, "wb") as wav_file:
            wav_file.setnchannels(1)
            wav_file.setsampwidth(2)
            wav_file.setframerate(16000)
            wav_file.writeframes(b"\x00\x00" * 16000)
        return 1.0


def _build_client(engine: TTSEnginePort, ready: bool = True) -> TestClient:
    app = FastAPI()
    app.include_router(create_v1_router(SynthesizeSpeechUseCase(engine)))
    app.include_router(create_health_router(lambda: ready))
    return TestClient(app)


@pytest.fixture
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def test_synthesize_success(shared_volume_root):
    client = _build_client(FakeTTSEngine())

    response = client.post(
        "/v1/tts/synthesize",
        json={"project_id": "proj-1", "scene_index": 0, "text": "hello", "language": "en"},
        headers={"X-Saga-ID": "saga-123"},
    )

    assert response.status_code == 200
    body = response.json()
    assert body["duration_seconds"] == 1.0
    assert body["audio_path"].endswith("0_en.wav")


def test_synthesize_empty_text_returns_400(shared_volume_root):
    client = _build_client(FakeTTSEngine())

    response = client.post(
        "/v1/tts/synthesize",
        json={"project_id": "proj-1", "scene_index": 0, "text": "   ", "language": "en"},
    )

    assert response.status_code == 400
    assert response.json()["error"] == "empty_text"


def test_synthesize_unsupported_language_returns_400(shared_volume_root):
    client = _build_client(FakeTTSEngine())

    response = client.post(
        "/v1/tts/synthesize",
        json={"project_id": "proj-1", "scene_index": 0, "text": "hello", "language": "fr"},
    )

    assert response.status_code == 400
    body = response.json()
    assert body["error"] == "unsupported_language"
    assert body["supported"] == ["vi", "en"]


def test_synthesize_engine_failure_returns_502(shared_volume_root):
    from domain.errors import TTSEngineError

    client = _build_client(FakeTTSEngine(fail_with=TTSEngineError("boom")))

    response = client.post(
        "/v1/tts/synthesize",
        json={"project_id": "proj-1", "scene_index": 0, "text": "hello", "language": "en"},
    )

    assert response.status_code == 502
    assert response.json()["error"] == "tts_engine_failure"


def test_health_reports_not_ready_before_models_loaded(shared_volume_root):
    client = _build_client(FakeTTSEngine(), ready=False)

    response = client.get("/health")

    assert response.status_code == 503


def test_health_reports_ready_after_models_loaded(shared_volume_root):
    client = _build_client(FakeTTSEngine(), ready=True)

    response = client.get("/health")

    assert response.status_code == 200
