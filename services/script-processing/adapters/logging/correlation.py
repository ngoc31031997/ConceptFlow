"""Correlation ID propagation (interface-contracts.md).

saga_id comes from the AMQP command envelope — this service has no REST
endpoint (ADR-0012).
"""

from __future__ import annotations

import logging

_correlation_id: str = "unknown"


def set_correlation_id(saga_id: str | None) -> None:
    global _correlation_id
    _correlation_id = saga_id or "unknown"


def get_request_logger(saga_id: str | None = None) -> logging.LoggerAdapter:
    return logging.LoggerAdapter(
        logging.getLogger("script_processing_service"), {"saga_id": saga_id or _correlation_id}
    )
