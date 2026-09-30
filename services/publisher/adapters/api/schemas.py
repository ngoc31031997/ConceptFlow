"""Pydantic request/response schemas for the REST API (adapters/api)."""

from pydantic import BaseModel


class OAuthAppResponse(BaseModel):
    """One configured OAuth client, as offered to the Creator.

    client_secret is deliberately absent — this shape is the only thing
    that reaches the browser, so the secret cannot leak through it.
    """

    client_id: str
    label: str
    project_id: str
    source_file: str
    redirect_ok: bool
    redirect_uri_hint: str | None = None


class YouTubeAccountResponse(BaseModel):
    """One connected YouTube channel."""

    channel_id: str
    channel_title: str
    client_id: str
    app_label: str
    is_default: bool
    # A channel connected without the force-ssl scope
    # cannot receive a caption track; surfaced here so the Creator learns
    # that from the channel picker, not from a video that quietly has no CC.
    has_caption_scope: bool = False


class OAuthCallbackResponse(BaseModel):
    connected: bool
    error: str | None = None
    state: str | None = None
    channel_id: str | None = None
    channel_title: str | None = None


class OAuthStatusResponse(BaseModel):
    # `connected` is kept alongside the account list so a caller asking "is
    # anything set up?" has a single boolean to read.
    connected: bool
    accounts: list[YouTubeAccountResponse] = []
