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

Synthesis retries with backoff (CR-013). Measured 16/20 successes calling the
endpoint with a valid key: the four failures were bare HTTP 401s with no JSON
body and an istio-envoy server header, scattered rather than clustered — Azure
gateway instances disagreeing about subscription state, not a bad key. Without
retries roughly one scene in five would fall back to Edge, inaudibly (both play
the same neural voices) but silently costing the commercial rights and SLA that
are the entire reason to choose Azure over Edge (ADR-0025).
"""

from __future__ import annotations

import logging
import os
import time
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

SAMPLE_RATE_HZ = 24000
OUTPUT_FORMAT = "riff-24khz-16bit-mono-pcm"

MAX_ATTEMPTS = 4
RETRY_BACKOFF_SECONDS = 1.5

# Cap on one HTTP attempt. Separate from the ceiling below because a single
# stalled request must not consume the budget the retries need — the trap
# EdgeTTSAdapter's comment warns about, and what the old flat 60s (used as both
# per-request and outer timeout) would have caused once retries existed.
ATTEMPT_TIMEOUT_SECONDS = 30

# Must outlast every attempt plus its backoff, or it would cut the retry loop
# short.
SYNTHESIS_TIMEOUT_SECONDS = (
    MAX_ATTEMPTS * ATTEMPT_TIMEOUT_SECONDS
    + int(RETRY_BACKOFF_SECONDS * sum(range(1, MAX_ATTEMPTS)))
    + 10  # file I/O
)

# Retrying a deterministic rejection is pure waste, so the split is explicit
# rather than "retry every failure". 401/403 are here because of the measured
# transient 401s above; a genuinely wrong key still fails every attempt, just
# a few seconds later, and RoutingTTSEngine falls back exactly as before.
RETRYABLE_STATUS_CODES = frozenset({401, 403, 408, 429, 500, 502, 503, 504})
MAX_RETRY_AFTER_SECONDS = 30  # ignore an absurd Retry-After rather than stalling a render

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
        audio = self._fetch_audio(text, voice_name)
        with open(output_path, "wb") as f:
            f.write(audio)

    def _fetch_audio(self, text: str, voice_name: str) -> bytes:
        """Retries transient rejections (CR-013). A permanent one — bad SSML, an
        unknown voice — is raised on the first attempt, since retrying a
        deterministic answer only delays the fallback."""
        for attempt in range(1, MAX_ATTEMPTS + 1):
            try:
                return self._attempt_fetch(text, voice_name)
            except urllib.error.HTTPError as exc:
                if exc.code not in RETRYABLE_STATUS_CODES:
                    raise TTSEngineError(
                        f"Azure TTS returned HTTP {exc.code} for voice {voice_name}: "
                        f"{exc.reason}"
                    ) from exc
                last_error: Exception = exc
                retry_after = _retry_after_seconds(exc)
            except (urllib.error.URLError, TimeoutError, OSError) as exc:
                # No response at all — nothing to conclude from, so treat it as
                # transient like EdgeTTSAdapter does.
                last_error = exc
                retry_after = None

            if attempt == MAX_ATTEMPTS:
                raise TTSEngineError(
                    f"Azure TTS failed {MAX_ATTEMPTS} attempts for voice "
                    f"{voice_name}: {last_error}"
                ) from last_error

            delay = retry_after if retry_after is not None else RETRY_BACKOFF_SECONDS * attempt
            logger.warning(
                "Azure TTS attempt %d/%d failed for voice %s (%s); retrying in %.1fs.",
                attempt,
                MAX_ATTEMPTS,
                voice_name,
                last_error,
                delay,
            )
            time.sleep(delay)

        raise AssertionError("unreachable — the loop either returns or raises")

    def _attempt_fetch(self, text: str, voice_name: str) -> bytes:
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
        with urllib.request.urlopen(request, timeout=ATTEMPT_TIMEOUT_SECONDS) as response:
            return response.read()


def _retry_after_seconds(exc: urllib.error.HTTPError) -> float | None:
    """Azure sends Retry-After on 429. Honouring it beats guessing, but it is
    clamped: an outsized value would stall a whole render behind one scene."""
    raw = exc.headers.get("Retry-After") if exc.headers else None
    if not raw:
        return None
    try:
        seconds = float(raw)
    except (TypeError, ValueError):
        return None  # HTTP-date form — fall back to the normal backoff
    return max(0.0, min(seconds, MAX_RETRY_AFTER_SECONDS))


def _build_ssml(text: str, voice_name: str) -> str:
    # "vi-VN-NamMinhNeural" -> "vi-VN". Azure wants the locale on the root
    # element even though the voice name already carries it.
    language_code = "-".join(voice_name.split("-")[:2])
    return (
        f'<speak version="1.0" xml:lang="{language_code}">'
        f'<voice name="{voice_name}">{escape(text)}</voice>'
        f"</speak>"
    )
