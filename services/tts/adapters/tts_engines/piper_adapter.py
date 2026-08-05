"""PiperTTSAdapter — implements TTSEnginePort by shelling out to the Piper
CLI binary (ADR-0010).

Piper's Python package (`piper-tts`) depends on `piper-phonemize`, which has
no prebuilt wheel for several platforms — shelling out to the standalone
Piper binary avoids that packaging problem while keeping the same behavior
(module-structure.md already anticipated "Piper CLI/binding").

Runs synthesis in a threadpool so it never blocks FastAPI's async event loop
(NFR Requirements, Performance), with a bounded timeout so a hung engine call
surfaces as a clear TTSEngineError instead of hanging the caller forever.
"""

from __future__ import annotations

import logging
import subprocess
import wave
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import TimeoutError as FutureTimeoutError

from adapters.tts_engines.voice_registry import get_voice_model_path
from domain.errors import TTSEngineError
from domain.ports import TTSEnginePort

logger = logging.getLogger(__name__)

SYNTHESIS_TIMEOUT_SECONDS = 60
PIPER_BINARY = "piper"


class PiperTTSAdapter(TTSEnginePort):
    """Voice model paths are resolved once at construction (NFR Design —
    in-process voice model cache: the .onnx files are memory-mapped by Piper
    on first use per language and stay warm in the OS page cache for the
    lifetime of the process).
    """

    def __init__(self, languages: list[str]) -> None:
        self._model_paths: dict[str, str] = {
            language: get_voice_model_path(language) for language in languages
        }
        self._executor = ThreadPoolExecutor(max_workers=4, thread_name_prefix="piper-synthesis")

    def synthesize(self, text: str, language: str, output_path: str) -> float:
        model_path = self._model_paths[language]

        future = self._executor.submit(self._run_piper, model_path, text, output_path)
        try:
            future.result(timeout=SYNTHESIS_TIMEOUT_SECONDS)
        except FutureTimeoutError as exc:
            raise TTSEngineError(f"Piper synthesis timed out after {SYNTHESIS_TIMEOUT_SECONDS}s") from exc
        except Exception as exc:  # noqa: BLE001 — any engine failure becomes a domain error
            logger.exception("Piper synthesis failed")
            raise TTSEngineError(str(exc)) from exc

        with wave.open(output_path, "rb") as wav_file:
            return wav_file.getnframes() / float(wav_file.getframerate())

    @staticmethod
    def _run_piper(model_path: str, text: str, output_path: str) -> None:
        result = subprocess.run(
            [PIPER_BINARY, "--model", model_path, "--output_file", output_path],
            input=text,
            capture_output=True,
            text=True,
            timeout=SYNTHESIS_TIMEOUT_SECONDS,
        )
        if result.returncode != 0:
            raise TTSEngineError(f"piper exited with code {result.returncode}: {result.stderr}")
