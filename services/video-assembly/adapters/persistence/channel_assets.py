"""ChannelAssetsRepository — persistence for `channel_assets` (CR-023 D1).

Like InboxRepository/OutboxRepository, this is a thin, concrete asyncpg
wrapper (module-structure.md's persistence convention for this service) —
not something ever swapped out behind a port, unlike VideoAssemblerPort.
"""

from __future__ import annotations

from typing import Protocol

from domain.models import ChannelAsset


class Connection(Protocol):
    """Minimal surface of asyncpg's connection/transaction we depend on."""

    async def execute(self, query: str, *args: object) -> str: ...
    async def fetchrow(self, query: str, *args: object) -> object | None: ...


class ChannelAssetsRepository:
    def __init__(self, pool) -> None:
        self._pool = pool

    async def get_active(self, conn: Connection, kind: str, render_quality: str) -> ChannelAsset | None:
        """The row with `superseded_at IS NULL` for (kind, render_quality),
        or None when nothing has ever been registered for it."""
        row = await conn.fetchrow(
            "SELECT id, kind, render_quality, source_hash, video_path, music_path, "
            "version, duration_seconds, music_source_hash FROM channel_assets "
            "WHERE kind = $1 AND render_quality = $2 AND superseded_at IS NULL",
            kind,
            render_quality,
        )
        return _row_to_asset(row) if row is not None else None

    async def get_by_id(self, asset_id: str) -> ChannelAsset | None:
        """Resolves an opaque asset_id (as carried on an assemble_video
        command's intro_asset_id/outro_asset_id) to its real path — used
        outside any transaction, since it is a plain read for
        application/assemble_video.py."""
        async with self._pool.acquire() as conn:
            row = await conn.fetchrow(
                "SELECT id, kind, render_quality, source_hash, video_path, music_path, "
                "version, duration_seconds, music_source_hash FROM channel_assets WHERE id = $1",
                asset_id,
            )
        return _row_to_asset(row) if row is not None else None

    async def register_new_version(
        self,
        conn: Connection,
        *,
        kind: str,
        render_quality: str,
        source_hash: str,
        video_path: str,
        music_path: str | None,
        duration_seconds: float,
        music_source_hash: str | None = None,
    ) -> ChannelAsset:
        """Supersedes whatever was active for (kind, render_quality) and
        inserts the new row as the next version (FR65.9) — must be called
        with the same connection/transaction as the caller's Inbox/Outbox
        writes, so all three commit atomically (ADR-0013), the same
        discipline OutboxRepository.enqueue() already documents.
        """
        previous = await self.get_active(conn, kind, render_quality)
        next_version = previous.version + 1 if previous is not None else 1
        if previous is not None:
            await conn.execute(
                "UPDATE channel_assets SET superseded_at = now() WHERE id = $1", previous.id
            )
        row = await conn.fetchrow(
            "INSERT INTO channel_assets "
            "(kind, render_quality, source_hash, video_path, music_path, version, duration_seconds, "
            "music_source_hash) "
            "VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id",
            kind,
            render_quality,
            source_hash,
            video_path,
            music_path,
            next_version,
            duration_seconds,
            music_source_hash,
        )
        return ChannelAsset(
            id=str(row["id"]),
            kind=kind,
            render_quality=render_quality,
            source_hash=source_hash,
            video_path=video_path,
            music_path=music_path,
            version=next_version,
            duration_seconds=duration_seconds,
            music_source_hash=music_source_hash,
        )


def _row_to_asset(row) -> ChannelAsset:
    return ChannelAsset(
        id=str(row["id"]),
        kind=row["kind"],
        render_quality=row["render_quality"],
        source_hash=row["source_hash"],
        video_path=row["video_path"],
        music_path=row["music_path"],
        version=row["version"],
        duration_seconds=row["duration_seconds"] or 0.0,
        music_source_hash=row["music_source_hash"],
    )
