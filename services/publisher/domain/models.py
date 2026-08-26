"""Domain value objects for the Publisher Service (module-structure.md)."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime


@dataclass(frozen=True)
class OAuthCredential:
    """Single-user OAuth credential (ADR-0016: plaintext, local-only threat model)."""

    access_token: str
    refresh_token: str
    expires_at: datetime
    channel_id: str


@dataclass(frozen=True)
class PublishRequest:
    """Input to publishing — video artifact plus YouTube metadata (Story E2)."""

    project_id: str
    video_path: str
    title: str
    description: str | None = None
    tags: list[str] = field(default_factory=list)
    visibility: str = ""


@dataclass(frozen=True)
class PublishResult:
    """Output of publishing, returned as the event payload.

    Deliberately minimal (Business Rule 5) — GUI only needs the URL to
    show the Creator a link to their published video.
    """

    youtube_video_url: str
