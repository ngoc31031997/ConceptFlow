"""MediaFormatInspector — ffprobe-based media format pre-check
(Functional Design Question 3, Revision LLD).

Runs BEFORE mux/concat so a codec/resolution/framerate mismatch between
animation clips surfaces as a clear domain error instead of an opaque
ffmpeg failure partway through assembly.
"""

from __future__ import annotations

import json
import subprocess
from dataclasses import dataclass

from domain.errors import InconsistentMediaFormatError

FFPROBE_BINARY = "ffprobe"


@dataclass(frozen=True)
class MediaFormat:
    """Internal value object — not part of any event contract, used only
    to compare clips within this adapter."""

    codec: str
    resolution: tuple[int, int]
    framerate: float


class MediaFormatInspector:
    def inspect(self, clip_path: str) -> MediaFormat:
        result = subprocess.run(
            [
                FFPROBE_BINARY,
                "-v",
                "error",
                "-select_streams",
                "v:0",
                "-show_entries",
                "stream=codec_name,width,height,r_frame_rate",
                "-of",
                "json",
                clip_path,
            ],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            raise InconsistentMediaFormatError(f"ffprobe failed for {clip_path}: {result.stderr.strip()}")

        stream = json.loads(result.stdout)["streams"][0]
        numerator, denominator = stream["r_frame_rate"].split("/")
        framerate = round(int(numerator) / int(denominator), 3)
        return MediaFormat(
            codec=stream["codec_name"],
            resolution=(stream["width"], stream["height"]),
            framerate=framerate,
        )

    def validate_consistent(self, clip_paths: list[str]) -> None:
        """Raises InconsistentMediaFormatError unless every clip shares the
        same codec/resolution/framerate as the first one (Business Rule 3)."""
        formats = [(clip_path, self.inspect(clip_path)) for clip_path in clip_paths]
        reference_path, reference_format = formats[0]

        mismatched = [
            clip_path for clip_path, media_format in formats[1:] if media_format != reference_format
        ]
        if mismatched:
            raise InconsistentMediaFormatError(
                f"media format mismatch vs {reference_path} ({reference_format}): {mismatched}"
            )
