"""PostgreSQL connection pool + schema bootstrap (ADR-0013, ADR-0016, ADR-0026).

Bootstraps the Outbox/Inbox tables plus youtube_accounts with a plain
CREATE TABLE IF NOT EXISTS at startup rather than a migration tool —
appropriate at this project's MVP scale (see ADR-0013's "Follow-ups").

CR-012 replaced the single-row oauth_credentials table (which had a
CHECK (id = 1) constraint, so connecting a second channel silently
overwrote the first) with youtube_accounts, keyed by channel_id. The old
table is left in place rather than dropped: it is the only copy of the
Creator's existing refresh token until the migration below has run, and
keeping it costs nothing.
"""

from __future__ import annotations

import logging
import os

import asyncpg

logger = logging.getLogger(__name__)

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

CREATE TABLE IF NOT EXISTS oauth_credentials (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    channel_id TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS youtube_accounts (
    channel_id TEXT PRIMARY KEY,
    channel_title TEXT NOT NULL DEFAULT '',
    client_id TEXT NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- At most one default channel. A partial unique index rather than
-- application-level checking, so a concurrent save() cannot leave the
-- Creator with two "default" channels and a coin-flip as to which one
-- publishes (CR-012 FR31.2).
CREATE UNIQUE INDEX IF NOT EXISTS idx_youtube_accounts_single_default
    ON youtube_accounts (is_default) WHERE is_default;
"""

# One-shot data migration (CR-012 FR31.3). Guarded by NOT EXISTS rather
# than a version table: it must be a no-op on every start after the first,
# and must never clobber a channel the Creator has since re-connected.
MIGRATE_LEGACY_CREDENTIAL = """
INSERT INTO youtube_accounts
    (channel_id, channel_title, client_id, access_token, refresh_token, expires_at, is_default)
SELECT channel_id, '', $1, access_token, refresh_token, expires_at, TRUE
FROM oauth_credentials
WHERE id = 1
  AND NOT EXISTS (SELECT 1 FROM youtube_accounts)
ON CONFLICT (channel_id) DO NOTHING
"""


async def create_pool() -> asyncpg.Pool:
    pool = await asyncpg.create_pool(DATABASE_URL)
    async with pool.acquire() as conn:
        await conn.execute(SCHEMA)
        await _migrate_legacy_credential(conn)
    return pool


async def _migrate_legacy_credential(conn: asyncpg.Connection) -> None:
    """Carries the pre-CR-012 single credential into youtube_accounts so the
    Creator does not have to re-connect the channel they are already using.

    The legacy row predates per-credential client ids, so it is attributed
    to whichever client the old env vars named — that is by construction the
    app that issued it, and the refresh token only works with that pair.
    """
    legacy_client_id = os.environ.get("GOOGLE_OAUTH_CLIENT_ID", "")
    if not legacy_client_id:
        # Without the issuing client id the migrated refresh token could
        # never be refreshed, so a migrated row would be worse than none.
        return

    result = await conn.execute(MIGRATE_LEGACY_CREDENTIAL, legacy_client_id)
    if result != "INSERT 0 0":
        logger.info("Migrated the pre-CR-012 OAuth credential into youtube_accounts")
