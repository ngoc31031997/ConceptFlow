"""Unit tests for the REST API layer (adapters/api/router.py).

Builds a minimal FastAPI app with fakes, no real Google OAuth/network
call needed.
"""

from __future__ import annotations

from fastapi import FastAPI
from fastapi.testclient import TestClient

from adapters.api.router import create_health_router, create_v1_router
from adapters.youtube.oauth_flow import RedirectUriNotRegisteredError
from adapters.youtube.oauth_state import NonceStore, decode_state
from application.handle_oauth_callback import HandleOAuthCallbackUseCase
from domain.models import OAuthApp, OAuthCredential
from tests.fakes import (
    TEST_CLIENT_ID,
    TEST_REDIRECT_URI,
    FakeOAuthAppRegistry,
    InMemoryCredentialStore,
    make_app,
    make_credential,
)


class FakeOAuthFlow:
    def __init__(self, authorization_url: str = "https://accounts.google.com/o/oauth2/auth?fake=1") -> None:
        self._authorization_url = authorization_url
        self.last_state: str | None = None
        self.last_app: OAuthApp | None = None

    @property
    def redirect_uri(self) -> str:
        return TEST_REDIRECT_URI

    def build_authorization_url(self, app: OAuthApp, state: str | None = None) -> str:
        if not app.accepts_redirect(TEST_REDIRECT_URI):
            raise RedirectUriNotRegisteredError(f"{app.label} is missing {TEST_REDIRECT_URI}")
        self.last_app = app
        self.last_state = state
        return f"{self._authorization_url}&state={state}"

    def exchange_code(self, code: str, app: OAuthApp) -> OAuthCredential:
        if code == "bad-code":
            raise ValueError("invalid code")
        return make_credential(channel_id="UC1", client_id=app.client_id, channel_title="Kênh Một")


def _build_client(credential_store=None, registry=None, oauth_flow=None, nonce_store=None):
    oauth_flow = oauth_flow or FakeOAuthFlow()
    credential_store = credential_store or InMemoryCredentialStore()
    registry = registry or FakeOAuthAppRegistry()
    nonce_store = nonce_store or NonceStore()
    handle_callback_use_case = HandleOAuthCallbackUseCase(oauth_flow, credential_store, registry)

    app = FastAPI()
    app.include_router(
        create_v1_router(
            oauth_flow, handle_callback_use_case, credential_store, registry, nonce_store
        )
    )
    app.include_router(create_health_router(lambda: True))
    return TestClient(app), oauth_flow, nonce_store


def _start_and_capture_state(client, oauth_flow, **params) -> str:
    client.get("/v1/auth/youtube/start", params=params, follow_redirects=False)
    return oauth_flow.last_state


def test_health_endpoint_returns_ok():
    client, _, _ = _build_client()

    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"status": "ok"}


def test_start_redirects_to_google_authorization_url():
    client, _, _ = _build_client()

    response = client.get("/v1/auth/youtube/start", follow_redirects=False)

    assert response.status_code == 302
    assert response.headers["location"].startswith("https://accounts.google.com/o/oauth2/auth?fake=1")


def test_start_encodes_the_app_and_project_into_state():
    client, oauth_flow, _ = _build_client()

    state = _start_and_capture_state(client, oauth_flow, state="project-1")

    decoded = decode_state(state)
    assert decoded.client_id == TEST_CLIENT_ID
    assert decoded.project_id == "project-1"
    assert decoded.nonce  # CSRF defence — must not be empty


def test_start_uses_the_only_app_without_being_told_which():
    client, oauth_flow, _ = _build_client()

    client.get("/v1/auth/youtube/start", follow_redirects=False)

    assert oauth_flow.last_app.client_id == TEST_CLIENT_ID


def test_start_requires_an_app_when_several_are_configured():
    registry = FakeOAuthAppRegistry([make_app(), make_app(client_id="client-b")])
    client, _, _ = _build_client(registry=registry)

    response = client.get("/v1/auth/youtube/start", follow_redirects=False)

    assert response.status_code == 400


def test_start_reports_a_missing_redirect_uri_instead_of_bouncing_to_google():
    """Google's own redirect_uri_mismatch page names neither the client nor
    the URI to add, so this must be caught first (CR-012 FR30.2)."""
    broken = make_app(redirect_uris=("http://localhost:3000/",))
    client, _, _ = _build_client(registry=FakeOAuthAppRegistry([broken]))

    response = client.get("/v1/auth/youtube/start", follow_redirects=False)

    assert response.status_code == 400
    assert TEST_REDIRECT_URI in response.json()["detail"]


def test_apps_endpoint_flags_a_client_whose_redirect_is_unregistered():
    broken = make_app(client_id="client-b", redirect_uris=("http://localhost:3000/",))
    client, _, _ = _build_client(registry=FakeOAuthAppRegistry([make_app(), broken]))

    body = client.get("/v1/auth/youtube/apps").json()

    by_id = {app["client_id"]: app for app in body}
    assert by_id[TEST_CLIENT_ID]["redirect_ok"] is True
    assert by_id["client-b"]["redirect_ok"] is False
    assert by_id["client-b"]["redirect_uri_hint"] == TEST_REDIRECT_URI


