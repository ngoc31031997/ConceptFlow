"""Abstract port for the Video Assembly Service (module-structure.md, ADR-0002).

The application layer depends only on this abstraction, never on ffmpeg
directly — this is what lets the assembly engine be swapped later without
touching business logic.
"""

from __future__ import annotations

from abc import ABC, abstractmethod

from domain.models import VideoAssemblyRequest


class VideoAssemblerPort(ABC):
    """Concatenates the ordered narration audio_segments into one track,
    muxes it onto request.video_path, overlays optional background music,
    and writes the result to output_path."""

    @abstractmethod
    def assemble(self, request: VideoAssemblyRequest, output_path: str) -> str | None:
        """Assembles request.video_path + request.audio_segments (+
        optional background music) into a single video file at output_path.

        Returns the path to a .srt caption-track file when request.subtitle_mode
        produced one (CR-015 FR38.4) — "track" or "both" with cues present —
        or None otherwise.

        Raises:
            domain.errors.AssemblyEngineError: ffmpeg fails or times out.
        """
