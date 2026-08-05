"""FastAPI routes for the TTS Service (interface-contracts.md, ADR-0008 URI versioning)."""

from __future__ import annotations

from collections.abc import Callable

from fastapi import APIRouter, Header, HTTPException
from fastapi.responses import JSONResponse

from adapters.api.schemas import SynthesizeRequestBody, SynthesizeResponseBody
from adapters.logging.correlation import SAGA_ID_HEADER, get_request_logger
from application.synthesize_speech import SynthesizeSpeechUseCase
from domain.errors import EmptyTextError, TTSEngineError, UnsupportedLanguageError
from domain.models import SpeechRequest


def create_v1_router(use_case: SynthesizeSpeechUseCase) -> APIRouter:
    router = APIRouter(prefix="/v1")

    @router.post("/tts/synthesize", response_model=SynthesizeResponseBody)
    def synthesize(
        body: SynthesizeRequestBody,
        x_saga_id: str | None = Header(default=None, alias=SAGA_ID_HEADER),
    ):
        logger = get_request_logger(x_saga_id)
        request = SpeechRequest(body.project_id, body.scene_index, body.text, body.language)

        try:
            result = use_case.synthesize(request)
        except EmptyTextError:
            logger.warning(
                "Rejected empty text for project_id=%s scene_index=%s", body.project_id, body.scene_index
            )
            return JSONResponse(status_code=400, content={"error": "empty_text"})
        except UnsupportedLanguageError as exc:
            logger.warning("Rejected unsupported language=%s", body.language)
            return JSONResponse(
                status_code=400,
                content={"error": "unsupported_language", "supported": exc.supported},
            )
        except TTSEngineError as exc:
            logger.error("TTS engine failure: %s", exc)
            return JSONResponse(status_code=502, content={"error": "tts_engine_failure", "detail": str(exc)})

        logger.info(
            "Synthesized audio for project_id=%s scene_index=%s duration=%.2fs",
            body.project_id,
            body.scene_index,
            result.duration_seconds,
        )
        return SynthesizeResponseBody(audio_path=result.audio_path, duration_seconds=result.duration_seconds)

    return router


def create_health_router(is_ready: Callable[[], bool]) -> APIRouter:
    """Infrastructure Design Question 5: ready only once the voice model cache is loaded."""
    router = APIRouter()

    @router.get("/health")
    def health():
        if not is_ready():
            raise HTTPException(status_code=503, detail="voice models not loaded yet")
        return {"status": "ok"}

    return router
