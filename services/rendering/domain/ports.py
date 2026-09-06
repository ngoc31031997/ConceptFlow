"""Ports (abstract interfaces) that adapters must implement.

Per Hexagonal architecture (ADR-0002), domain code depends only on these
abstractions — never on Manim/subprocess directly.
"""

from __future__ import annotations

from abc import ABC, abstractmethod

from domain.models import ScriptRenderRequest


class ManimScriptRendererPort(ABC):
    """Executes one whole Manim script (the Creator's own scene class) to
    produce a single video file. Concrete implementation
    (ManimScriptRenderer) lives under adapters/rendering/."""

    @abstractmethod
    def render(self, request: ScriptRenderRequest, output_path: str) -> None:
        """Renders request.scene_class_name from request.script_content
        (after substituting `self.wait(AUTO)` calls with the ordered
        narration_segments' durations) to output_path.

        Raises:
            domain.errors.AnimationEngineError: if the engine fails, times
                out, or the script's `self.wait(AUTO)` count doesn't match
                the number of narration_segments.
        """
