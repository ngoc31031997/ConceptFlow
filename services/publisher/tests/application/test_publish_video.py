"""Unit tests for PublishVideoUseCase (business-rules.md Rule 1, 2, 4)."""

from __future__ import annotations

import os
from datetime import UTC, datetime, timedelta

import pytest

from application.publish_video import PublishVideoUseCase
from domain.errors import InvalidPublishRequestError, MissingCredentialError
from domain.models import OAuthCredential, PublishRequest, PublishResult
from domain.ports import CredentialStorePort, VideoPublisherPort


def _touch(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write("data")


class FakeVideoPublisher(VideoPublisherPort):
    def __init__(self) -> None:
        self.calls: list[PublishRequest] = []

    def publish(self, request: PublishRequest, credential: OAuthCredential) -> PublishResult:
        self.calls.append(request)
        return PublishResult(youtube_video_url="https://www.youtube.com/watch?v=abc123")


class FakeCredentialStore(CredentialStorePort):
    def __init__(self, credential: OAuthCredential | None = None) -> None:
        self._credential = credential

    def get(self) -> OAuthCredential | None:
        return self._credential

    def save(self, credential: OAuthCredential) -> None:
        self._credential = credential


def _credential() -> OAuthCredential:
    return OAuthCredential(
        access_token="token",
        refresh_token="refresh",
        expires_at=datetime.now(UTC) + timedelta(hours=1),
        channel_id="UC123",
    )


@pytest.fixture
def shared_volume_root(tmp_path):
    return tmp_path


def _request(video_path: str, **overrides) -> PublishRequest:
    defaults = dict(
        project_id="proj-1",
        video_path=video_path,
        title="My Video",
        description="desc",
        tags=["python"],
        visibility="public",
    )
    defaults.update(overrides)
    return PublishRequest(**defaults)


def test_publishes_video_and_returns_result(shared_volume_root):
    video_path = str(shared_volume_root / "final.mp4")
    _touch(video_path)
    publisher = FakeVideoPublisher()
    use_case = PublishVideoUseCase(publisher, FakeCredentialStore(_credential()))

    result = use_case.publish(_request(video_path))

    assert result.youtube_video_url == "https://www.youtube.com/watch?v=abc123"
    assert len(publisher.calls) == 1


def test_missing_video_file_raises_invalid_publish_request_error(shared_volume_root):
    use_case = PublishVideoUseCase(FakeVideoPublisher(), FakeCredentialStore(_credential()))

    with pytest.raises(InvalidPublishRequestError):
        use_case.publish(_request(str(shared_volume_root / "missing.mp4")))


def test_empty_title_raises_invalid_publish_request_error(shared_volume_root):
    video_path = str(shared_volume_root / "final.mp4")
    _touch(video_path)
    use_case = PublishVideoUseCase(FakeVideoPublisher(), FakeCredentialStore(_credential()))

    with pytest.raises(InvalidPublishRequestError):
        use_case.publish(_request(video_path, title="  "))


def test_invalid_visibility_raises_invalid_publish_request_error(shared_volume_root):
    video_path = str(shared_volume_root / "final.mp4")
    _touch(video_path)
    use_case = PublishVideoUseCase(FakeVideoPublisher(), FakeCredentialStore(_credential()))

    with pytest.raises(InvalidPublishRequestError):
        use_case.publish(_request(video_path, visibility="everyone"))


def test_missing_credential_raises_missing_credential_error(shared_volume_root):
    video_path = str(shared_volume_root / "final.mp4")
    _touch(video_path)
    use_case = PublishVideoUseCase(FakeVideoPublisher(), FakeCredentialStore(None))

    with pytest.raises(MissingCredentialError):
        use_case.publish(_request(video_path))
