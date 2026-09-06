"""Pydantic request/response schemas for the REST API (adapters/api)."""

from pydantic import BaseModel


class OAuthCallbackResponse(BaseModel):
    connected: bool
    error: str | None = None
    state: str | None = None
