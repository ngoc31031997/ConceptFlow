"""Domain-specific exceptions for the Publisher Service."""

from __future__ import annotations

VALID_VISIBILITIES = ("public", "unlisted", "private")


class InvalidPublishRequestError(Exception):
    """Raised when zero-trust validation of a publish_video request fails:
    missing/nonexistent video_path, empty title, or invalid visibility
    (Business Rule 1, 2, 4)."""


class MissingCredentialError(Exception):
    """Raised when publishing is attempted before OAuth authentication
    (Story E1) has completed."""


class UploadError(Exception):
    """Raised when the YouTube upload fails (network/API error) or times
    out after UPLOAD_TIMEOUT_SECONDS (Business Rule 8).

    A transient failure — mapped to a publish_failed event the
    Orchestrator/GUI may retry.
    """
