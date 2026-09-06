"""FastAPI router for Publisher Service (ADR-0008: /v1 prefix).

The OAuth start/callback routes are outside the Saga (Business Rule 6) —
correlation uses X-Request-ID, not saga_id, and callback failures return
a REST error response rather than publishing an AMQP event.
"""

from __future__ import annotations

import asyncio
import logging

from fastapi import APIRouter, Request, Response
from fastapi.responses import RedirectResponse

from adapters.api.schemas import OAuthCallbackResponse
from adapters.logging.correlation import set_correlation_id
from adapters.youtube.oauth_flow import GoogleOAuthFlow
from application.handle_oauth_callback import HandleOAuthCallbackUseCase

logger = logging.getLogger(__name__)


def create_v1_router(
    oauth_flow: GoogleOAuthFlow, handle_callback_use_case: HandleOAuthCallbackUseCase
) -> APIRouter:
    router = APIRouter(prefix="/v1")

    @router.get("/auth/youtube/start")
    async def start(request: Request, response: Response, state: str | None = None) -> RedirectResponse:
        correlation_id = set_correlation_id(request.headers.get("X-Request-ID"))
        authorization_url = await asyncio.to_thread(oauth_flow.build_authorization_url, state)
        redirect = RedirectResponse(authorization_url, status_code=302)
        redirect.headers["X-Request-ID"] = correlation_id
        return redirect

    @router.get("/auth/youtube/callback", response_model=OAuthCallbackResponse)
    async def callback(
        request: Request, response: Response, code: str, state: str | None = None
    ) -> OAuthCallbackResponse:
        correlation_id = set_correlation_id(request.headers.get("X-Request-ID"))
        response.headers["X-Request-ID"] = correlation_id

        try:
            await asyncio.to_thread(handle_callback_use_case.handle, code)
        except Exception as exc:  # noqa: BLE001 — any exchange failure becomes a 400 (Business Rule 6)
            logger.warning("OAuth callback failed: %s", exc)
            response.status_code = 400
            return OAuthCallbackResponse(connected=False, error=str(exc), state=state)

        return OAuthCallbackResponse(connected=True, state=state)

    return router


def create_health_router(is_ready: callable) -> APIRouter:
    """Unversioned /health endpoint — infra concern, not part of the
    public v1 API contract (mirror Content Plugin Service)."""
    router = APIRouter()

    @router.get("/health", include_in_schema=False)
    def health() -> dict:
        return {"status": "ok" if is_ready() else "not_ready"}

    return router
