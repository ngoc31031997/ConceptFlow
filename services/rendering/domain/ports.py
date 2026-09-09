"""Ports (abstract interfaces) that adapters must implement.

Per Hexagonal architecture (ADR-0002), domain code depends only on these
abstractions — never on Manim/subprocess directly.
"""

from __future__ import annotations

from abc import ABC, abstractmethod

from domain.models import DryRunResult, ScriptRenderRequest, ScriptRenderResult


class ManimScriptRendererPort(ABC):
    """Executes one whole Manim script (the Creator's own scene class) to
    produce a single video file. Concrete implementation
    (ManimScriptRenderer) lives under adapters/rendering/."""

    @abstractmethod
    def dry_run(self, request: ScriptRenderRequest) -> DryRunResult:
        """Executes the script without producing a video, to learn which
        narration lines it produces and in what order (CR-018 FR49.1).

        Runs before TTS, so a script that fails here costs no voice quota
        (CR-020 FR56). request.narration_segments is ignored — the dry pass is
        what determines them.

        Raises:
            domain.errors.AnimationEngineError: if the script fails to run, times
                out, or produces no narration at all.
        """

    @abstractmethod
    def render(self, request: ScriptRenderRequest, output_path: str) -> ScriptRenderResult:
        """Renders request.scene_class_name from request.script_content to
        output_path, holding each `self.narrate(...)` for the matching
        narration_segments duration.

        Returns the result carrying output_path plus wait_offsets — where each
        narration segment actually begins in the finished video (CR-002
        FR10.1) — and the video's real duration.

        Raises:
            domain.errors.AnimationEngineError: if the engine fails, times
                out, or the recorded timing marks don't line up one-to-one with
                narration_segments — which means the script is not deterministic
                between the two passes.
        """
