"""EdgeTTSAdapter — implements TTSEnginePort via Microsoft Edge's Read Aloud
voices (ADR-0024, replacing Piper).

These are the same neural voices Azure sells, but reached through the endpoint
Edge's reader uses, so they need no account, key or billing — which is the
whole reason this replaced Piper: `vi_VN-vivos-x_low` was the only male
Vietnamese voice Piper published, and its quality is the largest monetization
risk this pipeline carries (CR-001 §C1, CR-005).

Three consequences the rest of the service depends on:

- This engine needs the network. Nothing in the pipeline runs fully offline any
  more (ADR-0024 records that trade-off).
- The service returns MP3, so synthesis is a two-step write: MP3 from
  edge-tts, then ffmpeg to the 24 kHz mono WAV the rest of the pipeline reads
  with the `wave` module.
- The endpoint refuses requests intermittently, and hard under bursts: measured
  1/8 successes calling it scene-after-scene, which is exactly how a render
  drives it. It also hangs requests rather than refusing them outright (both are
  reported upstream as 403/503s and stalls). The failures are transient —
  retrying recovered 5/5 where spacing calls 3s apart still only managed 2/5 —
  so synthesis retries with backoff, and each attempt is capped so a stall
  cannot consume the budget the retries need. Without Piper behind it there is
  no engine left to degrade to, which is why this is load-bearing rather than a
  nicety.
"""

from __future__ import annotations

import logging
import os
import subprocess
import tempfile
import time
import wave
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import TimeoutError as FutureTimeoutError

from domain.errors import TTSEngineError
from domain.ports import TTSEnginePort

logger = logging.getLogger(__name__)

SAMPLE_RATE_HZ = 24000

MAX_ATTEMPTS = 4
RETRY_BACKOFF_SECONDS = 1.5

# Per-attempt caps. The endpoint does not only refuse requests, it also hangs
# them; without these a single hung attempt would eat the whole budget below and
# the retries that recover from it would never run.
CONNECT_TIMEOUT_SECONDS = 10
RECEIVE_TIMEOUT_SECONDS = 30

# The outer ceiling has to outlast every attempt plus its backoff, or it would
# cut the retry loop short — which is the failure this adapter exists to survive.
SYNTHESIS_TIMEOUT_SECONDS = (
    MAX_ATTEMPTS * (CONNECT_TIMEOUT_SECONDS + RECEIVE_TIMEOUT_SECONDS)
    + int(RETRY_BACKOFF_SECONDS * sum(range(1, MAX_ATTEMPTS)))
    + 15  # transcode and file I/O
)


class EdgeTTSAdapter(TTSEnginePort):
    """Raises TTSEngineError on any failure. The caller (RoutingTTSEngine) is
    what decides whether that is recoverable — this adapter stays a plain
    engine implementation, mirroring GoogleTTSAdapter."""

    def __init__(self, timeout_seconds: int = SYNTHESIS_TIMEOUT_SECONDS) -> None:
        self._timeout_seconds = timeout_seconds
        self._executor = ThreadPoolExecutor(max_workers=4, thread_name_prefix="edge-tts")

    def synthesize(self, text: str, voice_id: str, output_path: str) -> float:
        future = self._executor.submit(self._synthesize, text, voice_id, output_path)
        try:
            future.result(timeout=self._timeout_seconds)
        except FutureTimeoutError as exc:
            raise TTSEngineError(
                f"Edge TTS timed out after {self._timeout_seconds}s"
            ) from exc
        except TTSEngineError:
            raise
        except Exception as exc:  # noqa: BLE001 — any engine failure becomes a domain error
            logger.warning("Edge TTS failed for voice %s: %s", voice_id, exc)
            raise TTSEngineError(str(exc)) from exc

        with wave.open(output_path, "rb") as wav_file:
            return wav_file.getnframes() / float(wav_file.getframerate())

    def _synthesize(self, text: str, voice_id: str, output_path: str) -> None:
        mp3_fd, mp3_path = tempfile.mkstemp(
            suffix=".mp3", dir=os.path.dirname(output_path) or None
        )
        os.close(mp3_fd)
        try:
            self._fetch_mp3(text, voice_id, mp3_path)
            _transcode_to_wav(mp3_path, output_path)
        finally:
            if os.path.exists(mp3_path):
                os.remove(mp3_path)

    @staticmethod
    def _fetch_mp3(text: str, voice_id: str, mp3_path: str) -> None:
        import edge_tts

        for attempt in range(1, MAX_ATTEMPTS + 1):
            try:
                edge_tts.Communicate(
                    text,
                    voice_id,
                    connect_timeout=CONNECT_TIMEOUT_SECONDS,
                    receive_timeout=RECEIVE_TIMEOUT_SECONDS,
                ).save_sync(mp3_path)
                return
            except Exception as exc:  # noqa: BLE001 — every refusal is retryable
                if attempt == MAX_ATTEMPTS:
                    raise TTSEngineError(
                        f"Edge TTS refused {MAX_ATTEMPTS} attempts for voice "
                        f"{voice_id}: {exc}"
                    ) from exc
                logger.warning(
                    "Edge TTS attempt %d/%d failed for voice %s (%s); retrying.",
                    attempt,
                    MAX_ATTEMPTS,
                    voice_id,
                    exc,
                )
                time.sleep(RETRY_BACKOFF_SECONDS * attempt)


def _transcode_to_wav(mp3_path: str, output_path: str) -> None:
    result = subprocess.run(
        [
            "ffmpeg",
            "-y",
            "-loglevel",
            "error",
            "-i",
            mp3_path,
            "-ar",
            str(SAMPLE_RATE_HZ),
            "-ac",
            "1",
            "-c:a",
            "pcm_s16le",
            output_path,
        ],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise TTSEngineError(f"ffmpeg exited with code {result.returncode}: {result.stderr}")
