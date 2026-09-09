"""AzureTTSAdapter — implements TTSEnginePort via Azure AI Speech (CR-009).

Azure serves the same neural voices EdgeTTSAdapter reaches for free, but as a
documented, account-backed API: an F0 resource allows 500,000 neural characters
a month at no cost, with commercial rights and an SLA. That is the difference
worth exposing to the Creator — same sound, different guarantees — so Azure is
offered as its own set of catalogue entries rather than silently substituted
(CR-011).

Unlike Edge, Azure can return RIFF/WAV directly, so there is no ffmpeg step:
`riff-24khz-16bit-mono-pcm` is exactly what the rest of the pipeline reads with
the `wave` module.

Credentials come from AZURE_SPEECH_KEY and AZURE_SPEECH_REGION, kept separate
from the Publisher's YouTube OAuth (different scope, different lifecycle — the
same principle ADR-0023 applied to Google).
"""

from __future__ import annotations

import logging
import os
import urllib.error
import urllib.request
import wave
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import TimeoutError as FutureTimeoutError
from xml.sax.saxutils import escape

from adapters.tts_engines.voice_registry import azure_voice_name
from domain.errors import TTSEngineError
from domain.ports import TTSEnginePort

logger = logging.getLogger(__name__)

SYNTHESIS_TIMEOUT_SECONDS = 60
SAMPLE_RATE_HZ = 24000
OUTPUT_FORMAT = "riff-24khz-16bit-mono-pcm"

KEY_ENV_VAR = "AZURE_SPEECH_KEY"
REGION_ENV_VAR = "AZURE_SPEECH_REGION"


def is_configured() -> bool:
    return bool(os.environ.get(KEY_ENV_VAR) and os.environ.get(REGION_ENV_VAR))


class AzureTTSAdapter(TTSEnginePort):
    """Raises TTSEngineError on any failure. The caller (RoutingTTSEngine) is
    what turns that into an Edge retry — this adapter deliberately does not know
    about fallback, so it stays a plain engine implementation."""

    def __init__(
        self,
        key: str | None = None,
        region: str | None = None,
        timeout_seconds: int = SYNTHESIS_TIMEOUT_SECONDS,
    ) -> None:
        self._key = key or os.environ[KEY_ENV_VAR]
        self._region = region or os.environ[REGION_ENV_VAR]
        self._timeout_seconds = timeout_seconds
        self._executor = ThreadPoolExecutor(max_workers=4, thread_name_prefix="azure-tts")

    def synthesize(self, text: str, voice_id: str, output_path: str) -> float:
        future = self._executor.submit(self._synthesize, text, voice_id, output_path)
        try:
            future.result(timeout=self._timeout_seconds)
        except FutureTimeoutError as exc:
            raise TTSEngineError(
                f"Azure TTS timed out after {self._timeout_seconds}s"
            ) from exc
        except TTSEngineError:
            raise
        except Exception as exc:  # noqa: BLE001 — any engine failure becomes a domain error
            logger.warning("Azure TTS failed for voice %s: %s", voice_id, exc)
            raise TTSEngineError(str(exc)) from exc

        with wave.open(output_path, "rb") as wav_file:
            return wav_file.getnframes() / float(wav_file.getframerate())

    def _synthesize(self, text: str, voice_id: str, output_path: str) -> None:
        voice_name = azure_voice_name(voice_id)
        request = urllib.request.Request(
            f"https://{self._region}.tts.speech.microsoft.com/cognitiveservices/v1",
            data=_build_ssml(text, voice_name).encode("utf-8"),
            headers={
                "Ocp-Apim-Subscription-Key": self._key,
                "Content-Type": "application/ssml+xml",
                "X-Microsoft-OutputFormat": OUTPUT_FORMAT,
                "User-Agent": "ConceptFlow",
            },
            method="POST",
        )

        try:
            with urllib.request.urlopen(request, timeout=self._timeout_seconds) as response:
                audio = response.read()
        except urllib.error.HTTPError as exc:
            # 401/403 means the key or region is wrong, 429 means the free tier
            # is spent — both have to reach the Creator as a real error rather
            # than a silently different voice.
            raise TTSEngineError(
                f"Azure TTS returned HTTP {exc.code} for voice {voice_name}: "
                f"{exc.reason}"
            ) from exc

        with open(output_path, "wb") as f:
            f.write(audio)


def _build_ssml(text: str, voice_name: str) -> str:
    # "vi-VN-NamMinhNeural" -> "vi-VN". Azure wants the locale on the root
    # element even though the voice name already carries it.
    language_code = "-".join(voice_name.split("-")[:2])
    return (
        f'<speak version="1.0" xml:lang="{language_code}">'
        f'<voice name="{voice_name}">{escape(text)}</voice>'
        f"</speak>"
    )
