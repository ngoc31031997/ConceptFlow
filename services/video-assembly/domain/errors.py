"""Domain-specific exceptions for the Video Assembly Service."""

from __future__ import annotations


class MissingArtifactError(Exception):
    """Raised when a required input file (video/audio segment/background
    music) is missing, or the request is otherwise malformed (Business Rule 1)."""


class AssemblyEngineError(Exception):
    """Raised when ffmpeg fails or times out (Business Rule 9).

    A transient failure — mapped to an `assembly_failed` event the
    Orchestrator may retry, input artifacts are left untouched.
    """
