"""YouTubeVideoPublisher — implements VideoPublisherPort via
google-api-python-client's resumable upload (Low-Level Design Question 5).

Runs the upload in a ThreadPoolExecutor so it never blocks the FastAPI
event loop (google-api-python-client has no async API) — mirror Unit
3/5/6's threadpool+timeout pattern.
"""

from __future__ import annotations

import logging
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import TimeoutError as FutureTimeoutError
from datetime import UTC, datetime, timedelta

from google.auth.exceptions import RefreshError
from google.auth.transport.requests import Request as GoogleAuthRequest
from google.oauth2.credentials import Credentials as GoogleCredentials
from googleapiclient.discovery import build
from googleapiclient.http import MediaFileUpload

from domain.errors import UploadError
from domain.models import OAuthCredential, PublishRequest, PublishResult
from domain.ports import CredentialStorePort, VideoPublisherPort

logger = logging.getLogger(__name__)

DEFAULT_UPLOAD_TIMEOUT_SECONDS = 600
REFRESH_BUFFER_SECONDS = 60


class YouTubeVideoPublisher(VideoPublisherPort):
    def __init__(
        self,
        client_id: str,
        client_secret: str,
        credential_store: CredentialStorePort,
        timeout_seconds: int = DEFAULT_UPLOAD_TIMEOUT_SECONDS,
    ) -> None:
        self._client_id = client_id
        self._client_secret = client_secret
        self._credential_store = credential_store
        self._timeout_seconds = timeout_seconds
        self._executor = ThreadPoolExecutor(max_workers=2, thread_name_prefix="youtube-upload")

    def publish(self, request: PublishRequest, credential: OAuthCredential) -> PublishResult:
        credential = self._refresh_if_needed(credential)

        future = self._executor.submit(self._upload, request, credential)
        try:
            video_id = future.result(timeout=self._timeout_seconds)
        except FutureTimeoutError as exc:
            raise UploadError(f"YouTube upload timed out after {self._timeout_seconds}s") from exc
        except Exception as exc:  # noqa: BLE001 — any upload failure becomes a domain error
            logger.exception("YouTube upload failed")
            raise UploadError(str(exc)) from exc

        return PublishResult(youtube_video_url=f"https://www.youtube.com/watch?v={video_id}")

    def _refresh_if_needed(self, credential: OAuthCredential) -> OAuthCredential:
        """Proactive refresh (Business Rule 3): checks expires_at before
        calling the YouTube API, rather than waiting for a 401."""
        if datetime.now(UTC) < credential.expires_at - timedelta(seconds=REFRESH_BUFFER_SECONDS):
            return credential

        google_creds = self._to_google_credentials(credential)
        try:
            google_creds.refresh(GoogleAuthRequest())
        except RefreshError as exc:
            raise UploadError(f"failed to refresh access token: {exc}") from exc

        refreshed = OAuthCredential(
            access_token=google_creds.token,
            refresh_token=credential.refresh_token,
            expires_at=google_creds.expiry or datetime.now(UTC) + timedelta(hours=1),
            channel_id=credential.channel_id,
        )
        self._credential_store.save(refreshed)
        return refreshed

    def _to_google_credentials(self, credential: OAuthCredential) -> GoogleCredentials:
        return GoogleCredentials(
            token=credential.access_token,
            refresh_token=credential.refresh_token,
            token_uri="https://oauth2.googleapis.com/token",
            client_id=self._client_id,
            client_secret=self._client_secret,
        )

    def _upload(self, request: PublishRequest, credential: OAuthCredential) -> str:
        youtube = build("youtube", "v3", credentials=self._to_google_credentials(credential))
        body = {
            "snippet": {
                "title": request.title,
                "description": request.description or "",
                "tags": request.tags,
            },
            "status": {"privacyStatus": request.visibility},
        }
        media = MediaFileUpload(request.video_path, chunksize=-1, resumable=True)
        response = youtube.videos().insert(part="snippet,status", body=body, media_body=media).execute()
        return response["id"]
