"""Ports (abstract interfaces) that adapters must implement.

Per Hexagonal architecture (ADR-0002), domain code depends only on these
abstractions — never on Manim/subprocess directly.
"""

from __future__ import annotations

from abc import ABC, abstractmethod

from domain.models import ScriptRenderRequest, ScriptRenderResult


class ManimScriptRendererPort(ABC):
    """Executes one whole Manim script (the Creator's own scene class) to
    produce a single video file. Concrete implementation
    (ManimScriptRenderer) lives under adapters/rendering/."""

    @abstractmethod
    def render(self, request: ScriptRenderRequest, output_path: str) -> ScriptRenderResult:
        """Renders request.scene_class_name from request.script_content
        (after substituting `self.wait(AUTO)` calls with the ordered
        narration_segments' durations) to output_path.

        Returns the result carrying output_path plus wait_offsets — where each
        narration segment actually begins in the finished video (CR-002
        FR10.1) — and the video's real duration.

        Raises:
            domain.errors.AnimationEngineError: if the engine fails, times
                out, the script's `self.wait(AUTO)` count doesn't match the
                number of narration_segments, or the recorded timing marks
                don't line up one-to-one with them.
        """
