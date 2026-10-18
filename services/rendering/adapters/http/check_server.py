"""HTTP surface of the Rendering service — only the compile check.

The service is otherwise a message consumer. This runs inside the same
process and event loop, so the checks take a semaphore: a check (tsc and a
headless-browser layout pass) is heavy and must not stack up beside a real
render.
"""

from __future__ import annotations

import asyncio
import logging
import time
from typing import Literal

from fastapi import FastAPI
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from adapters.http.check_metrics import CheckMetrics
from adapters.rendering.illustration_previewer import IllustrationPreviewError
from adapters.rendering.typescript_checker import TypeScriptCheckError
from application.check_script import CheckScriptUseCase, LayoutContext
from application.preview_illustration import PreviewIllustrationUseCase
from application.stills import StillsUseCase
from domain.kits import UnknownKitError
from domain.layout_rules import SubtitleBand

logger = logging.getLogger(__name__)


class IllustrationBody(BaseModel):
    kit_key: str  # the visual language the project is pinned to; required
    # Role of the drawing in this video: "hero", "support" or "background".
    tier: Literal["hero", "support", "background"] = "support"
    name: str
    code: str = ""  # trống = hình dựng sẵn của bộ minh hoạ
    props: dict = {}
    gif: bool = True
    # "figure" (a figure of a shot) or "backdrop" (a whole-frame place).
    kind: Literal["figure", "backdrop"] = "figure"


class StillBody(BaseModel):
    """One frame of a sample composition of a kit."""
    composition: str
    kit_key: str
    props: dict = {}
    frame: int = 10


class PruneBody(BaseModel):
    """Stills not in `keep` and older than `min_age_hours` are deleted."""
    keep: list[str] = []
    min_age_hours: float = 24


class SubtitleBandBody(BaseModel):
    """The strip burned-in subtitles cover, measured from the frame edge."""
    edge: Literal["top", "bottom"]
    px: int


class CheckBody(BaseModel):
    code: str
    kit_key: str  # the visual language the project is pinned to; required
    scene_class_name: str = ""
    # For the layout check (Remotion only). No band = nothing is
    # burned into the frame; no font = the Stage's default.
    subtitle_band: SubtitleBandBody | None = None
    video_font: str = ""


def create_check_app(
    use_case: CheckScriptUseCase, concurrency: int = 2, metrics: CheckMetrics | None = None,
    illustrations: PreviewIllustrationUseCase | None = None,
    stills: StillsUseCase | None = None,
    kits: dict[str, int] | None = None,
) -> FastAPI:
    metrics = metrics or CheckMetrics()
    app = FastAPI(title="rendering-check")
    gate = asyncio.Semaphore(concurrency)

    @app.get("/health")
    async def health() -> dict:
        return {"status": "ok"}

    @app.get("/metrics/checks")
    async def check_metrics() -> dict:
        return metrics.snapshot()

    async def run(engine: str, body: CheckBody):
        queued = time.monotonic()
        started = queued
        renders_running = 0
        timed_out = False
        try:
            async with gate:
                started = time.monotonic()
                # Sampled when the check starts, not when it was queued: this is
                # the render it actually competes with for CPU.
                renders_running = metrics.renders_running
                try:
                    band = body.subtitle_band
                    layout = LayoutContext(
                        subtitle_band=SubtitleBand(band.edge, band.px) if band else None,
                        video_font=body.video_font,
                        kit_key=body.kit_key,
                    )
                    out = await asyncio.to_thread(
                        use_case.check, engine, body.code, body.scene_class_name, layout)
                except ValueError as exc:
                    return JSONResponse({"error": str(exc)}, status_code=400)
                except TypeScriptCheckError as exc:
                    # The checker itself is broken: say so, never answer "ok".
                    timed_out = "timed out" in str(exc)
                    logger.error("typescript check could not run: %s", exc)
                    return JSONResponse({"error": str(exc)}, status_code=503)
        except asyncio.CancelledError:
            # The caller gave up (llm-service's RENDERING_CHECK_TIMEOUT_SECONDS).
            timed_out = True
            raise
        finally:
            now = time.monotonic()
            metrics.record(
                wait_seconds=started - queued, run_seconds=now - started,
                renders_running=renders_running, timed_out=timed_out,
            )
            logger.info(
                "check engine=%s wait=%.2fs run=%.2fs renders_running=%d timed_out=%s",
                engine, started - queued, now - started, renders_running, timed_out,
            )
        return {
            "ok": out.ok,
            "diagnostics": [
                {"message": d.message, "line": d.line, "kind": d.kind, "rule": d.rule}
                for d in out.diagnostics
            ],
            "warnings": out.warnings,
        }

    @app.post("/v1/check/remotion")
    async def check_remotion(body: CheckBody):
        return await run("remotion", body)

    # Xem trước một hình của thư viện minh hoạ. Hàng đợi riêng, một
    # yêu cầu một lúc — bộ dựng xem trước chỉ mở một tab trình duyệt.
    preview_gate = asyncio.Semaphore(1)

    @app.post("/v1/illustrations/preview")
    async def preview_illustration(body: IllustrationBody):
        if illustrations is None:
            return JSONResponse({"error": "illustration preview is not enabled"}, status_code=404)
        async with preview_gate:
            try:
                out = await asyncio.to_thread(
                    illustrations.run, body.kit_key, body.name, body.code, body.props, body.gif, body.kind,
                    body.tier)
            except UnknownKitError as exc:
                return JSONResponse({"error": str(exc)}, status_code=400)
            except (IllustrationPreviewError, TypeScriptCheckError) as exc:
                logger.error("illustration preview could not run: %s", exc)
                return JSONResponse({"error": str(exc)}, status_code=503)
        return {
            "ok": out.ok, "diagnostics": out.diagnostics, "warnings": out.warnings,
            "png": out.png, "gif": out.gif,
        }

    @app.get("/v1/kits")
    async def list_kits():
        return {"kits": [{"key": k, "version": v} for k, v in sorted((kits or {}).items())]}

    still_gate = asyncio.Semaphore(1)

    @app.post("/v1/stills")
    async def make_still(body: StillBody):
        if stills is None:
            return JSONResponse({"error": "stills are not enabled"}, status_code=404)
        async with still_gate:
            try:
                out = await asyncio.to_thread(stills.render, body.composition, body.kit_key, body.props, body.frame)
            except UnknownKitError as exc:
                return JSONResponse({"error": str(exc)}, status_code=400)
            except IllustrationPreviewError as exc:
                logger.error("still could not be rendered: %s", exc)
                return JSONResponse({"error": str(exc)}, status_code=503)
        return {"file": out.file, "cache_hit": out.cache_hit}

    @app.post("/v1/stills/prune")
    async def prune_stills(body: PruneBody):
        if stills is None:
            return JSONResponse({"error": "stills are not enabled"}, status_code=404)
        try:
            removed = await asyncio.to_thread(stills.prune, body.keep, body.min_age_hours)
        except ValueError as exc:
            return JSONResponse({"error": str(exc)}, status_code=400)
        return {"removed": removed}

    return app
