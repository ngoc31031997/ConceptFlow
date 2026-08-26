"""PostgresCredentialStore — implements CredentialStorePort (ADR-0016).

CredentialStorePort is synchronous (domain/ports.py) so PublishVideoUseCase
and YouTubeVideoPublisher's internal token-refresh persistence can call it
directly without an event loop — both run inside a worker thread via
asyncio.to_thread (mirror Unit 6's AssembleVideoUseCase), where the
asyncpg pool (bound to the main event loop) cannot be awaited safely.
psycopg2 (a plain blocking driver) sidesteps that loop-binding problem for
this one small, low-frequency table — the Inbox/Outbox tables stay on
asyncpg since those are only ever accessed from async code.

Single-user: oauth_credentials always has at most 1 row (id=1, enforced
by a CHECK constraint in db.py's schema), upserted on every save().
"""

from __future__ import annotations

import psycopg2

from domain.models import OAuthCredential
from domain.ports import CredentialStorePort


class PostgresCredentialStore(CredentialStorePort):
    def __init__(self, database_url: str) -> None:
        self._database_url = database_url

    def get(self) -> OAuthCredential | None:
        query = (
            "SELECT access_token, refresh_token, expires_at, channel_id "
            "FROM oauth_credentials WHERE id = 1"
        )
        with psycopg2.connect(self._database_url) as conn, conn.cursor() as cur:
            cur.execute(query)
            row = cur.fetchone()
        if row is None:
            return None
        access_token, refresh_token, expires_at, channel_id = row
        return OAuthCredential(
            access_token=access_token,
            refresh_token=refresh_token,
            expires_at=expires_at,
            channel_id=channel_id,
        )

    def save(self, credential: OAuthCredential) -> None:
        query = """
            INSERT INTO oauth_credentials
                (id, access_token, refresh_token, expires_at, channel_id, updated_at)
            VALUES (1, %s, %s, %s, %s, now())
            ON CONFLICT (id) DO UPDATE SET
                access_token = EXCLUDED.access_token,
                refresh_token = EXCLUDED.refresh_token,
                expires_at = EXCLUDED.expires_at,
                channel_id = EXCLUDED.channel_id,
                updated_at = now()
        """
        params = (
            credential.access_token,
            credential.refresh_token,
            credential.expires_at,
            credential.channel_id,
        )
        with psycopg2.connect(self._database_url) as conn, conn.cursor() as cur:
            cur.execute(query, params)
            conn.commit()
