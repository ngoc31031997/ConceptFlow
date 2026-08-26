"""PublishVideoUseCase — business-logic-model.md.

No batch wrapper needed — publish_video is already a single operation
per command (mirror Unit 6's AssembleVideoUseCase).
"""

from __future__ import annotations

import os

from domain.errors import InvalidPublishRequestError, MissingCredentialError
from domain.models import PublishRequest, PublishResult
from domain.ports import CredentialStorePort, VideoPublisherPort

VALID_VISIBILITIES = ("public", "unlisted", "private")


class PublishVideoUseCase:
    def __init__(self, publisher: VideoPublisherPort, credential_store: CredentialStorePort) -> None:
        self._publisher = publisher
        self._credential_store = credential_store

    def publish(self, request: PublishRequest) -> PublishResult:
        self._validate(request)

        credential = self._credential_store.get()
        if credential is None:
            raise MissingCredentialError("not authenticated — complete YouTube OAuth first (Story E1)")

        return self._publisher.publish(request, credential)

    @staticmethod
    def _validate(request: PublishRequest) -> None:
        """Zero-trust validation (Business Rule 1): does not trust that
        the GUI already validated this data (Business Rule 2)."""
        if not request.video_path or not os.path.isfile(request.video_path):
            raise InvalidPublishRequestError(f"missing video file: {request.video_path}")
        if not request.title or not request.title.strip():
            raise InvalidPublishRequestError("title is required")
        if request.visibility not in VALID_VISIBILITIES:
            raise InvalidPublishRequestError(
                f"visibility must be one of {VALID_VISIBILITIES}, got {request.visibility!r}"
            )
