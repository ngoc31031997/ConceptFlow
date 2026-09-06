"""Voice catalog — the single source of truth for which Piper voices are
bundled in this image (CR-001, replacing the earlier one-voice-per-language
map from Low-Level Design Question 4).

The catalog is exported to the shared volume at startup (see
`export_catalog`) so the API Gateway can serve `GET /v1/voices` without the
TTS Service needing an HTTP server of its own — ADR-0014 deliberately
removed REST from this service.
"""

from __future__ import annotations

import json
import os
from dataclasses import asdict, dataclass

VOICE_MODEL_DIR = "/app/voices"
SHARED_VOLUME_ROOT = "/shared"
VOICE_SAMPLES_DIRNAME = "voice_samples"
CATALOG_FILENAME = "catalog.json"


@dataclass(frozen=True)
class Voice:
    voice_id: str
    language: str
    gender: str
    quality: str
    label: str


VOICES: tuple[Voice, ...] = (
    Voice("vi_VN-vais1000-medium", "vi", "female", "medium", "Tiếng Việt — Nữ (VAIS)"),
    Voice("vi_VN-vivos-x_low", "vi", "male", "x_low", "Tiếng Việt — Nam (VIVOS)"),
    Voice("en_US-lessac-medium", "en", "female", "medium", "English — Female (Lessac)"),
    Voice("en_US-ryan-high", "en", "male", "high", "English — Male (Ryan)"),
)

DEFAULT_VOICE_BY_LANGUAGE: dict[str, str] = {
    "vi": "vi_VN-vais1000-medium",
    "en": "en_US-lessac-medium",
}

_BY_ID = {voice.voice_id: voice for voice in VOICES}


def get_voice(voice_id: str) -> Voice:
    """Raises KeyError if the voice is not bundled in this image."""
    return _BY_ID[voice_id]


def get_voice_model_path(voice_id: str) -> str:
    """Raises KeyError if the voice is not bundled in this image."""
    return f"{VOICE_MODEL_DIR}/{_BY_ID[voice_id].voice_id}.onnx"


def resolve_voice_id(voice_id: str | None, language: str) -> str:
    """Falls back to the language's default voice when no voice was chosen —
    keeps projects created before CR-001 (which carry only a language) working.
    """
    if voice_id and voice_id in _BY_ID:
        return voice_id
    return DEFAULT_VOICE_BY_LANGUAGE[language]


def sample_path(voice_id: str) -> str:
    return os.path.join(SHARED_VOLUME_ROOT, VOICE_SAMPLES_DIRNAME, f"{voice_id}.wav")


def catalog_path() -> str:
    return os.path.join(SHARED_VOLUME_ROOT, VOICE_SAMPLES_DIRNAME, CATALOG_FILENAME)


def export_catalog() -> None:
    """Write the catalog to the shared volume for the API Gateway to serve."""
    path = catalog_path()
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump([asdict(voice) for voice in VOICES], f, ensure_ascii=False, indent=2)
