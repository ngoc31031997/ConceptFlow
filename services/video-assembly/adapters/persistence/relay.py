"""OutboxRelay — polling publisher (ADR-0013).

Background task: periodically reads outbox_events rows that haven't been
published yet, publishes each to RabbitMQ, then marks it published. This
is the only place that actually calls exchange.publish() for events
recorded via OutboxRepository — the consumer only ever writes to the
Outbox, never publishes directly.

A polling publisher was chosen over CDC/Debezium-style change capture:
simplest to run locally, no extra infrastructure, and more than fast
enough for this project's scale (see ADR-0013).
"""

from __future__ import annotations

import asyncio
import contextlib
import logging
from typing import Protocol

import asyncpg

logger = logging.getLogger(__name__)

POLL_INTERVAL_SECONDS = 1.0
BATCH_SIZE = 50


class PublishableExchange(Protocol):
    async def publish(self, message: object, routing_key: str) -> None: ...


class OutboxRelay:
    def __init__(
        self,
        pool: asyncpg.Pool,
        exchange: PublishableExchange,
        make_message,
        routing_key: str,
        poll_interval_seconds: float = POLL_INTERVAL_SECONDS,
    ) -> None:
        self._pool = pool
        self._exchange = exchange
        self._make_message = make_message
        self._routing_key = routing_key
        self._poll_interval_seconds = poll_interval_seconds
        self._task: asyncio.Task | None = None

    def start(self) -> None:
        self._task = asyncio.create_task(self._run())

    async def stop(self) -> None:
        if self._task is None:
            return
        self._task.cancel()
        with contextlib.suppress(asyncio.CancelledError):
            await self._task

    async def _run(self) -> None:
        while True:
            try:
                await self.relay_once()
            except Exception:  # noqa: BLE001 — keep the relay loop alive across transient DB/AMQP errors
                logger.exception("OutboxRelay poll failed")
            await asyncio.sleep(self._poll_interval_seconds)

    async def relay_once(self) -> int:
        """Publishes at most BATCH_SIZE unpublished rows; returns how many were relayed."""
        async with self._pool.acquire() as conn:
            rows = await conn.fetch(
                "SELECT id, payload FROM outbox_events "
                "WHERE published_at IS NULL ORDER BY created_at LIMIT $1",
                BATCH_SIZE,
            )
            for row in rows:
                body = row["payload"].encode("utf-8")
                message = self._make_message(body)
                await self._exchange.publish(message, routing_key=self._routing_key)
                await conn.execute("UPDATE outbox_events SET published_at = now() WHERE id = $1", row["id"])
            return len(rows)
