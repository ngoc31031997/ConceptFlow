"""GoogleOAuthFlow — wraps google-auth-oauthlib's Authorization Code flow
(Low-Level Design Question 3).

Only this module talks to google-auth-oauthlib directly — application
layer depends on the OAuthFlowPort protocol (application/handle_oauth_callback.py).
"""

from __future__ import annotations

from datetime import UTC, datetime, timedelta

from google_auth_oauthlib.flow import Flow
from googleapiclient.discovery import build

from domain.models import OAuthCredential

YOUTUBE_UPLOAD_SCOPE = "https://www.googleapis.com/auth/youtube.upload"
YOUTUBE_READONLY_SCOPE = "https://www.googleapis.com/auth/youtube.readonly"


class GoogleOAuthFlow:
    def __init__(self, client_id: str, client_secret: str, redirect_uri: str) -> None:
        self._client_config = {
            "web": {
                "client_id": client_id,
                "client_secret": client_secret,
                "auth_uri": "https://accounts.google.com/o/oauth2/auth",
                "token_uri": "https://oauth2.googleapis.com/token",
            }
        }
        self._redirect_uri = redirect_uri

    def _new_flow(self) -> Flow:
        flow = Flow.from_client_config(
            self._client_config, scopes=[YOUTUBE_UPLOAD_SCOPE, YOUTUBE_READONLY_SCOPE]
        )
        flow.redirect_uri = self._redirect_uri
        return flow

    def build_authorization_url(self) -> str:
        flow = self._new_flow()
        authorization_url, _state = flow.authorization_url(access_type="offline", prompt="consent")
        return authorization_url

    def exchange_code(self, code: str) -> OAuthCredential:
        flow = self._new_flow()
        flow.fetch_token(code=code)
        creds = flow.credentials

        channel_id = self._fetch_channel_id(creds)
        expires_at = creds.expiry if creds.expiry is not None else datetime.now(UTC) + timedelta(hours=1)

        return OAuthCredential(
            access_token=creds.token,
            refresh_token=creds.refresh_token,
            expires_at=expires_at,
            channel_id=channel_id,
        )

    @staticmethod
    def _fetch_channel_id(creds) -> str:
        youtube = build("youtube", "v3", credentials=creds)
        response = youtube.channels().list(part="id", mine=True).execute()
        return response["items"][0]["id"]
