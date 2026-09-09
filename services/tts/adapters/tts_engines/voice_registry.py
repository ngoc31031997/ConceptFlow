"""Voice catalog — the single source of truth for which voices this service
offers (CR-001, replacing the earlier one-voice-per-language map from
Low-Level Design Question 4).

The catalog is exported to the shared volume at startup (see
`export_catalog`) so the API Gateway can serve `GET /v1/voices` without the
TTS Service needing an HTTP server of its own — ADR-0014 deliberately
removed REST from this service.
"""

from __future__ import annotations

import json
import os
from collections.abc import Collection
from dataclasses import asdict, dataclass

SHARED_VOLUME_ROOT = "/shared"
VOICE_SAMPLES_DIRNAME = "voice_samples"
CATALOG_FILENAME = "catalog.json"


# Engine identifiers. Edge is the engine every project uses by default
# (ADR-0024); Azure reaches the same voices through the documented paid API
# (CR-011); Google is kept wired but dormant (ADR-0023). Azure and Google only
# run when their credentials are configured.
ENGINE_EDGE = "edge"
ENGINE_AZURE = "azure"
ENGINE_GOOGLE = "google"

# Azure and Edge serve the identical voice names, so Azure's catalogue entries
# are prefixed to keep the two selectable separately without colliding here.
AZURE_VOICE_PREFIX = "azure:"


@dataclass(frozen=True)
class Voice:
    voice_id: str
    language: str
    gender: str
    quality: str
    label: str
    engine: str = ENGINE_EDGE


VOICES: tuple[Voice, ...] = (
    # Edge Read Aloud neural voices (ADR-0024) — the default tier. Same neural
    # voices Azure sells, reached without an account or key. These are the only
    # two Vietnamese voices the service publishes.
    Voice("vi-VN-HoaiMyNeural", "vi", "female", "neural", "Tiếng Việt — Nữ"),
    Voice("vi-VN-NamMinhNeural", "vi", "male", "neural", "Tiếng Việt — Nam"),
    Voice("en-US-JennyNeural", "en", "female", "neural", "English — Female"),
    Voice("en-US-GuyNeural", "en", "male", "neural", "English — Male"),
    # Azure AI Speech (CR-011) — the same four voices through the documented
    # API: 500k neural characters a month free, commercial rights, an SLA.
    # Selectable only when AZURE_SPEECH_KEY/REGION are configured.
    Voice(
        f"{AZURE_VOICE_PREFIX}vi-VN-HoaiMyNeural", "vi", "female", "neural",
        "Tiếng Việt — Nữ", ENGINE_AZURE,
    ),
    Voice(
        f"{AZURE_VOICE_PREFIX}vi-VN-NamMinhNeural", "vi", "male", "neural",
        "Tiếng Việt — Nam", ENGINE_AZURE,
    ),
    Voice(
        f"{AZURE_VOICE_PREFIX}en-US-JennyNeural", "en", "female", "neural",
        "English — Female", ENGINE_AZURE,
    ),
    Voice(
        f"{AZURE_VOICE_PREFIX}en-US-GuyNeural", "en", "male", "neural",
        "English — Male", ENGINE_AZURE,
    ),
    # Google WaveNet (ADR-0023) — dormant since Google withdrew its free tier
    # (CR-010). Selectable only if GOOGLE_APPLICATION_CREDENTIALS is set.
    Voice("vi-VN-Wavenet-A", "vi", "female", "wavenet", "Tiếng Việt — Nữ", ENGINE_GOOGLE),
    Voice("vi-VN-Wavenet-B", "vi", "male", "wavenet", "Tiếng Việt — Nam", ENGINE_GOOGLE),
    Voice("vi-VN-Wavenet-C", "vi", "female", "wavenet", "Tiếng Việt — Nữ 2", ENGINE_GOOGLE),
    Voice("en-US-Wavenet-F", "en", "female", "wavenet", "English — Female", ENGINE_GOOGLE),
    Voice("en-US-Wavenet-D", "en", "male", "wavenet", "English — Male", ENGINE_GOOGLE),
)

DEFAULT_VOICE_BY_LANGUAGE: dict[str, str] = {
    "vi": "vi-VN-HoaiMyNeural",
    "en": "en-US-JennyNeural",
}

# Used when Google is unavailable and a Google voice was requested.
FALLBACK_VOICE_BY_LANGUAGE: dict[str, str] = {
    "vi": "vi-VN-HoaiMyNeural",
    "en": "en-US-JennyNeural",
}

_BY_ID = {voice.voice_id: voice for voice in VOICES}


def get_voice(voice_id: str) -> Voice:
    """Raises KeyError if the voice is not bundled in this image."""
    return _BY_ID[voice_id]


def azure_voice_name(voice_id: str) -> str:
    """The name Azure itself knows, with the catalogue prefix stripped."""
    return voice_id.removeprefix(AZURE_VOICE_PREFIX)


def engine_for(voice_id: str) -> str:
    """Which engine owns a voice. Unknown ids are treated as Edge, since that
    is the engine that works without credentials."""
    voice = _BY_ID.get(voice_id)
    return voice.engine if voice else ENGINE_EDGE


def voices_for_engine(engine: str) -> list[Voice]:
    return [voice for voice in VOICES if voice.engine == engine]


def fallback_voice_for(voice_id: str) -> str:
    """The voice to use when a Google voice cannot be synthesized (CR-005
    FR13.3). Falls back within the same language so a Vietnamese project does
    not suddenly speak English."""
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


def samples_dir() -> str:
    return os.path.join(SHARED_VOLUME_ROOT, VOICE_SAMPLES_DIRNAME)


def sample_path(voice_id: str) -> str:
    return os.path.join(samples_dir(), f"{voice_id}.wav")


def catalog_path() -> str:
    return os.path.join(samples_dir(), CATALOG_FILENAME)


def offered_voices(available_engines: Collection[str] | None = None) -> list[Voice]:
    """The voices this deployment can actually deliver.

    A voice whose engine has no credential is not an option — RoutingTTSEngine
    silently substitutes the equivalent Edge voice for it, so offering it means
    promising a voice the Creator will never hear. Before this filter the
    catalogue advertised all thirteen voices on a stack configured for none of
    the metered engines, and the only trace of the substitution was a log line
    inside the container.

    `None` means "no filtering" — kept so the full catalogue stays inspectable.
    """
    if available_engines is None:
        return list(VOICES)
    allowed = set(available_engines)
    return [voice for voice in VOICES if voice.engine in allowed]


def export_catalog(available_engines: Collection[str] | None = None) -> None:
    """Write the catalog to the shared volume for the API Gateway to serve."""
    path = catalog_path()
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(
            [asdict(voice) for voice in offered_voices(available_engines)],
            f,
            ensure_ascii=False,
            indent=2,
        )