def test_apps_endpoint_never_exposes_the_client_secret():
    client, _, _ = _build_client()

    body = client.get("/v1/auth/youtube/apps").text

    assert "secret-a" not in body


def test_callback_success_returns_the_connected_channel():
    client, oauth_flow, _ = _build_client()
    state = _start_and_capture_state(client, oauth_flow, state="project-1")

    response = client.get("/v1/auth/youtube/callback", params={"code": "good-code", "state": state})

    assert response.status_code == 200
    assert response.json() == {
        "connected": True,
        "error": None,
        "state": "project-1",
        "channel_id": "UC1",
        "channel_title": "Kênh Một",
    }


def test_callback_failure_returns_400():
    client, oauth_flow, _ = _build_client()
    state = _start_and_capture_state(client, oauth_flow)

    response = client.get("/v1/auth/youtube/callback", params={"code": "bad-code", "state": state})

    assert response.status_code == 400
    assert response.json()["connected"] is False


def test_callback_rejects_a_state_this_server_never_issued():
    """Without this, a forged callback could attach an attacker's channel to
    the Creator's installation (CR-012 FR33.2)."""
    from adapters.youtube.oauth_state import encode_state

    client, _, _ = _build_client()
    forged = encode_state(TEST_CLIENT_ID, "project-1", "nonce-never-issued")

    response = client.get("/v1/auth/youtube/callback", params={"code": "good-code", "state": forged})

    assert response.status_code == 400
    assert response.json()["connected"] is False


def test_callback_rejects_a_replayed_state():
    client, oauth_flow, _ = _build_client()
    state = _start_and_capture_state(client, oauth_flow)

    first = client.get("/v1/auth/youtube/callback", params={"code": "good-code", "state": state})
    replay = client.get("/v1/auth/youtube/callback", params={"code": "good-code", "state": state})

    assert first.status_code == 200
    assert replay.status_code == 400


def test_status_returns_not_connected_when_no_credential():
    client, _, _ = _build_client()

    response = client.get("/v1/auth/youtube/status")

    assert response.status_code == 200
    assert response.json() == {"connected": False, "accounts": []}


def test_status_lists_every_connected_channel():
    store = InMemoryCredentialStore([
        make_credential(channel_id="UC1", is_default=True),
        make_credential(channel_id="UC2"),
    ])
    client, _, _ = _build_client(credential_store=store)

    body = client.get("/v1/auth/youtube/status").json()

    assert body["connected"] is True
    assert [a["channel_id"] for a in body["accounts"]] == ["UC1", "UC2"]


def test_accounts_endpoint_labels_a_channel_whose_client_is_gone():
    store = InMemoryCredentialStore([make_credential(channel_id="UC1", client_id="client-removed")])
    client, _, _ = _build_client(credential_store=store, registry=FakeOAuthAppRegistry([]))

    body = client.get("/v1/auth/youtube/accounts").json()

    assert "không còn cấu hình" in body[0]["app_label"]


def test_accounts_endpoint_flags_caption_scope(monkeypatch):
    """CR-015 FR40.2: the Creator learns a channel needs reconnecting from
    the channel picker, not from a video that quietly has no CC."""
    from adapters.youtube.oauth_flow import YOUTUBE_FORCE_SSL_SCOPE

    store = InMemoryCredentialStore(
        [
            make_credential(channel_id="UC_new", scopes=(YOUTUBE_FORCE_SSL_SCOPE,)),
            make_credential(channel_id="UC_old", scopes=()),
        ]
    )
    client, _, _ = _build_client(credential_store=store)

    body = client.get("/v1/auth/youtube/accounts").json()

    by_id = {a["channel_id"]: a for a in body}
    assert by_id["UC_new"]["has_caption_scope"] is True
    assert by_id["UC_old"]["has_caption_scope"] is False


def test_delete_account_disconnects_the_channel():
    store = InMemoryCredentialStore([make_credential(channel_id="UC1", is_default=True)])
    client, _, _ = _build_client(credential_store=store)

    response = client.delete("/v1/auth/youtube/accounts/UC1")

    assert response.status_code == 204
    assert store.list() == []


def test_set_default_moves_the_flag():
    store = InMemoryCredentialStore([
        make_credential(channel_id="UC1", is_default=True),
        make_credential(channel_id="UC2"),
    ])
    client, _, _ = _build_client(credential_store=store)

    response = client.post("/v1/auth/youtube/accounts/UC2/default")

    assert response.status_code == 200
    assert store.get("UC2").is_default is True
    assert store.get("UC1").is_default is False


def test_set_default_on_an_unknown_channel_returns_404():
    client, _, _ = _build_client()

    response = client.post("/v1/auth/youtube/accounts/UC_missing/default")

    assert response.status_code == 404
