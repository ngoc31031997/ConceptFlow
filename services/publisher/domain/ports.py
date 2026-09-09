"""Abstract ports for the Publisher Service (module-structure.md, ADR-0002).

The application layer depends only on these abstractions, never on the
Google API client, the filesystem, or Postgres directly — this is what
lets the upload platform, the credential store, or the source of OAuth
client configuration be swapped later without touching business logic.
"""

from __future__ import annotations

from abc import ABC, abstractmethod

from domain.models import OAuthApp, OAuthCredential, PublishRequest, PublishResult


class VideoPublisherPort(ABC):
    @abstractmethod
    def publish(self, request: PublishRequest, credential: OAuthCredential) -> PublishResult:
        """Uploads request.video_path to YouTube with the given metadata,
        refreshing credential first if it has expired (Business Rule 3).

        Raises:
            domain.errors.UploadError: network/API failure or timeout.
        """


class OAuthAppRegistryPort(ABC):
    """Read-only catalogue of configured OAuth clients (ADR-0026 tier 1)."""

    @abstractmethod
    def list(self) -> list[OAuthApp]:
        """All usable apps, in a stable order for the GUI picker."""

    @abstractmethod
    def get(self, client_id: str) -> OAuthApp | None:
        """The app with this client_id, or None if it is not configured —
        e.g. a credential whose client_secret file has since been removed."""


class CredentialStorePort(ABC):
    """Store of consented YouTube channels (ADR-0026 tier 2)."""

    @abstractmethod
    def get(self, channel_id: str | None = None) -> OAuthCredential | None:
        """Returns the credential for channel_id, or the default channel
        when channel_id is None. None if there is no such channel (Story E1).
        """

    @abstractmethod
    def list(self) -> list[OAuthCredential]:
        """Every connected channel, default first then by channel title."""

    @abstractmethod
    def save(self, credential: OAuthCredential) -> None:
        """Upserts by channel_id (CR-012 FR31.5) — re-consenting a channel
        updates that channel's row and leaves every other channel alone.

        The first connected channel becomes the default; later ones do not
        silently steal it.
        """

    @abstractmethod
    def delete(self, channel_id: str) -> None:
        """Disconnects a channel. If it was the default, another connected
        channel is promoted so publishing without an explicit channel keeps
        working."""

    @abstractmethod
    def set_default(self, channel_id: str) -> None:
        """Makes channel_id the channel used when a publish request names none."""
