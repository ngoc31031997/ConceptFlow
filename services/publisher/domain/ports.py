"""Abstract ports for the Publisher Service (module-structure.md, ADR-0002).

The application layer depends only on these abstractions, never on the
Google API client or Postgres directly — this is what lets the upload
platform or credential store be swapped later without touching business
logic.
"""

from __future__ import annotations

from abc import ABC, abstractmethod

from domain.models import OAuthCredential, PublishRequest, PublishResult


class VideoPublisherPort(ABC):
    @abstractmethod
    def publish(self, request: PublishRequest, credential: OAuthCredential) -> PublishResult:
        """Uploads request.video_path to YouTube with the given metadata,
        refreshing credential first if it has expired (Business Rule 3).

        Raises:
            domain.errors.UploadError: network/API failure or timeout.
        """


class CredentialStorePort(ABC):
    @abstractmethod
    def get(self) -> OAuthCredential | None:
        """Returns the stored credential, or None if never authenticated (Story E1)."""

    @abstractmethod
    def save(self, credential: OAuthCredential) -> None:
        """Persists (upserts) the single-user OAuth credential."""
