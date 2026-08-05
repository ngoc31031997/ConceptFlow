"""Correlation ID propagation utilities.

REST requests use X-Request-ID (generated if the caller doesn't send
one); AMQP messages use saga_id from the message envelope. Both are
attached to the logging context so every log line for a given
request/message carries its correlation ID (Low-Level Design,
Question 5).
"""

from __future__ import annotations

import logging
import uuid
from contextvars import ContextVar

_correlation_id: ContextVar[str] = ContextVar("correlation_id", default="-")


def set_correlation_id(value: str | None) -> str:
    resolved = value or str(uuid.uuid4())
    _correlation_id.set(resolved)
    return resolved


def get_correlation_id() -> str:
    return _correlation_id.get()


class CorrelationIdLogFilter(logging.Filter):
    """Injects the current correlation ID into every log record as
    `record.correlation_id`, for use in a logging Formatter."""

    def filter(self, record: logging.LogRecord) -> bool:
        record.correlation_id = get_correlation_id()
        return True
