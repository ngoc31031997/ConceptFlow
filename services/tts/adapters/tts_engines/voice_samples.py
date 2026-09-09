"""Generates one short preview clip per offered voice at startup (CR-001 FR4.5).

Written to the shared volume rather than the image so the API Gateway can
serve the files without reaching into the TTS container. Regenerated only
when missing, so a restart costs nothing after the first run.
"""

from __future__ import annotations

import logging
import os
from collections.abc import Collection

from adapters.tts_engines.voice_registry import (
    export_catalog,
    offered_voices,
    sample_path,
    samples_dir,
)
from domain.ports import TTSEnginePort

logger = logging.getLogger(__name__)

SAMPLE_TEXT = {
    "vi": "Xin chào, đây là giọng đọc mẫu cho video của bạn.",
    "en": "Hello, this is a sample of the voice for your video.",
}


def _prune_stale_samples(offered_ids: set[str]) -> None:
    """Deletes preview clips for voices this deployment no longer offers.

    Not housekeeping — correctness. A clip is kept across restarts, and a clip
    for an unconfigured metered voice was produced by the Edge fallback rather
    than by the engine on its label. Leaving it means that adding
    AZURE_SPEECH_KEY later would never regenerate it (the file already exists),
    so the Azure previews would go on playing Edge audio forever. The same
    delete also clears the Piper clips left behind by ADR-0024.
    """
    directory = samples_dir()
    if not os.path.isdir(directory):
        return
    for name in os.listdir(directory):
        if not name.endswith(".wav"):
            continue
        if name[: -len(".wav")] in offered_ids:
            continue
        try:
            os.remove(os.path.join(directory, name))
            logger.info("Removed stale voice sample %s", name)
        except OSError:
            logger.warning("Could not remove stale voice sample %s", name, exc_info=True)


def generate_missing_samples(
    engine: TTSEnginePort, available_engines: Collection[str] | None = None
) -> None:
    voices = offered_voices(available_engines)
    export_catalog(available_engines)
    _prune_stale_samples({voice.voice_id for voice in voices})

    for voice in voices:
        path = sample_path(voice.voice_id)
        if os.path.isfile(path):
            continue
        os.makedirs(os.path.dirname(path), exist_ok=True)
        try:
            engine.synthesize(SAMPLE_TEXT[voice.language], voice.voice_id, path)
        except Exception:  # noqa: BLE001 — a missing preview must not stop the service
            logger.exception("Failed to generate voice sample for %s", voice.voice_id)
