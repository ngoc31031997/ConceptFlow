"""PostgreSQL connection pool + schema bootstrap (ADR-0013).

Bootstraps the Outbox/Inbox tables with a plain CREATE TABLE IF NOT
EXISTS at startup rather than a migration tool — appropriate at this
project's MVP scale (see ADR-0013's "Follow-ups").
"""

from __future__ import annotations

import os

import asyncpg

DATABASE_URL = os.environ["DATABASE_URL"]

SCHEMA = """
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_unpublished
    ON outbox_events (created_at) WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS processed_messages (
    message_id UUID PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
"""


async def create_pool() -> asyncpg.Pool:
    pool = await asyncpg.create_pool(DATABASE_URL)
    async with pool.acquire() as conn:
        await conn.execute(SCHEMA)
    return pool
