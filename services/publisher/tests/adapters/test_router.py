"""Unit tests for the REST API layer (adapters/api/router.py).

Builds a minimal FastAPI app with fakes, no real Google OAuth/network
call needed.
"""

from __future__ import annotations

from fastapi import FastAPI
from fastapi.testclient import TestClient

from adapters.api.router import create_health_router, create_v1_router
from application.handle_oauth_callback import HandleOAuthCallbackUseCase


class FakeOAuthFlow:
    def __init__(self, authorization_url: str = "https://accounts.google.com/o/oauth2/auth?fake=1") -> None:
        self._authorization_url = authorization_url

    def build_authorization_url(self) -> str:
        return self._authorization_url

    def exchange_code(self, code: str):
        if code == "bad-code":
            raise ValueError("invalid code")
        from datetime import UTC, datetime

        from domain.models import OAuthCredential

        return OAuthCredential(
            access_token="token", refresh_token="refresh", expires_at=datetime.now(UTC), channel_id="UC1"
        )


class FakeCredentialStore:
    def get(self):
        return None

    def save(self, credential) -> None:
        pass


def _build_client() -> TestClient:
    oauth_flow = FakeOAuthFlow()
    handle_callback_use_case = HandleOAuthCallbackUseCase(oauth_flow, FakeCredentialStore())

    app = FastAPI()
    app.include_router(create_v1_router(oauth_flow, handle_callback_use_case))
    app.include_router(create_health_router(lambda: True))
    return TestClient(app)


def test_health_endpoint_returns_ok():
    client = _build_client()

    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"status": "ok"}


def test_start_redirects_to_google_authorization_url():
    client = _build_client()

    response = client.get("/v1/auth/youtube/start", follow_redirects=False)

    assert response.status_code == 302
    assert response.headers["location"] == "https://accounts.google.com/o/oauth2/auth?fake=1"


def test_callback_success_returns_connected_true():
    client = _build_client()

    response = client.get("/v1/auth/youtube/callback", params={"code": "good-code"})

    assert response.status_code == 200
    assert response.json() == {"connected": True, "error": None}


def test_callback_failure_returns_400():
    client = _build_client()

    response = client.get("/v1/auth/youtube/callback", params={"code": "bad-code"})

    assert response.status_code == 400
    assert response.json()["connected"] is False
