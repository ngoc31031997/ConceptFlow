"""Unit tests for YouTubeVideoPublisher (business-rules.md Rule 3, 8).

Mocks googleapiclient.discovery.build and google.auth.transport so no
real Google API/network call is needed to run the test suite.
"""

from __future__ import annotations

from datetime import UTC, datetime, timedelta
from unittest.mock import MagicMock, patch

import pytest

from adapters.youtube.youtube_publisher import YouTubeVideoPublisher
from domain.errors import UploadError
from domain.models import OAuthCredential, PublishRequest
from domain.ports import CredentialStorePort


class FakeCredentialStore(CredentialStorePort):
    def __init__(self) -> None:
        self.saved: OAuthCredential | None = None

    def get(self) -> OAuthCredential | None:
        return self.saved

    def save(self, credential: OAuthCredential) -> None:
        self.saved = credential


def _credential(expires_in_seconds: int = 3600) -> OAuthCredential:
    return OAuthCredential(
        access_token="token",
        refresh_token="refresh",
        expires_at=datetime.now(UTC) + timedelta(seconds=expires_in_seconds),
        channel_id="UC123",
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
        publisher = YouTubeVideoPublisher("client-id", "client-secret", FakeCredentialStore())
        result = publisher.publish(_request(), _credential())

    assert result.youtube_video_url == "https://www.youtube.com/watch?v=abc123"


def test_publish_raises_upload_error_on_api_failure():
    mock_youtube = MagicMock()
    mock_youtube.videos.return_value.insert.return_value.execute.side_effect = RuntimeError("api down")

    with (
        patch("adapters.youtube.youtube_publisher.build", return_value=mock_youtube),
        patch("adapters.youtube.youtube_publisher.MediaFileUpload"),
    ):
        publisher = YouTubeVideoPublisher("client-id", "client-secret", FakeCredentialStore())
        with pytest.raises(UploadError):
            publisher.publish(_request(), _credential())


def test_publish_refreshes_expired_credential_before_upload():
    mock_youtube = MagicMock()
    mock_youtube.videos.return_value.insert.return_value.execute.return_value = {"id": "abc123"}
    store = FakeCredentialStore()

    def fake_refresh(self, request):
        self.token = "refreshed-token"
        self.expiry = datetime.now(UTC) + timedelta(hours=1)

    with (
        patch("adapters.youtube.youtube_publisher.build", return_value=mock_youtube),
        patch("adapters.youtube.youtube_publisher.MediaFileUpload"),
        patch("google.oauth2.credentials.Credentials.refresh", fake_refresh),
    ):
        publisher = YouTubeVideoPublisher("client-id", "client-secret", store)
        publisher.publish(_request(), _credential(expires_in_seconds=-10))

    assert store.saved is not None
    assert store.saved.access_token == "refreshed-token"


def test_publish_does_not_refresh_when_credential_still_valid():
    mock_youtube = MagicMock()
    mock_youtube.videos.return_value.insert.return_value.execute.return_value = {"id": "abc123"}
    store = FakeCredentialStore()

    with (
        patch("adapters.youtube.youtube_publisher.build", return_value=mock_youtube),
        patch("adapters.youtube.youtube_publisher.MediaFileUpload"),
        patch("google.oauth2.credentials.Credentials.refresh") as mock_refresh,
    ):
        publisher = YouTubeVideoPublisher("client-id", "client-secret", store)
        publisher.publish(_request(), _credential(expires_in_seconds=3600))

    mock_refresh.assert_not_called()
    assert store.saved is None
