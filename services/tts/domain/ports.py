"""Abstract port for the TTS Service (module-structure.md, ADR-0002).

The domain/application layers depend only on this abstraction, never on a
concrete engine — this is what lets ADR-0010 swap Piper for another engine
later without touching business logic.
"""

from __future__ import annotations

from abc import ABC, abstractmethod


class TTSEnginePort(ABC):
    """Synthesizes speech audio to a file and reports its duration."""

    @abstractmethod
    def synthesize(self, text: str, language: str, output_path: str) -> float:
        """Write synthesized audio for `text` (in `language`) to `output_path`.

        Returns:
            The resulting audio duration in seconds.

        Raises:
            domain.errors.TTSEngineError: if the engine fails or times out.
        """
