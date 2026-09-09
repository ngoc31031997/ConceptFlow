"""Shared in-memory test doubles for the Publisher Service ports.

CR-012 widened CredentialStorePort from two methods to five. Re-implementing
that surface in every test module invites the doubles to drift apart from
each other and from the real Postgres store, so they live here once.
"""

from __future__ import annotations

from datetime import UTC, datetime, timedelta

from domain.models import OAuthApp, OAuthCredential
from domain.ports import CredentialStorePort, OAuthAppRegistryPort

TEST_CLIENT_ID = "client-a.apps.googleusercontent.com"
TEST_REDIRECT_URI = "http://localhost:3000/oauth/youtube/callback"


def make_app(client_id: str = TEST_CLIENT_ID, **overrides) -> OAuthApp:
    defaults = dict(
        client_id=client_id,
        client_secret="secret-a",
        project_id="concer-508105",
        redirect_uris=(TEST_REDIRECT_URI,),
        source_file="client_secret_a.json",
    )
    defaults.update(overrides)
    return OAuthApp(**defaults)


def make_credential(
    channel_id: str = "UC123",
    client_id: str = TEST_CLIENT_ID,
    expires_in: timedelta = timedelta(hours=1),
    **overrides,
) -> OAuthCredential:
    defaults = dict(
        access_token="token",
        refresh_token="refresh",
        expires_at=datetime.now(UTC) + expires_in,
        channel_id=channel_id,
        client_id=client_id,
        channel_title=f"Channel {channel_id}",
        is_default=False,
    )
    defaults.update(overrides)
    return OAuthCredential(**defaults)


class FakeOAuthAppRegistry(OAuthAppRegistryPort):
    def __init__(self, apps: list[OAuthApp] | None = None) -> None:
        self._apps = list(apps) if apps is not None else [make_app()]

    def list(self) -> list[OAuthApp]:
        return list(self._apps)

    def get(self, client_id: str) -> OAuthApp | None:
        return next((app for app in self._apps if app.client_id == client_id), None)


class InMemoryCredentialStore(CredentialStorePort):
    """Mirrors the semantics the Postgres store implements in SQL: upsert by
    channel_id, first-connected-wins for the default flag, and a preserved
    refresh token when a re-consent supplies none."""

    def __init__(self, credentials: list[OAuthCredential] | None = None) -> None:
        self._by_channel: dict[str, OAuthCredential] = {}
        for credential in credentials or []:
            self._by_channel[credential.channel_id] = credential

    def get(self, channel_id: str | None = None) -> OAuthCredential | None:
        if channel_id is not None:
            return self._by_channel.get(channel_id)
        ordered = self.list()
        return ordered[0] if ordered else None

    def list(self) -> list[OAuthCredential]:
        return sorted(
            self._by_channel.values(),
            key=lambda c: (not c.is_default, c.channel_title, c.channel_id),
        )

    def save(self, credential: OAuthCredential) -> None:
        existing = self._by_channel.get(credential.channel_id)
        is_default = existing.is_default if existing else not self._by_channel
        refresh_token = credential.refresh_token or (existing.refresh_token if existing else "")
        self._by_channel[credential.channel_id] = OAuthCredential(
            access_token=credential.access_token,
            refresh_token=refresh_token,
            expires_at=credential.expires_at,
            channel_id=credential.channel_id,
            client_id=credential.client_id,
            channel_title=credential.channel_title,
            is_default=is_default,
        )

    def delete(self, channel_id: str) -> None:
        self._by_channel.pop(channel_id, None)
        if self._by_channel and not any(c.is_default for c in self._by_channel.values()):
            oldest = next(iter(self._by_channel))
            self.set_default(oldest)

    def set_default(self, channel_id: str) -> None:
        for key, credential in self._by_channel.items():
            self._by_channel[key] = OAuthCredential(
                access_token=credential.access_token,
                refresh_token=credential.refresh_token,
                expires_at=credential.expires_at,
                channel_id=credential.channel_id,
                client_id=credential.client_id,
                channel_title=credential.channel_title,
                is_default=key == channel_id,
            )
