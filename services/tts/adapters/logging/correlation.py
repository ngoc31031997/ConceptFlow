"""Correlation ID propagation (interface-contracts.md).

Revision (ADR-0014): saga_id now comes from the AMQP command envelope,
not an HTTP header (TTS Service no longer serves REST).
"""

from __future__ import annotations

import logging

_correlation_id: str = "unknown"


def set_correlation_id(saga_id: str | None) -> None:
    global _correlation_id
    _correlation_id = saga_id or "unknown"


def get_request_logger(saga_id: str | None = None) -> logging.LoggerAdapter:
    return logging.LoggerAdapter(logging.getLogger("tts_service"), {"saga_id": saga_id or _correlation_id})
