"""Domain value objects for the Publisher Service (module-structure.md)."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime


@dataclass(frozen=True)
class OAuthApp:
    """One Google OAuth client — i.e. one GCP project, i.e. one quota bucket
    (ADR-0026 tier 1).

    Sourced from a client_secret*.json file in the secrets directory. The
    number of apps controls throughput (each GCP project gets its own
    10,000 units/day, and videos.insert costs 1,600); the number of
    credentials controls how many channels can be published to. Those are
    two independent axes, which is why this is a separate type from
    OAuthCredential rather than fields on it.
    """

    client_id: str
    client_secret: str
    project_id: str
    redirect_uris: tuple[str, ...] = ()
    source_file: str = ""

    @property
    def label(self) -> str:
        """Human-facing name for the app picker — the GCP project id is what
        the Creator sees in Cloud Console, so it is what they can match on."""
        return self.project_id or self.client_id.split("-")[0]

    def accepts_redirect(self, redirect_uri: str) -> bool:
        """Google matches redirect URIs byte-for-byte, so this comparison is
        deliberately exact — a trailing-slash difference is a real mismatch,
        not a near miss (CR-012 FR30.1)."""
        return redirect_uri in self.redirect_uris


@dataclass(frozen=True)
class OAuthCredential:
    """One consented YouTube channel (ADR-0026 tier 2; ADR-0016: plaintext,
    local-only threat model).

    Keyed by channel_id rather than a fixed id=1 row: a Google account can
    own many channels (its personal one plus any number of Brand Accounts),
    and a token is bound to exactly the one channel picked at consent time,
    so "one Google account" is not a useful key (CR-012 FR36.2).

    client_id records which app minted this credential. That is not
    bookkeeping — a refresh token can only be refreshed with the very
    client_id/secret pair that issued it, so once more than one app exists
    a single global pair fails with invalid_client (CR-012 FR32.4).
    """

    access_token: str
    refresh_token: str
    expires_at: datetime
    channel_id: str
    client_id: str = ""
    channel_title: str = ""
    is_default: bool = False
    # CR-015 FR40 — the scopes Google actually granted at consent, not the
    # scopes the app asked for (a Creator can decline one on Google's
    # consent screen). Empty tuple means "granted before CR-015 shipped",
    # which is read as "youtube.upload only" — never as "has force-ssl too"
    # (ADR-0028). Deliberately not Optional: "no scopes recorded" and "we
    # checked and there are none" would otherwise be indistinguishable.
    scopes: tuple[str, ...] = ()


@dataclass(frozen=True)
class PublishRequest:
    """Input to publishing — video artifact plus YouTube metadata (Story E2)."""

    project_id: str
    video_path: str
    title: str
    description: str | None = None
    tags: list[str] = field(default_factory=list)
    visibility: str = ""
    publish_at: str | None = None  # RFC3339 — only valid alongside visibility == "private"
    thumbnail_path: str | None = None  # absolute path on shared_artifacts, from a manual upload
    channel_id: str | None = None  # None => the default channel (projects predating CR-012)
    # CR-015 FR39 — the .srt Video Assembly wrote when subtitle_mode asked
    # for a caption track. None when subtitles are off or burn-in only.
    caption_path: str | None = None
    # BCP-47 language the caption track is in (CR-015 FR39.3) — required
    # by captions.insert, and needed to tell YouTube what to auto-translate
    # FROM. Only meaningful alongside caption_path.
    caption_language: str | None = None


@dataclass(frozen=True)
class PublishResult:
    """Output of publishing, returned as the event payload.

    Deliberately minimal (Business Rule 5) — GUI only needs the URL to
    show the Creator a link to their published video.
    """

    youtube_video_url: str
    # CR-015 FR39.4 — unlike a failed thumbnail (visible the moment the
    # Creator opens YouTube), a failed or skipped caption upload is
    # otherwise silent, so it has to travel back through the result rather
    # than live only in a log line:
    #   None             — no caption_path was requested
    #   "uploaded"       — captions.insert succeeded
    #   "skipped_no_scope" — credential predates force-ssl re-consent (FR40.2)
    #   "failed"         — captions.insert was attempted and raised
    caption_status: str | None = None
