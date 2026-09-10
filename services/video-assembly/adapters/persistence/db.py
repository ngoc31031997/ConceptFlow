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

-- CR-023 D1 — the channel's fixed intro/outro. Keyed by (kind, render_quality);
-- `superseded_at IS NULL` marks the currently active row for that pair, and a
-- new version is inserted (never updated in place) so a version history stays
-- around for the FR65.9 version counter. duration_seconds is ffprobe's
-- measured length of video_path, kept here so assemble_video.py never has to
-- probe it again per project.
CREATE TABLE IF NOT EXISTS channel_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind TEXT NOT NULL,
    render_quality TEXT NOT NULL,
    source_hash TEXT NOT NULL,
    video_path TEXT NOT NULL,
    music_path TEXT,
    version INT NOT NULL,
    duration_seconds DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    superseded_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_channel_assets_active
    ON channel_assets (kind, render_quality) WHERE superseded_at IS NULL;

-- CR-023 FR66.5 — the music bed is uploaded separately from the clip, so the
-- FR65.6 "same source, don't rebuild" cache needs its own hash: source_hash
-- stays the hash of the VIDEO source, music_source_hash is the hash of the
-- music file muxed into it. Sharing one column would make the next video
-- upload compare against a music hash and rebuild (or skip) wrongly.
ALTER TABLE channel_assets ADD COLUMN IF NOT EXISTS music_source_hash TEXT;
"""


async def create_pool() -> asyncpg.Pool:
    pool = await asyncpg.create_pool(DATABASE_URL)
    async with pool.acquire() as conn:
        await conn.execute(SCHEMA)
    return pool
