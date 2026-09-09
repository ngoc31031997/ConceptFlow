"""Unit tests for YouTubeVideoPublisher (business-rules.md Rule 3, 8).

Mocks googleapiclient.discovery.build and google.auth.transport so no
real Google API/network call is needed to run the test suite.
"""

from __future__ import annotations

from datetime import UTC, datetime, timedelta
from unittest.mock import MagicMock, patch

import pytest
from google.oauth2.credentials import Credentials as GoogleCredentials

from adapters.youtube.youtube_publisher import YouTubeVideoPublisher
from domain.errors import UploadError
from domain.models import OAuthCredential, PublishRequest
from tests.fakes import (
    FakeOAuthAppRegistry,
    InMemoryCredentialStore,
    make_app,
    make_credential,
)


def _credential(expires_in_seconds: int = 3600) -> OAuthCredential:
    return make_credential(
        channel_id="UC123", expires_in=timedelta(seconds=expires_in_seconds)
    )


def _request(video_path: str = "final.mp4") -> PublishRequest:
    return PublishRequest(
        project_id="proj-1",
        video_path=video_path,
        title="My Video",
        description="desc",
        tags=["python"],
        visibility="public",
    )


def test_publish_returns_video_url_on_success():
    mock_youtube = MagicMock()
    mock_youtube.videos.return_value.insert.return_value.execute.return_value = {"id": "abc123"}

    with (
        patch("adapters.youtube.youtube_publisher.build", return_value=mock_youtube),
        patch("adapters.youtube.youtube_publisher.MediaFileUpload"),
    ):
        publisher = YouTubeVideoPublisher(FakeOAuthAppRegistry(), InMemoryCredentialStore())
        result = publisher.publish(_request(), _credential())

    assert result.youtube_video_url == "https://www.youtube.com/watch?v=abc123"


def test_publish_raises_upload_error_on_api_failure():
    mock_youtube = MagicMock()
    mock_youtube.videos.return_value.insert.return_value.execute.side_effect = RuntimeError("api down")

    with (
        patch("adapters.youtube.youtube_publisher.build", return_value=mock_youtube),
        patch("adapters.youtube.youtube_publisher.MediaFileUpload"),
    ):
        publisher = YouTubeVideoPublisher(FakeOAuthAppRegistry(), InMemoryCredentialStore())
        with pytest.raises(UploadError):
            publisher.publish(_request(), _credential())


def test_publish_refreshes_expired_credential_before_upload():
    mock_youtube = MagicMock()
    mock_youtube.videos.return_value.insert.return_value.execute.return_value = {"id": "abc123"}
    store = InMemoryCredentialStore()

    def fake_refresh(self, request):
        self.token = "refreshed-token"
        self.expiry = datetime.now(UTC) + timedelta(hours=1)

    with (
        patch("adapters.youtube.youtube_publisher.build", return_value=mock_youtube),
        patch("adapters.youtube.youtube_publisher.MediaFileUpload"),
        patch("google.oauth2.credentials.Credentials.refresh", fake_refresh),
    ):
        publisher = YouTubeVideoPublisher(FakeOAuthAppRegistry(), store)
        publisher.publish(_request(), _credential(expires_in_seconds=-10))

    stored = store.get("UC123")
    assert stored is not None
    assert stored.access_token == "refreshed-token"


def test_publish_does_not_refresh_when_credential_still_valid():
    mock_youtube = MagicMock()
    mock_youtube.videos.return_value.insert.return_value.execute.return_value = {"id": "abc123"}
    store = InMemoryCredentialStore()

    with (
        patch("adapters.youtube.youtube_publisher.build", return_value=mock_youtube),
        patch("adapters.youtube.youtube_publisher.MediaFileUpload"),
        patch("google.oauth2.credentials.Credentials.refresh") as mock_refresh,
    ):
        publisher = YouTubeVideoPublisher(FakeOAuthAppRegistry(), store)
        publisher.publish(_request(), _credential(expires_in_seconds=3600))

    mock_refresh.assert_not_called()
    assert store.get("UC123") is None


def test_refreshes_with_the_client_that_issued_the_credential():
    """A refresh token is only valid against the exact client_id/secret pair
    that minted it, so with several apps configured the publisher must look
    up the credential's own app (CR-012 FR32.4)."""
    mock_youtube = MagicMock()
    mock_youtube.videos.return_value.insert.return_value.execute.return_value = {"id": "abc123"}
    store = InMemoryCredentialStore()
    registry = FakeOAuthAppRegistry([
        make_app(),
        make_app(client_id="client-b", client_secret="secret-b"),
    ])
    captured = {}

    real_init = GoogleCredentials.__init__

    def capturing_init(self, *args, **kwargs):
        captured.update(kwargs)
        real_init(self, *args, **kwargs)

    with (
        patch("adapters.youtube.youtube_publisher.build", return_value=mock_youtube),
        patch("adapters.youtube.youtube_publisher.MediaFileUpload"),
        patch.object(GoogleCredentials, "__init__", capturing_init),
    ):
        publisher = YouTubeVideoPublisher(registry, store)
        publisher.publish(_request(), make_credential(channel_id="UC_b", client_id="client-b"))

    assert captured["client_id"] == "client-b"
    assert captured["client_secret"] == "secret-b"


def test_upload_error_names_the_missing_client_when_its_secret_file_is_gone():
    """A channel outlives the client_secret file it was connected with; the
    failure has to say what to restore rather than surfacing Google's
    invalid_client."""
    store = InMemoryCredentialStore()
    publisher = YouTubeVideoPublisher(FakeOAuthAppRegistry([]), store)

    with pytest.raises(UploadError) as exc_info:
        publisher.publish(_request(), make_credential(client_id="client-gone"))

    assert "client-gone" in str(exc_info.value)
