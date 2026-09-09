"""Unit tests for PublishVideoUseCase (business-rules.md Rule 1, 2, 4)."""

from __future__ import annotations

import os

import pytest

from application.publish_video import PublishVideoUseCase
from domain.errors import InvalidPublishRequestError, MissingCredentialError
from domain.models import OAuthCredential, PublishRequest, PublishResult
from domain.ports import VideoPublisherPort
from tests.fakes import InMemoryCredentialStore, make_credential


def _touch(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write("data")


class FakeVideoPublisher(VideoPublisherPort):
    def __init__(self) -> None:
        self.calls: list[PublishRequest] = []
        self.used_credential: OAuthCredential | None = None

    def publish(self, request: PublishRequest, credential: OAuthCredential) -> PublishResult:
        self.calls.append(request)
        self.used_credential = credential
        return PublishResult(youtube_video_url="https://www.youtube.com/watch?v=abc123")


def _store(credential: OAuthCredential | None = None) -> InMemoryCredentialStore:
    return InMemoryCredentialStore([credential] if credential is not None else [])


def _credential() -> OAuthCredential:
    return make_credential(channel_id="UC123")


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
    use_case = PublishVideoUseCase(publisher, _store(_credential()))

    result = use_case.publish(_request(video_path))

    assert result.youtube_video_url == "https://www.youtube.com/watch?v=abc123"
    assert len(publisher.calls) == 1


def test_missing_video_file_raises_invalid_publish_request_error(shared_volume_root):
    use_case = PublishVideoUseCase(FakeVideoPublisher(), _store(_credential()))

    with pytest.raises(InvalidPublishRequestError):
        use_case.publish(_request(str(shared_volume_root / "missing.mp4")))


def test_empty_title_raises_invalid_publish_request_error(shared_volume_root):
    video_path = str(shared_volume_root / "final.mp4")
    _touch(video_path)
    use_case = PublishVideoUseCase(FakeVideoPublisher(), _store(_credential()))

    with pytest.raises(InvalidPublishRequestError):
        use_case.publish(_request(video_path, title="  "))


def test_invalid_visibility_raises_invalid_publish_request_error(shared_volume_root):
    video_path = str(shared_volume_root / "final.mp4")
    _touch(video_path)
    use_case = PublishVideoUseCase(FakeVideoPublisher(), _store(_credential()))

    with pytest.raises(InvalidPublishRequestError):
        use_case.publish(_request(video_path, visibility="everyone"))


def test_missing_credential_raises_missing_credential_error(shared_volume_root):
    video_path = str(shared_volume_root / "final.mp4")
    _touch(video_path)
    use_case = PublishVideoUseCase(FakeVideoPublisher(), _store())

    with pytest.raises(MissingCredentialError):
        use_case.publish(_request(video_path))


def test_publishes_to_the_channel_named_in_the_request(shared_volume_root):
    """With several channels connected, the request's channel decides —
    not the default."""
    video_path = str(shared_volume_root / "final.mp4")
    _touch(video_path)
    default = make_credential(channel_id="UC_default", is_default=True)
    other = make_credential(channel_id="UC_other")
    publisher = FakeVideoPublisher()
    use_case = PublishVideoUseCase(publisher, InMemoryCredentialStore([default, other]))

    use_case.publish(_request(video_path, channel_id="UC_other"))

    assert publisher.used_credential.channel_id == "UC_other"


def test_unknown_channel_raises_rather_than_falling_back_to_the_default(shared_volume_root):
    """Publishing to a channel the Creator did not choose is public and
    cannot be undone, so a stale channel_id must fail loudly (FR32.3)."""
    video_path = str(shared_volume_root / "final.mp4")
    _touch(video_path)
    default = make_credential(channel_id="UC_default", is_default=True)
    use_case = PublishVideoUseCase(FakeVideoPublisher(), InMemoryCredentialStore([default]))

    with pytest.raises(MissingCredentialError) as exc_info:
        use_case.publish(_request(video_path, channel_id="UC_deleted"))

    assert "UC_deleted" in str(exc_info.value)


def test_no_channel_in_request_uses_the_default_channel(shared_volume_root):
    """Projects created before CR-012 carry no channel and must keep working."""
    video_path = str(shared_volume_root / "final.mp4")
    _touch(video_path)
    other = make_credential(channel_id="UC_other")
    default = make_credential(channel_id="UC_default", is_default=True)
    publisher = FakeVideoPublisher()
    use_case = PublishVideoUseCase(publisher, InMemoryCredentialStore([other, default]))

    use_case.publish(_request(video_path))

    assert publisher.used_credential.channel_id == "UC_default"
