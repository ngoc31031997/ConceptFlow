"""InboxRepository — durable message dedupe (ADR-0013).

Replaces the previous in-memory IdempotencyStore: the processed_messages
table survives a restart, so a redelivered message is still recognized
as already-handled even if the service crashed and restarted in between.
"""

from __future__ import annotations

from typing import Protocol

import asyncpg


class Connection(Protocol):
    """Minimal surface of asyncpg's connection/transaction we depend on."""

    async def execute(self, query: str, *args: object) -> str: ...
    async def fetchrow(self, query: str, *args: object) -> object | None: ...


class InboxRepository:
    def __init__(self, pool: asyncpg.Pool) -> None:
        self._pool = pool

    async def has_processed(self, message_id: str) -> bool:
        async with self._pool.acquire() as conn:
            row = await conn.fetchrow(
                "SELECT 1 FROM processed_messages WHERE message_id = $1", message_id
            )
        return row is not None

    async def mark_processed(self, conn: Connection, message_id: str) -> None:
        """Must be called with the same connection/transaction used for the
        corresponding OutboxRepository.enqueue() call, so both writes commit
        atomically."""
        await conn.execute(
            "INSERT INTO processed_messages (message_id) VALUES ($1) ON CONFLICT DO NOTHING",
            message_id,
        )
