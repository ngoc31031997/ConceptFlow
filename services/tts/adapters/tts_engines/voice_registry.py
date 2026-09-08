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


# Engine identifiers (ADR-0023). Piper runs locally and is always available;
# Google needs credentials and the network, so it is the better voice but not
# the dependable one — which is exactly why it degrades to Piper rather than
# failing (CR-005 FR13.3).
ENGINE_PIPER = "piper"
ENGINE_GOOGLE = "google"


@dataclass(frozen=True)
class Voice:
    voice_id: str
    language: str
    gender: str
    quality: str
    label: str
    engine: str = ENGINE_PIPER


VOICES: tuple[Voice, ...] = (
    # Google WaveNet (ADR-0023) — the default tier. 4M free characters a month,
    # roughly 500 ten-minute videos, so at a personal channel's pace this is
    # free in practice.
    Voice("vi-VN-Wavenet-A", "vi", "female", "wavenet", "Tiếng Việt — Nữ (Google)", ENGINE_GOOGLE),
    Voice("vi-VN-Wavenet-B", "vi", "male", "wavenet", "Tiếng Việt — Nam (Google)", ENGINE_GOOGLE),
    Voice("vi-VN-Wavenet-C", "vi", "female", "wavenet", "Tiếng Việt — Nữ 2 (Google)", ENGINE_GOOGLE),
    Voice("en-US-Wavenet-F", "en", "female", "wavenet", "English — Female (Google)", ENGINE_GOOGLE),
    Voice("en-US-Wavenet-D", "en", "male", "wavenet", "English — Male (Google)", ENGINE_GOOGLE),
    # Piper — kept as the offline fallback (ADR-0010 is superseded as the
    # default, not removed).
    Voice("vi_VN-vais1000-medium", "vi", "female", "medium", "Tiếng Việt — Nữ (offline)"),
    Voice("vi_VN-vivos-x_low", "vi", "male", "x_low", "Tiếng Việt — Nam (offline)"),
    Voice("en_US-lessac-medium", "en", "female", "medium", "English — Female (offline)"),
    Voice("en_US-ryan-high", "en", "male", "high", "English — Male (offline)"),
)

# Google voices are the default because voice quality is the largest
# monetization risk this pipeline carries (CR-005).
DEFAULT_VOICE_BY_LANGUAGE: dict[str, str] = {
    "vi": "vi-VN-Wavenet-A",
    "en": "en-US-Wavenet-F",
}

# Used when Google is unavailable and a Google voice was requested.
FALLBACK_VOICE_BY_LANGUAGE: dict[str, str] = {
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


def engine_for(voice_id: str) -> str:
    """Which engine owns a voice. Unknown ids are treated as Piper, since that
    is the engine that works without credentials."""
    voice = _BY_ID.get(voice_id)
    return voice.engine if voice else ENGINE_PIPER


def voices_for_engine(engine: str) -> list[Voice]:
    return [voice for voice in VOICES if voice.engine == engine]


def fallback_voice_for(voice_id: str) -> str:
    """The offline voice to use when a Google voice cannot be synthesized
    (CR-005 FR13.3). Falls back within the same language so a Vietnamese
    project does not suddenly speak English."""
    voice = _BY_ID.get(voice_id)
    language = voice.language if voice else "en"
    return FALLBACK_VOICE_BY_LANGUAGE[language]


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
