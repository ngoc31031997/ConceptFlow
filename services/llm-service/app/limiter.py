"""Shared request-rate limiter.

Hive's documented default ceiling is 5 requests per second. Chunks of one
video run in parallel and retries fire on their own schedule, so without a
shared limiter they would keep re-colliding on the same beat.
"""

from __future__ import annotations

import asyncio
import time


class RateLimiter:
    def __init__(self, per_second: float) -> None:
        if per_second <= 0:
            raise ValueError("per_second must be positive")
        self._interval = 1.0 / per_second
        self._next = 0.0
        self._lock = asyncio.Lock()

    async def acquire(self) -> None:
        async with self._lock:
            now = time.monotonic()
            wait = self._next - now
            self._next = max(now, self._next) + self._interval
        if wait > 0:
            await asyncio.sleep(wait)
