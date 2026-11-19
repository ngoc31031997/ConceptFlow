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

CREATE TABLE IF NOT EXISTS illustration_folders (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0,
    is_system BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS illustrations (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    folder_id TEXT NOT NULL REFERENCES illustration_folders (id),
    tags TEXT[] NOT NULL DEFAULT '{}',
    description TEXT NOT NULL DEFAULT '',
    usage TEXT NOT NULL DEFAULT '',
    code TEXT NOT NULL DEFAULT '',
    builtin BOOLEAN NOT NULL DEFAULT false,
    status TEXT NOT NULL DEFAULT 'draft',
    version INTEGER NOT NULL DEFAULT 1,
    kind TEXT NOT NULL DEFAULT 'figure',
    detail_level TEXT NOT NULL DEFAULT 'medium',
    ref_id TEXT NOT NULL DEFAULT '',
    exemplar BOOLEAN NOT NULL DEFAULT false,
    source_id TEXT REFERENCES illustrations (id) ON DELETE SET NULL,
    home_folder_id TEXT REFERENCES illustration_folders (id) ON DELETE SET NULL,
    warnings JSONB NOT NULL DEFAULT '[]',
    preview_version INTEGER NOT NULL DEFAULT 0,
    preview_png_path TEXT NOT NULL DEFAULT '',
    preview_gif_path TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT illustrations_name_key UNIQUE (name)
);

CREATE INDEX IF NOT EXISTS idx_illustrations_folder ON illustrations (folder_id);
"""


async def create_pool() -> asyncpg.Pool:
    pool = await asyncpg.create_pool(DATABASE_URL)
    async with pool.acquire() as conn:
        await conn.execute(SCHEMA)
    return pool
