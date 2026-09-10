"""Ports (abstract interfaces) that adapters must implement.

Per Hexagonal architecture (ADR-0002), domain code depends only on these
abstractions — never on Manim/subprocess directly.
"""

from __future__ import annotations

from abc import ABC, abstractmethod

from domain.models import (
    ChannelAssetRenderRequest,
    ChannelAssetRenderResult,
    DryRunResult,
    ScriptRenderRequest,
    ScriptRenderResult,
)


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


class ChannelAssetRendererPort(ABC):
    """Renders one of the two fixed channel-identity scenes
    (`conceptflow.channel_idents.DefaultIntroSting` / `.ChannelOutro`) —
    CR-023 FR65, D3/D4.

    Deliberately a **separate** port from `ManimScriptRendererPort` rather
    than a new abstract method on it: adding a method there would force every
    existing implementer (including CR-018/020's test fakes) to grow a stub
    they have no use for, just to keep satisfying `ABC`. `ManimScriptRenderer`
    implements both ports — one adapter, two narrow interfaces.
    """

    @abstractmethod
    def render_channel_asset(
        self, request: ChannelAssetRenderRequest, output_path: str
    ) -> ChannelAssetRenderResult:
        """Renders `request.kind`'s scene to `output_path`.

        Unlike `ManimScriptRendererPort.render()`, there is no narration-timing
        contract to honour — neither scene calls `self.narrate(...)`, so this
        is a single Manim pass with no marks file to reconcile.

        Raises:
            domain.errors.AnimationEngineError: if the engine fails or times out.
        """

    @abstractmethod
    def resolve_render_quality(self, requested: str | None) -> str:
        """The quality this renderer would actually use for `requested`.

        Exposed because the caller has to name the output path before the
        render runs, and FR65.5 keys that path by quality — an absent or
        unknown `requested` falls back to the service default, and the caller
        cannot guess which one that is.
        """
