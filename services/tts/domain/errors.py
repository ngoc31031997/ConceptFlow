"""Domain-specific exceptions for the TTS Service."""

from __future__ import annotations


class EmptyTextError(Exception):
    """Raised when the narration text is empty (Business Rule 1)."""


class UnsupportedLanguageError(Exception):
    """Raised when the requested language has no registered voice model (Business Rule 2)."""

    def __init__(self, language: str, supported: list[str]) -> None:
        self.language = language
        self.supported = supported
        super().__init__(f"Unsupported language: {language!r} (supported: {supported})")


class TTSEngineError(Exception):
    """Raised when the underlying TTS engine fails or times out (Business Rule 5/6).

    A transient failure — mapped to a `synthesis_failed` event the Orchestrator may
    retry (ADR-0014), as opposed to the permanent EmptyTextError/UnsupportedLanguageError
    above.
    """
