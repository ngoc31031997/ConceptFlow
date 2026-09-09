"""PostgresCredentialStore — implements CredentialStorePort (ADR-0016, ADR-0026).

CredentialStorePort is synchronous (domain/ports.py) so PublishVideoUseCase
and YouTubeVideoPublisher's internal token-refresh persistence can call it
directly without an event loop — both run inside a worker thread via
asyncio.to_thread (mirror Unit 6's AssembleVideoUseCase), where the
asyncpg pool (bound to the main event loop) cannot be awaited safely.
psycopg2 (a plain blocking driver) sidesteps that loop-binding problem for
this one small, low-frequency table — the Inbox/Outbox tables stay on
asyncpg since those are only ever accessed from async code.

CR-012: multi-channel. Rows are keyed by channel_id, so connecting a
second channel adds a row instead of overwriting the first.
"""

from __future__ import annotations

import psycopg2

from domain.models import OAuthCredential
from domain.ports import CredentialStorePort

_COLUMNS = (
    "channel_id, channel_title, client_id, access_token, refresh_token, expires_at, is_default"
)


class PostgresCredentialStore(CredentialStorePort):
    def __init__(self, database_url: str) -> None:
        self._database_url = database_url

    def get(self, channel_id: str | None = None) -> OAuthCredential | None:
        if channel_id is None:
            # Fall back to any single connected channel when none is flagged
            # default — a store holding exactly one channel should publish,
            # not report "not authenticated" over a bookkeeping detail.
            query = (
                f"SELECT {_COLUMNS} FROM youtube_accounts "
                "ORDER BY is_default DESC, created_at ASC LIMIT 1"
            )
            params: tuple = ()
        else:
            query = f"SELECT {_COLUMNS} FROM youtube_accounts WHERE channel_id = %s"
            params = (channel_id,)

        with psycopg2.connect(self._database_url) as conn, conn.cursor() as cur:
            cur.execute(query, params)
            row = cur.fetchone()
        return _to_credential(row) if row is not None else None

    def list(self) -> list[OAuthCredential]:
        query = (
            f"SELECT {_COLUMNS} FROM youtube_accounts "
            "ORDER BY is_default DESC, channel_title ASC, channel_id ASC"
        )
        with psycopg2.connect(self._database_url) as conn, conn.cursor() as cur:
            cur.execute(query)
            rows = cur.fetchall()
        return [_to_credential(row) for row in rows]

    def save(self, credential: OAuthCredential) -> None:
        """Upsert by channel_id (FR31.5).

        Two details this statement has to get right, both of which are
        silent data loss if missed:

        - refresh_token: Google only returns one on the *first* consent for
          a given user/client pair. A re-consent that omits it must keep the
          stored one, or the channel becomes unrefreshable (FR31.6).
        - is_default: the first channel connected becomes the default;
          later ones must not steal the flag from the Creator's choice.
        """
        query = f"""
            INSERT INTO youtube_accounts ({_COLUMNS}, updated_at)
            VALUES (
                %s, %s, %s, %s, %s, %s,
                NOT EXISTS (SELECT 1 FROM youtube_accounts),
                now()
            )
            ON CONFLICT (channel_id) DO UPDATE SET
                channel_title = EXCLUDED.channel_title,
                client_id = EXCLUDED.client_id,
                access_token = EXCLUDED.access_token,
                refresh_token = COALESCE(
                    NULLIF(EXCLUDED.refresh_token, ''), youtube_accounts.refresh_token
                ),
                expires_at = EXCLUDED.expires_at,
                updated_at = now()
        """
        params = (
            credential.channel_id,
            credential.channel_title,
            credential.client_id,
            credential.access_token,
            credential.refresh_token,
            credential.expires_at,
        )
        with psycopg2.connect(self._database_url) as conn, conn.cursor() as cur:
            cur.execute(query, params)
            conn.commit()

    def delete(self, channel_id: str) -> None:
        """Deletes, then promotes another channel if the default was removed.

        Done in one transaction so there is never a window where channels
        exist but none is default — publishing without an explicit channel
        would fail during that window.
        """
        with psycopg2.connect(self._database_url) as conn, conn.cursor() as cur:
            cur.execute("DELETE FROM youtube_accounts WHERE channel_id = %s", (channel_id,))
            cur.execute(
                """
                UPDATE youtube_accounts SET is_default = TRUE, updated_at = now()
                WHERE channel_id = (
                    SELECT channel_id FROM youtube_accounts ORDER BY created_at ASC LIMIT 1
                )
                AND NOT EXISTS (SELECT 1 FROM youtube_accounts WHERE is_default)
                """
            )
            conn.commit()

    def set_default(self, channel_id: str) -> None:
        """Clears the old default before setting the new one — the partial
        unique index in db.py rejects the pair existing at once, so the
        order here is load-bearing, not stylistic."""
        with psycopg2.connect(self._database_url) as conn, conn.cursor() as cur:
            cur.execute(
                "UPDATE youtube_accounts SET is_default = FALSE, updated_at = now() "
                "WHERE is_default AND channel_id <> %s",
                (channel_id,),
            )
            cur.execute(
                "UPDATE youtube_accounts SET is_default = TRUE, updated_at = now() "
                "WHERE channel_id = %s",
                (channel_id,),
            )
            conn.commit()


def _to_credential(row: tuple) -> OAuthCredential:
    channel_id, channel_title, client_id, access_token, refresh_token, expires_at, is_default = row
    return OAuthCredential(
        access_token=access_token,
        refresh_token=refresh_token,
        expires_at=expires_at,
        channel_id=channel_id,
        client_id=client_id,
        channel_title=channel_title,
        is_default=is_default,
    )
