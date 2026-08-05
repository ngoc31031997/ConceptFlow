"""In-memory idempotency store (NFR Design: TTL 24h, resets on restart).

Deliberately simple: a dict of message_id -> expiry timestamp. A
background task periodically purges expired entries so the set doesn't
grow unbounded. Losing this state on restart is an accepted trade-off
(Rule 5 / NFR Design) because classify_scenes has no durable
side-effect to duplicate beyond an extra event publish.
"""

from __future__ import annotations

import time

_DEFAULT_TTL_SECONDS = 24 * 60 * 60


class IdempotencyStore:
    def __init__(self, ttl_seconds: int = _DEFAULT_TTL_SECONDS) -> None:
        self._ttl_seconds = ttl_seconds
        self._seen: dict[str, float] = {}

    def already_processed(self, message_id: str) -> bool:
        self._purge_expired()
        return message_id in self._seen

    def mark_processed(self, message_id: str) -> None:
        self._seen[message_id] = time.monotonic() + self._ttl_seconds

    def _purge_expired(self) -> None:
        now = time.monotonic()
        expired = [mid for mid, expiry in self._seen.items() if expiry <= now]
        for mid in expired:
            del self._seen[mid]
