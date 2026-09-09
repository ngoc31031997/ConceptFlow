"""Abstract port for the TTS Service (module-structure.md, ADR-0002).

The domain/application layers depend only on this abstraction, never on a
concrete engine — this is what let ADR-0023 add Google alongside the local
engine, and ADR-0024 replace that local engine with Edge, without touching
business logic.
"""

from __future__ import annotations

from abc import ABC, abstractmethod


class TTSEnginePort(ABC):
    """Synthesizes speech audio to a file and reports its duration."""

    @abstractmethod
    def synthesize(self, text: str, voice_id: str, output_path: str) -> float:
        """Write synthesized audio for `text` (using `voice_id`) to `output_path`.

        Returns:
            The resulting audio duration in seconds.

        Raises:
            domain.errors.TTSEngineError: if the engine fails or times out.
        """
