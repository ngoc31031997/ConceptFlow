"""The two frames a video is built on and the safe area of each.

A long video is 1920x1080; a short is built vertically at 1080x1920, its safe
area keeping clear of the Shorts overlay (title and channel at the bottom, the
button column on the right). The same numbers live in
rendering/remotion_project/src/conceptflow-mini/primitives.tsx (FRAMES),
rendering/domain/frame.py and authoring-service/internal/domain/frame.go;
tests/test_frame.py compares against the TypeScript file.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class Frame:
    """A frame's size and its safe area (left, top, right, bottom, in px)."""

    width: int
    height: int
    safe: tuple[int, int, int, int]

    @property
    def size(self) -> str:
        return f"{self.width}x{self.height}"

    def safe_text(self) -> str:
        left, top, right, bottom = self.safe
        return f"({left}, {top})–({right}, {bottom})"


LANDSCAPE = Frame(1920, 1080, (96, 96, 1824, 984))
PORTRAIT = Frame(1080, 1920, (72, 200, 940, 1560))
FRAMES = (LANDSCAPE, PORTRAIT)
MAX_HEIGHT = max(f.height for f in FRAMES)


def frame_of(width: int, height: int) -> Frame:
    """The frame of that exact size. Raises ValueError for any other size."""
    for f in FRAMES:
        if (f.width, f.height) == (width, height):
            return f
    raise ValueError(f"no {width}x{height} frame: a video is 1920x1080 or 1080x1920")
