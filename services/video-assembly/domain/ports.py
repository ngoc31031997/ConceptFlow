"""Abstract port for the Video Assembly Service (module-structure.md, ADR-0002).

The application layer depends only on this abstraction, never on ffmpeg
directly — this is what lets the assembly engine be swapped later without
touching business logic.
"""

from __future__ import annotations

from abc import ABC, abstractmethod

from domain.models import VideoAssemblyRequest


class VideoAssemblerPort(ABC):
    """Muxes per-scene animation+audio, concatenates in scene order,
    overlays optional background music, and writes the result to
    output_path."""

    @abstractmethod
    def assemble(self, request: VideoAssemblyRequest, output_path: str) -> None:
        """Assembles request.scenes (+ optional background music) into a
        single video file at output_path.

        Raises:
            domain.errors.InvalidSceneIndexError: scene_index values are
                not a contiguous 0-based sequence.
            domain.errors.InconsistentMediaFormatError: animation clips
                have mismatched codec/resolution/framerate.
            domain.errors.AssemblyEngineError: ffmpeg fails or times out.
        """
