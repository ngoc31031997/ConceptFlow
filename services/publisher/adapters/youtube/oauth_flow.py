"""GoogleOAuthFlow — wraps google-auth-oauthlib's Authorization Code flow
(Low-Level Design Question 3).

Only this module talks to google-auth-oauthlib directly — application
layer depends on the OAuthFlowPort protocol (application/handle_oauth_callback.py).

CR-012: the flow is app-aware. Which OAuth app to use is chosen per
authorization (and recovered from `state` on the way back), because the
token exchange needs that specific app's client_secret.
"""

from __future__ import annotations

from datetime import UTC, datetime, timedelta

from google_auth_oauthlib.flow import Flow
from googleapiclient.discovery import build

from domain.models import OAuthApp, OAuthCredential

YOUTUBE_UPLOAD_SCOPE = "https://www.googleapis.com/auth/youtube.upload"
YOUTUBE_READONLY_SCOPE = "https://www.googleapis.com/auth/youtube.readonly"
# CR-015 FR40.1 — captions.insert has no narrower scope than this one, which
# also grants far more than captions (edit/delete video, playlists, comments).
# Accepted deliberately (ADR-0028): the OAuth app is in Testing, so adding it
# needs no Google re-verification, but every channel connected before this
# shipped has to be re-consented to actually receive it (FR40.2/40.3).
YOUTUBE_FORCE_SSL_SCOPE = "https://www.googleapis.com/auth/youtube.force-ssl"


class RedirectUriNotRegisteredError(Exception):
    """The configured redirect URI is absent from an app's redirect_uris.

    Raised before sending the Creator to Google, so they get an actionable
    message instead of Google's own redirect_uri_mismatch page, which does
    not say what to add or where (CR-012 FR30.2).
    """


class GoogleOAuthFlow:
    def __init__(self, redirect_uri: str) -> None:
        # One redirect URI for every app: it is a property of *this*
        # deployment, not of any particular Google client, so every app must
        # register the same value.
        self._redirect_uri = redirect_uri

    @property
    def redirect_uri(self) -> str:
        return self._redirect_uri

    def _new_flow(self, app: OAuthApp) -> Flow:
        client_config = {
            "web": {
                "client_id": app.client_id,
                "client_secret": app.client_secret,
                "auth_uri": "https://accounts.google.com/o/oauth2/auth",
                "token_uri": "https://oauth2.googleapis.com/token",
            }
        }
        flow = Flow.from_client_config(
            client_config,
            scopes=[YOUTUBE_UPLOAD_SCOPE, YOUTUBE_READONLY_SCOPE, YOUTUBE_FORCE_SSL_SCOPE],
        )
        flow.redirect_uri = self._redirect_uri
        return flow

    def build_authorization_url(self, app: OAuthApp, state: str | None = None) -> str:
        if not app.accepts_redirect(self._redirect_uri):
            raise RedirectUriNotRegisteredError(
                f"OAuth client '{app.label}' ({app.source_file}) does not list "
                f"{self._redirect_uri!r} among its authorized redirect URIs. "
                "Add that exact string under APIs & Services > Credentials > "
                "your OAuth client > Authorized redirect URIs in Google Cloud Console, "
                "then download the client secret file again."
            )

        flow = self._new_flow(app)
        authorization_url, _state = flow.authorization_url(
            access_type="offline",
            # "consent" alone re-shows the consent screen but reuses whichever
            # Google session is already signed in, so a second channel could
            # not be connected without signing out of Google first.
            # "select_account" is what surfaces the account/channel chooser
            # every time (CR-012 FR36.1).
            prompt="select_account consent",
            state=state,
        )
        return authorization_url

    def exchange_code(self, code: str, app: OAuthApp) -> OAuthCredential:
        flow = self._new_flow(app)
        flow.fetch_token(code=code)
        creds = flow.credentials

        channel_id, channel_title = self._fetch_channel(creds)
        expires_at = creds.expiry if creds.expiry is not None else datetime.now(UTC) + timedelta(hours=1)

        return OAuthCredential(
            access_token=creds.token,
            # Empty rather than None on a re-consent: the store treats ''
            # as "keep the stored one" (credential_store.save, FR31.6).
            refresh_token=creds.refresh_token or "",
            expires_at=expires_at,
            channel_id=channel_id,
            client_id=app.client_id,
            channel_title=channel_title,
            # CR-015 FR40.1: granted_scopes is what Google actually handed
            # back, which can be a strict subset of what _new_flow asked for
            # — the Creator can untick a scope on the consent screen. Storing
            # what was requested instead would make the FR40.2 pre-upload
            # check always pass, silently reintroducing the late-403 failure
            # that check exists to prevent.
            scopes=tuple(creds.granted_scopes or creds.scopes or ()),
        )

    @staticmethod
    def _fetch_channel(creds) -> tuple[str, str]:
        """Returns (channel_id, channel_title) for the single channel this
        token is bound to — the one picked on Google's chooser. mine=True
        never returns the account's other channels, which is why each
        channel needs its own consent (FR36.4)."""
        youtube = build("youtube", "v3", credentials=creds)
        response = youtube.channels().list(part="id,snippet", mine=True).execute()
        item = response["items"][0]
        return item["id"], item.get("snippet", {}).get("title", "")
