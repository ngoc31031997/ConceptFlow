"""Unit tests for the REST API layer (adapters/api/router.py).

Builds a minimal FastAPI app with fakes, no real Google OAuth/network
call needed.
"""

from __future__ import annotations

from datetime import UTC, datetime

from fastapi import FastAPI
from fastapi.testclient import TestClient

from adapters.api.router import create_health_router, create_v1_router
from application.handle_oauth_callback import HandleOAuthCallbackUseCase
from domain.models import OAuthCredential


class FakeOAuthFlow:
    def __init__(self, authorization_url: str = "https://accounts.google.com/o/oauth2/auth?fake=1") -> None:
        self._authorization_url = authorization_url

    def build_authorization_url(self, state: str | None = None) -> str:
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
    def __init__(self) -> None:
        self._credential = None

    def get(self):
        return self._credential

    def save(self, credential) -> None:
        self._credential = credential


def _build_client(credential_store=None) -> TestClient:
    oauth_flow = FakeOAuthFlow()
    credential_store = credential_store or FakeCredentialStore()
    handle_callback_use_case = HandleOAuthCallbackUseCase(oauth_flow, credential_store)

    app = FastAPI()
    app.include_router(create_v1_router(oauth_flow, handle_callback_use_case, credential_store))
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

    response = client.get("/v1/auth/youtube/callback", params={"code": "good-code", "state": "project-1"})

    assert response.status_code == 200
    assert response.json() == {"connected": True, "error": None, "state": "project-1"}


def test_callback_failure_returns_400():
    client = _build_client()

    response = client.get("/v1/auth/youtube/callback", params={"code": "bad-code"})

    assert response.status_code == 400
    assert response.json()["connected"] is False


def test_status_returns_not_connected_when_no_credential():
    client = _build_client()

    response = client.get("/v1/auth/youtube/status")

    assert response.status_code == 200
    assert response.json() == {"connected": False}


def test_status_returns_connected_when_credential_saved():
    store = FakeCredentialStore()
    store.save(
        OAuthCredential(access_token="t", refresh_token="r", expires_at=datetime.now(UTC), channel_id="UC1")
    )
    client = _build_client(credential_store=store)

    response = client.get("/v1/auth/youtube/status")

    assert response.json() == {"connected": True}
