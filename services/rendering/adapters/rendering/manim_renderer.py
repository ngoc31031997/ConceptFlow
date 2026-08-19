"""ManimAnimationRenderer — implements AnimationRendererPort (ADR-0015).

Runs Manim's rendering pipeline in a threadpool so it never blocks the
asyncio event loop (NFR Requirements, Performance), with a bounded
timeout (RENDER_TIMEOUT_SECONDS, default 300s) so a hung render surfaces
as a clear AnimationEngineError instead of hanging the caller forever.

Manim writes output through its own media-directory convention rather
than to an arbitrary path, so each render uses an isolated temp
media_dir and the resulting .mp4 is moved to the caller's output_path
afterward.
"""

from __future__ import annotations

import logging
import os
import shutil
import tempfile
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import TimeoutError as FutureTimeoutError

from adapters.rendering.registry import AnimationTemplateRegistry
from adapters.storage.artifact_paths import read_duration_seconds
from domain.errors import AnimationEngineError, UnsupportedTemplateError
from domain.models import SceneRenderRequest
from domain.ports import AnimationRendererPort, AnimationTemplatePort

logger = logging.getLogger(__name__)

DEFAULT_RENDER_TIMEOUT_SECONDS = 300


class ManimAnimationRenderer(AnimationRendererPort):
    def __init__(
        self,
        registry: AnimationTemplateRegistry,
        timeout_seconds: int = DEFAULT_RENDER_TIMEOUT_SECONDS,
    ) -> None:
        self._registry = registry
        self._timeout_seconds = timeout_seconds
        self._executor = ThreadPoolExecutor(max_workers=1, thread_name_prefix="manim-render")

    def render(self, request: SceneRenderRequest, output_path: str) -> float:
        template = self._registry.get(request.animation_template_id)
        if template is None:
            raise UnsupportedTemplateError(request.animation_template_id)

        future = self._executor.submit(self._render_to_file, template, request, output_path)
        try:
            future.result(timeout=self._timeout_seconds)
        except FutureTimeoutError as exc:
            raise AnimationEngineError(
                f"Manim render timed out after {self._timeout_seconds}s"
            ) from exc
        except Exception as exc:  # noqa: BLE001 — any engine failure becomes a domain error
            logger.exception("Manim render failed")
            raise AnimationEngineError(str(exc)) from exc

        return read_duration_seconds(output_path)

    @staticmethod
    def _render_to_file(
        template: AnimationTemplatePort, request: SceneRenderRequest, output_path: str
    ) -> None:
        from manim import config

        scene = template.build_scene(request)

        media_dir = tempfile.mkdtemp(prefix="manim-media-")
        try:
            config.media_dir = media_dir
            config.disable_caching = True
            config.output_file = "scene"
            scene.render()

            rendered_path = ManimAnimationRenderer._find_rendered_file(media_dir)
            shutil.move(rendered_path, output_path)
        finally:
            shutil.rmtree(media_dir, ignore_errors=True)

    @staticmethod
    def _find_rendered_file(media_dir: str) -> str:
        for root, _dirs, files in os.walk(media_dir):
            for name in files:
                if name.endswith(".mp4"):
                    return os.path.join(root, name)
        raise AnimationEngineError(f"Manim did not produce an .mp4 file under {media_dir}")
