"""EngineRouterRenderer — picks Manim or Remotion per request (feature/remotion-engine).

`ScriptRenderRequest.engine` says which one; this class is the only place
that decision gets made, so every call site upstream (both command handlers,
both use cases) stays engine-agnostic — they already only know about
`ManimScriptRendererPort`, and this class satisfies that same port by
delegating to whichever concrete renderer matches.

Defaults to the Manim renderer for an unrecognized/empty `engine` — every
project created before this field existed sends none at all.
"""

from __future__ import annotations

import logging
from collections.abc import Callable

from domain.errors import AnimationEngineError
from domain.models import DryRunResult, ScriptRenderRequest, ScriptRenderResult
from domain.ports import ManimScriptRendererPort

logger = logging.getLogger(__name__)


class EngineRouterRenderer(ManimScriptRendererPort):
    def __init__(
        self,
        manim: ManimScriptRendererPort,
        remotion: ManimScriptRendererPort,
    ) -> None:
        self._renderers = {"manim": manim, "remotion": remotion}

    def set_heartbeat(self, callback: Callable[[float, int | None], None] | None) -> None:
        # Only ManimScriptRenderer implements this (heartbeat during a long
        # `manim` subprocess); RemotionScriptRenderer has no such hook yet.
        # RenderScriptUseCase.set_heartbeat() already tolerates a renderer
        # with none (`getattr(..., "set_heartbeat", None)`), so forward it to
        # every renderer that has one rather than picking just the active one
        # — the active engine isn't known until the request arrives.
        for renderer in self._renderers.values():
            setter = getattr(renderer, "set_heartbeat", None)
            if setter is not None:
                setter(callback)

    def dry_run(self, request: ScriptRenderRequest) -> DryRunResult:
        return self._renderer_for(request.engine).dry_run(request)

    def render(self, request: ScriptRenderRequest, output_path: str) -> ScriptRenderResult:
        return self._renderer_for(request.engine).render(request, output_path)

    def _renderer_for(self, engine: str) -> ManimScriptRendererPort:
        if not engine:
            return self._renderers["manim"]
        renderer = self._renderers.get(engine)
        if renderer is None:
            raise AnimationEngineError(
                f"unknown render engine {engine!r} — expected one of {sorted(self._renderers)}"
            )
        return renderer
