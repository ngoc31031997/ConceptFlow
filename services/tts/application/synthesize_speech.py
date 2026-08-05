"""SynthesizeSpeechUseCase — business-logic-model.md."""

from __future__ import annotations

from adapters.storage.artifact_paths import (
    audio_exists,
    compute_audio_path,
    ensure_parent_dir,
    read_duration_seconds,
)
from domain.errors import EmptyTextError, UnsupportedLanguageError
from domain.models import SpeechRequest, SpeechResult
from domain.ports import TTSEnginePort

SUPPORTED_LANGUAGES = ("vi", "en")


class SynthesizeSpeechUseCase:
    """Orchestrates validation, idempotency, and speech synthesis for one scene."""

    def __init__(self, engine: TTSEnginePort) -> None:
        self._engine = engine

    def synthesize(self, request: SpeechRequest) -> SpeechResult:
        text = request.text.strip()
        if not text:
            raise EmptyTextError

        if request.language not in SUPPORTED_LANGUAGES:
            raise UnsupportedLanguageError(request.language, list(SUPPORTED_LANGUAGES))

        audio_path = compute_audio_path(request.project_id, request.scene_index, request.language)

        if audio_exists(audio_path):
            # Idempotency (Business Rule 4): reuse the artifact from a prior call
            # instead of re-synthesizing.
            return SpeechResult(audio_path=audio_path, duration_seconds=read_duration_seconds(audio_path))

        # Text is passed to the engine verbatim — no preprocessing (Business Rule 3).
        ensure_parent_dir(audio_path)
        self._engine.synthesize(text, request.language, audio_path)
        duration_seconds = read_duration_seconds(audio_path)
        return SpeechResult(audio_path=audio_path, duration_seconds=duration_seconds)
