"""FastAPI router for Publisher Service (ADR-0008: /v1 prefix).

The OAuth start/callback routes are outside the Saga (Business Rule 6) —
correlation uses X-Request-ID, not saga_id, and callback failures return
a REST error response rather than publishing an AMQP event.
"""

from __future__ import annotations

import asyncio
import logging

from fastapi import APIRouter, HTTPException, Request, Response
from fastapi.responses import RedirectResponse

from adapters.api.schemas import (
    OAuthAppResponse,
    OAuthCallbackResponse,
    OAuthStatusResponse,
    YouTubeAccountResponse,
)
from adapters.logging.correlation import set_correlation_id
from adapters.youtube.oauth_flow import (
    YOUTUBE_FORCE_SSL_SCOPE,
    GoogleOAuthFlow,
    RedirectUriNotRegisteredError,
)
from adapters.youtube.oauth_state import NonceStore, decode_state, encode_state
from application.handle_oauth_callback import HandleOAuthCallbackUseCase
from domain.models import OAuthCredential
from domain.ports import CredentialStorePort, OAuthAppRegistryPort

logger = logging.getLogger(__name__)


def create_v1_router(
    oauth_flow: GoogleOAuthFlow,
    handle_callback_use_case: HandleOAuthCallbackUseCase,
    credential_store: CredentialStorePort,
    app_registry: OAuthAppRegistryPort,
    nonce_store: NonceStore,
) -> APIRouter:
    router = APIRouter(prefix="/v1")

    def _to_account(credential: OAuthCredential) -> YouTubeAccountResponse:
        app = app_registry.get(credential.client_id)
        return YouTubeAccountResponse(
            channel_id=credential.channel_id,
            channel_title=credential.channel_title,
            client_id=credential.client_id,
            # An app whose secret file was removed still has channels
            # pointing at it; say so rather than showing a blank cell.
            app_label=app.label if app is not None else "(client không còn cấu hình)",
            is_default=credential.is_default,
            has_caption_scope=YOUTUBE_FORCE_SSL_SCOPE in credential.scopes,
        )

    @router.get("/auth/youtube/apps", response_model=list[OAuthAppResponse])
    async def list_apps() -> list[OAuthAppResponse]:
        redirect_uri = oauth_flow.redirect_uri
        return [
            OAuthAppResponse(
                client_id=app.client_id,
                label=app.label,
                project_id=app.project_id,
                source_file=app.source_file,
                redirect_ok=app.accepts_redirect(redirect_uri),
                # Surfaced up front so the Creator can fix Cloud Console
                # before being bounced to a Google error page (FR30.3).
                redirect_uri_hint=None if app.accepts_redirect(redirect_uri) else redirect_uri,
            )
            for app in app_registry.list()
        ]

    @router.get("/auth/youtube/status", response_model=OAuthStatusResponse)
    async def status() -> OAuthStatusResponse:
        credentials = await asyncio.to_thread(credential_store.list)
        return OAuthStatusResponse(
            connected=bool(credentials),
            accounts=[_to_account(credential) for credential in credentials],
        )

    @router.get("/auth/youtube/accounts", response_model=list[YouTubeAccountResponse])
    async def list_accounts() -> list[YouTubeAccountResponse]:
        credentials = await asyncio.to_thread(credential_store.list)
        return [_to_account(credential) for credential in credentials]

    @router.delete("/auth/youtube/accounts/{channel_id}", status_code=204)
    async def delete_account(channel_id: str) -> Response:
        await asyncio.to_thread(credential_store.delete, channel_id)
        return Response(status_code=204)

    @router.post("/auth/youtube/accounts/{channel_id}/default", response_model=YouTubeAccountResponse)
    async def make_default(channel_id: str) -> YouTubeAccountResponse:
        credential = await asyncio.to_thread(credential_store.get, channel_id)
        if credential is None:
            raise HTTPException(status_code=404, detail=f"channel {channel_id!r} is not connected")
        await asyncio.to_thread(credential_store.set_default, channel_id)
        refreshed = await asyncio.to_thread(credential_store.get, channel_id)
        return _to_account(refreshed or credential)

    @router.get("/auth/youtube/start")
    async def start(
        request: Request,
        response: Response,
        state: str | None = None,
        app: str | None = None,
    ) -> RedirectResponse:
        correlation_id = set_correlation_id(request.headers.get("X-Request-ID"))

        oauth_app = _select_app(app_registry, app)
        nonce = nonce_store.issue()
        # `state` arrives as the project_id the GUI wants to return to; it
        # gets wrapped together with the app id and the nonce.
        encoded_state = encode_state(oauth_app.client_id, state, nonce)

        try:
            authorization_url = await asyncio.to_thread(
                oauth_flow.build_authorization_url, oauth_app, encoded_state
            )
        except RedirectUriNotRegisteredError as exc:
            # 400 with the exact URI to register, instead of forwarding the
            # Creator to Google's redirect_uri_mismatch page (FR30.2).
            raise HTTPException(status_code=400, detail=str(exc)) from exc

        redirect = RedirectResponse(authorization_url, status_code=302)
        redirect.headers["X-Request-ID"] = correlation_id
        return redirect

    @router.get("/auth/youtube/callback", response_model=OAuthCallbackResponse)
    async def callback(
        request: Request, response: Response, code: str, state: str | None = None
    ) -> OAuthCallbackResponse:
        correlation_id = set_correlation_id(request.headers.get("X-Request-ID"))
        response.headers["X-Request-ID"] = correlation_id

        decoded = decode_state(state)
        project_id = decoded.project_id if decoded is not None else None

        if decoded is not None and decoded.nonce and not nonce_store.consume(decoded.nonce):
            # Unknown, expired or already-used nonce. Refusing here is what
            # stops a forged callback attaching someone else's channel.
            logger.warning("OAuth callback rejected: unrecognised state nonce")
            response.status_code = 400
            return OAuthCallbackResponse(
                connected=False,
                error="Phiên kết nối đã hết hạn hoặc không hợp lệ. Hãy bấm kết nối lại.",
                state=project_id,
            )

        client_id = decoded.client_id if decoded is not None else ""
        try:
            credential = await asyncio.to_thread(
                handle_callback_use_case.handle, code, client_id
            )
        except Exception as exc:  # noqa: BLE001 — any exchange failure becomes a 400 (Business Rule 6)
            logger.warning("OAuth callback failed: %s", exc)
            response.status_code = 400
            return OAuthCallbackResponse(connected=False, error=str(exc), state=project_id)

        return OAuthCallbackResponse(
            connected=True,
            state=project_id,
            channel_id=credential.channel_id,
            channel_title=credential.channel_title,
        )

    return router


def _select_app(app_registry: OAuthAppRegistryPort, client_id: str | None):
    """Resolves the ?app= parameter, defaulting to the only app when just
    one is configured so the single-app setup needs no picker (FR34.2)."""
    apps = app_registry.list()
    if not apps:
        raise HTTPException(
            status_code=503,
            detail=(
                "Chưa có OAuth client nào được cấu hình — thả file client_secret*.json "
                "vào thư mục secrets/ rồi khởi động lại publisher."
            ),
        )

    if client_id:
        app = app_registry.get(client_id)
        if app is None:
            raise HTTPException(status_code=404, detail=f"OAuth client {client_id!r} is not configured")
        return app

    if len(apps) == 1:
        return apps[0]

    raise HTTPException(
        status_code=400,
        detail="nhiều OAuth client đang được cấu hình — cần chỉ rõ ?app=<client_id>",
    )


def create_health_router(is_ready: callable) -> APIRouter:
    """Unversioned /health endpoint — infra concern, not part of the
    public v1 API contract (mirror Content Plugin Service)."""
    router = APIRouter()

    @router.get("/health", include_in_schema=False)
    def health() -> dict:
        return {"status": "ok" if is_ready() else "not_ready"}

    return router
