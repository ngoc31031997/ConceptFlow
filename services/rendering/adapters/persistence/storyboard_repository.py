"""PostgreSQL cho Storyboard: khung đã vẽ `storyboard_shots`, bản chốt `storyboard_versions`,
và xoá cả dự án."""

from __future__ import annotations

import json

import asyncpg

from domain.illustration import format_time
from domain.storyboard import ShotFrame, StoryboardVersion, VersionConflictError

SHOT_COLUMNS = "project_id, shot_id, unit_id, beat_id, frame_path, frame_hash, spec_hash, drawn_at, error"
VERSION_COLUMNS = "project_id, version, document, sequence_version, note, locked_at"
# Bảng của dự án bị xoá khi xoá Storyboard (tài sản và bộ đếm mã của chúng, khung, bản chốt).
PROJECT_TABLES = ("project_assets", "project_asset_counters", "storyboard_shots", "storyboard_versions")


def _to_shot(row: asyncpg.Record) -> ShotFrame:
    project_id, shot_id, unit_id, beat_id, frame_path, frame_hash, spec_hash, drawn_at, error = tuple(row)
    return ShotFrame(
        project_id=project_id, shot_id=shot_id, unit_id=unit_id, beat_id=beat_id, frame_path=frame_path,
        frame_hash=frame_hash, spec_hash=spec_hash, drawn_at=format_time(drawn_at), error=error)


def _to_version(row: asyncpg.Record) -> StoryboardVersion:
    project_id, version, document, sequence_version, note, locked_at = tuple(row)
    return StoryboardVersion(
        project_id=project_id, version=version,
        document=json.loads(document) if isinstance(document, str) else document,
        sequence_version=sequence_version, note=note, locked_at=format_time(locked_at))


class StoryboardRepository:
    def __init__(self, pool: asyncpg.Pool) -> None:
        self._pool = pool

    async def get_shot(self, project_id: str, shot_id: str) -> ShotFrame | None:
        row = await self._pool.fetchrow(
            f"SELECT {SHOT_COLUMNS} FROM storyboard_shots WHERE project_id = $1 AND shot_id = $2",
            project_id, shot_id)
        return _to_shot(row) if row else None

    async def upsert_shot(self, shot: ShotFrame) -> ShotFrame:
        row = await self._pool.fetchrow(
            f"""
            INSERT INTO storyboard_shots (project_id, shot_id, unit_id, beat_id, frame_path, frame_hash,
                spec_hash, drawn_at, error)
            VALUES ($1, $2, $3, $4, $5, $6, $7, now(), $8)
            ON CONFLICT (project_id, shot_id) DO UPDATE SET unit_id = EXCLUDED.unit_id,
                beat_id = EXCLUDED.beat_id, frame_path = EXCLUDED.frame_path,
                frame_hash = EXCLUDED.frame_hash, spec_hash = EXCLUDED.spec_hash,
                drawn_at = now(), error = EXCLUDED.error
            RETURNING {SHOT_COLUMNS}
            """, shot.project_id, shot.shot_id, shot.unit_id, shot.beat_id, shot.frame_path,
            shot.frame_hash, shot.spec_hash, shot.error)
        return _to_shot(row)

    async def next_version(self, project_id: str) -> int:
        """Số phiên bản kế tiếp: lớn nhất đang có cộng một (1 khi chưa có)."""
        return await self._pool.fetchval(
            "SELECT COALESCE(MAX(version), 0) + 1 FROM storyboard_versions WHERE project_id = $1", project_id)

    async def insert_version(self, version: StoryboardVersion) -> StoryboardVersion:
        """Thêm bản chốt; số phiên bản đã có thì ném VersionConflictError. Không có đường sửa hay xoá."""
        try:
            row = await self._pool.fetchrow(
                f"""
                INSERT INTO storyboard_versions (project_id, version, document, sequence_version, note)
                VALUES ($1, $2, $3::jsonb, $4, $5) RETURNING {VERSION_COLUMNS}
                """, version.project_id, version.version,
                json.dumps(version.document, ensure_ascii=False), version.sequence_version, version.note)
        except asyncpg.UniqueViolationError as exc:
            raise VersionConflictError(version.version) from exc
        return _to_version(row)

    async def latest_version(self, project_id: str) -> StoryboardVersion | None:
        row = await self._pool.fetchrow(
            f"SELECT {VERSION_COLUMNS} FROM storyboard_versions WHERE project_id = $1 "
            "ORDER BY version DESC LIMIT 1", project_id)
        return _to_version(row) if row else None

    async def get_version(self, project_id: str, version: int) -> StoryboardVersion | None:
        row = await self._pool.fetchrow(
            f"SELECT {VERSION_COLUMNS} FROM storyboard_versions WHERE project_id = $1 AND version = $2",
            project_id, version)
        return _to_version(row) if row else None

    async def delete_project(self, project_id: str) -> None:
        """Xoá mọi hàng Storyboard của dự án trong một giao dịch; không có hàng nào thì không phải lỗi."""
        async with self._pool.acquire() as conn, conn.transaction():
            for table in PROJECT_TABLES:
                await conn.execute(f"DELETE FROM {table} WHERE project_id = $1", project_id)
