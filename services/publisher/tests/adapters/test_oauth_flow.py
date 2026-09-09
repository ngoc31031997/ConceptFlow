"""Unit tests for GoogleOAuthFlow's scope handling (CR-015 FR40.1).

Mocks google_auth_oauthlib.flow.Flow so no real Google endpoint is needed.
"""

from __future__ import annotations

from datetime import UTC, datetime, timedelta
from unittest.mock import MagicMock, patch

from google.oauth2.credentials import Credentials as GoogleCredentials

from adapters.youtube.oauth_flow import (
    YOUTUBE_FORCE_SSL_SCOPE,
    YOUTUBE_READONLY_SCOPE,
    YOUTUBE_UPLOAD_SCOPE,
    GoogleOAuthFlow,
)
from tests.fakes import TEST_REDIRECT_URI, make_app


def _fake_google_credentials(granted_scopes=None, scopes=None) -> GoogleCredentials:
    creds = MagicMock(spec=GoogleCredentials)
    creds.token = "access-token"
    creds.refresh_token = "refresh-token"
    creds.expiry = datetime.now(UTC) + timedelta(hours=1)
    creds.granted_scopes = granted_scopes
    creds.scopes = scopes
    return creds


def _channel_response() -> dict:
    return {"items": [{"id": "UC123", "snippet": {"title": "My Channel"}}]}


def test_authorization_url_requests_force_ssl_alongside_upload_and_readonly():
    """FR40.1: captions.insert needs force-ssl, which upload/readonly do
    not cover — it must be requested at consent time, not discovered later
    as a missing scope on an existing token."""
    captured_scopes = {}

    def fake_from_client_config(client_config, scopes, **kwargs):
        captured_scopes["scopes"] = scopes
        mock_flow = MagicMock()
        mock_flow.authorization_url.return_value = ("https://accounts.google.com/auth", "state")
        return mock_flow

    with patch("adapters.youtube.oauth_flow.Flow.from_client_config", side_effect=fake_from_client_config):
        flow = GoogleOAuthFlow(redirect_uri=TEST_REDIRECT_URI)
        flow.build_authorization_url(make_app())

    assert set(captured_scopes["scopes"]) == {
        YOUTUBE_UPLOAD_SCOPE,
        YOUTUBE_READONLY_SCOPE,
        YOUTUBE_FORCE_SSL_SCOPE,
    }


def test_exchange_code_stores_the_scopes_google_actually_granted():
    """Google's granted_scopes can be a strict subset of what was requested
    — the Creator can untick one on the consent screen. That subset, not
    the request list, is what must end up on the credential."""
    mock_flow = MagicMock()
    mock_flow.credentials = _fake_google_credentials(
        granted_scopes=[YOUTUBE_UPLOAD_SCOPE, YOUTUBE_READONLY_SCOPE]
    )

    with (
        patch("adapters.youtube.oauth_flow.Flow.from_client_config", return_value=mock_flow),
        patch("adapters.youtube.oauth_flow.build") as mock_build,
    ):
        mock_build.return_value.channels.return_value.list.return_value.execute.return_value = (
            _channel_response()
        )
        flow = GoogleOAuthFlow(redirect_uri=TEST_REDIRECT_URI)
        credential = flow.exchange_code("auth-code", make_app())

    # force-ssl was requested but not granted — must NOT be on the credential.
    assert credential.scopes == (YOUTUBE_UPLOAD_SCOPE, YOUTUBE_READONLY_SCOPE)
    assert YOUTUBE_FORCE_SSL_SCOPE not in credential.scopes


def test_exchange_code_falls_back_to_requested_scopes_when_granted_scopes_is_absent():
    """Older google-auth-oauthlib versions may leave granted_scopes empty
    even when every scope was in fact granted — falling back to `.scopes`
    (what was requested) avoids treating a fully-successful consent as if
    nothing had been granted."""
    mock_flow = MagicMock()
    mock_flow.credentials = _fake_google_credentials(
        granted_scopes=None,
        scopes=[YOUTUBE_UPLOAD_SCOPE, YOUTUBE_READONLY_SCOPE, YOUTUBE_FORCE_SSL_SCOPE],
    )

    with (
        patch("adapters.youtube.oauth_flow.Flow.from_client_config", return_value=mock_flow),
        patch("adapters.youtube.oauth_flow.build") as mock_build,
    ):
        mock_build.return_value.channels.return_value.list.return_value.execute.return_value = (
            _channel_response()
        )
        flow = GoogleOAuthFlow(redirect_uri=TEST_REDIRECT_URI)
        credential = flow.exchange_code("auth-code", make_app())

    assert credential.scopes == (YOUTUBE_UPLOAD_SCOPE, YOUTUBE_READONLY_SCOPE, YOUTUBE_FORCE_SSL_SCOPE)
