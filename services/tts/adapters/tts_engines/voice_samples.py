"""Generates one short preview clip per bundled voice at startup (CR-001 FR4.5).

Written to the shared volume rather than the image so the API Gateway can
serve the files without reaching into the TTS container. Regenerated only
when missing, so a restart costs nothing after the first run.
"""

from __future__ import annotations

import logging
import os

from adapters.tts_engines.voice_registry import VOICES, export_catalog, sample_path
from domain.ports import TTSEnginePort

logger = logging.getLogger(__name__)

SAMPLE_TEXT = {
    "vi": "Xin chào, đây là giọng đọc mẫu cho video của bạn.",
    "en": "Hello, this is a sample of the voice for your video.",
}


def generate_missing_samples(engine: TTSEnginePort) -> None:
    export_catalog()
    for voice in VOICES:
        path = sample_path(voice.voice_id)
        if os.path.isfile(path):
            continue
        os.makedirs(os.path.dirname(path), exist_ok=True)
        try:
            engine.synthesize(SAMPLE_TEXT[voice.language], voice.voice_id, path)
        except Exception:  # noqa: BLE001 — a missing preview must not stop the service
            logger.exception("Failed to generate voice sample for %s", voice.voice_id)
