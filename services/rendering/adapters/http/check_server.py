"""HTTP surface of the Rendering service — only the compile check (CR-039 FR104).

The service is otherwise a message consumer. This runs inside the same
process and event loop, so the checks take a semaphore: a Manim dry run is
heavy and must not stack up beside a real render.
"""

from __future__ import annotations

import asyncio
import logging
import time

from fastapi import FastAPI
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from adapters.rendering.typescript_checker import TypeScriptCheckError
from adapters.http.check_metrics import CheckMetrics
from adapters.rendering.illustration_previewer import IllustrationPreviewError
from application.check_script import CheckScriptUseCase
from application.preview_illustration import PreviewIllustrationUseCase

logger = logging.getLogger(__name__)


class IllustrationBody(BaseModel):
    name: str
    code: str = ""  # trống = hình dựng sẵn của bộ minh hoạ
    props: dict = {}
    gif: bool = True


class CheckBody(BaseModel):
    code: str
    scene_class_name: str = ""


def create_check_app(
    use_case: CheckScriptUseCase, concurrency: int = 2, metrics: CheckMetrics | None = None,
    illustrations: PreviewIllustrationUseCase | None = None,
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
                    out = await asyncio.to_thread(use_case.check, engine, body.code, body.scene_class_name)
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
            "diagnostics": [{"message": d.message, "line": d.line} for d in out.diagnostics],
            "raw": out.raw,
        }

    @app.post("/v1/check/remotion")
    async def check_remotion(body: CheckBody):
        return await run("remotion", body)

    @app.post("/v1/check/manim")
    async def check_manim(body: CheckBody):
        return await run("manim", body)

    # CR-044: xem trước một hình của thư viện minh hoạ. Hàng đợi riêng, một
    # yêu cầu một lúc — bộ dựng xem trước chỉ mở một tab trình duyệt.
    preview_gate = asyncio.Semaphore(1)

    @app.post("/v1/illustrations/preview")
    async def preview_illustration(body: IllustrationBody):
        if illustrations is None:
            return JSONResponse({"error": "illustration preview is not enabled"}, status_code=404)
        async with preview_gate:
            try:
                out = await asyncio.to_thread(illustrations.run, body.name, body.code, body.props, body.gif)
            except (IllustrationPreviewError, TypeScriptCheckError) as exc:
                logger.error("illustration preview could not run: %s", exc)
                return JSONResponse({"error": str(exc)}, status_code=503)
        return {"ok": out.ok, "diagnostics": out.diagnostics, "png": out.png, "gif": out.gif}

    return app
