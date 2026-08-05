"""Composition root for the TTS Service.

Wires domain/application/adapters together (constructor injection, per
dependency-injection.md) and exposes the FastAPI app. Voice models are
loaded once at startup (FastAPI lifespan) and kept in memory for the
lifetime of the process (NFR Design — in-process voice model cache).
"""

from __future__ import annotations

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI

from adapters.api.router import create_health_router, create_v1_router
from adapters.tts_engines.piper_adapter import PiperTTSAdapter
from application.synthesize_speech import SUPPORTED_LANGUAGES, SynthesizeSpeechUseCase

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class ServiceState:
    """Tracks readiness for the /health endpoint (Infrastructure Design,
    Question 5): ready only after voice models have finished loading."""

    def __init__(self) -> None:
        self.models_loaded = False

    def is_ready(self) -> bool:
        return self.models_loaded


def create_app() -> FastAPI:
    state = ServiceState()

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        engine = PiperTTSAdapter(languages=list(SUPPORTED_LANGUAGES))
        state.models_loaded = True
        logger.info("TTS Service ready — voice models loaded for %s", SUPPORTED_LANGUAGES)

        app.include_router(create_v1_router(SynthesizeSpeechUseCase(engine)))

        yield

    app = FastAPI(title="TTS Service", lifespan=lifespan)
    app.include_router(create_health_router(state.is_ready))
    return app
