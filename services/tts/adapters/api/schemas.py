"""Pydantic request/response models for the TTS Service REST API (interface-contracts.md)."""

from __future__ import annotations

from pydantic import BaseModel, Field


class SynthesizeRequestBody(BaseModel):
    project_id: str
    scene_index: int
    text: str
    language: str = Field(description='One of "vi" or "en"')


class SynthesizeResponseBody(BaseModel):
    audio_path: str
    duration_seconds: float


class ErrorResponseBody(BaseModel):
    error: str
    detail: str | None = None
    supported: list[str] | None = None
