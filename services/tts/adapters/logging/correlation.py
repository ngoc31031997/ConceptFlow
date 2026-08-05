"""Correlation ID propagation (interface-contracts.md).

Reads the X-Saga-ID header sent by the Rendering Service and attaches it to
every log line for the request, so a Saga's logs can be traced across units.
"""

from __future__ import annotations

import logging

SAGA_ID_HEADER = "X-Saga-ID"


def get_request_logger(saga_id: str | None) -> logging.LoggerAdapter:
    return logging.LoggerAdapter(logging.getLogger("tts_service"), {"saga_id": saga_id or "unknown"})
