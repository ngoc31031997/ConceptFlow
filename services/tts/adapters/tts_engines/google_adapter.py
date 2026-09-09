"""GoogleTTSAdapter — implements TTSEnginePort via Google Cloud Text-to-Speech
(ADR-0023).

Dormant since Google withdrew the WaveNet free tier this adapter was chosen for
(CR-010); Edge owns the default voices now (ADR-0024). It stays wired because
the credential is the only thing it needs to work again.

Runs in a threadpool with a bounded timeout, mirroring EdgeTTSAdapter — the
Google client library is synchronous.

Output is 24 kHz mono LINEAR16 written as a .wav, matching what the rest of the
pipeline already reads with the `wave` module.
"""

from __future__ import annotations

import logging
import wave
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import TimeoutError as FutureTimeoutError

from domain.errors import TTSEngineError
from domain.ports import TTSEnginePort

logger = logging.getLogger(__name__)

SYNTHESIS_TIMEOUT_SECONDS = 60
SAMPLE_RATE_HZ = 24000


class GoogleTTSAdapter(TTSEnginePort):
    """Raises TTSEngineError on any failure. The caller (FallbackTTSEngine) is
    what turns that into an Edge retry — this adapter deliberately does not
    know about fallback, so it stays a plain engine implementation."""

    def __init__(self, timeout_seconds: int = SYNTHESIS_TIMEOUT_SECONDS) -> None:
        self._timeout_seconds = timeout_seconds
        self._executor = ThreadPoolExecutor(max_workers=4, thread_name_prefix="google-tts")
        self._client = None

    def _get_client(self):
        """Built lazily so the service starts even with no credentials — that
        case has to degrade to Edge, not crash the container on boot."""
        if self._client is None:
            from google.cloud import texttospeech

            self._client = texttospeech.TextToSpeechClient()
        return self._client

    def synthesize(self, text: str, voice_id: str, output_path: str) -> float:
        future = self._executor.submit(self._synthesize, text, voice_id, output_path)
        try:
            future.result(timeout=self._timeout_seconds)
        except FutureTimeoutError as exc:
            raise TTSEngineError(
                f"Google TTS timed out after {self._timeout_seconds}s"
            ) from exc
        except TTSEngineError:
            raise
        except Exception as exc:  # noqa: BLE001 — any engine failure becomes a domain error
            logger.warning("Google TTS failed for voice %s: %s", voice_id, exc)
            raise TTSEngineError(str(exc)) from exc

        with wave.open(output_path, "rb") as wav_file:
            return wav_file.getnframes() / float(wav_file.getframerate())

    def _synthesize(self, text: str, voice_id: str, output_path: str) -> None:
        from google.cloud import texttospeech

        # "vi-VN-Wavenet-A" -> language code "vi-VN". Google requires the
        # language code alongside the voice name even though the name implies it.
        language_code = "-".join(voice_id.split("-")[:2])

        response = self._get_client().synthesize_speech(
            input=texttospeech.SynthesisInput(text=text),
            voice=texttospeech.VoiceSelectionParams(
                language_code=language_code, name=voice_id
            ),
            audio_config=texttospeech.AudioConfig(
                audio_encoding=texttospeech.AudioEncoding.LINEAR16,
                sample_rate_hertz=SAMPLE_RATE_HZ,
            ),
        )

        # LINEAR16 comes back as a complete WAV container, so this is written
        # as-is rather than re-wrapped.
        with open(output_path, "wb") as f:
            f.write(response.audio_content)
