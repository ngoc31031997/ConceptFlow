"""Domain-specific exceptions for the Video Assembly Service."""

from __future__ import annotations


class MissingArtifactError(Exception):
    """Raised when a required input file (clip/audio/background music) is
    missing, or the request is otherwise malformed (Business Rule 1)."""


class InvalidSceneIndexError(Exception):
    """Raised when scene_index values are not a contiguous 0-based
    sequence — duplicate or missing index (Business Rule 2, Revision LLD
    Question 2)."""


class InconsistentMediaFormatError(Exception):
    """Raised when ffprobe pre-check finds codec/resolution/framerate
    mismatch between animation clips (Business Rule 3, Revision LLD
    Question 3)."""


class AssemblyEngineError(Exception):
    """Raised when ffmpeg fails or times out (Business Rule 9).

    A transient failure — mapped to an `assembly_failed` event the
    Orchestrator may retry, input artifacts are left untouched.
    """
