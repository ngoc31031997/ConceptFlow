"""PostgreSQL cho thư viện hình minh hoạ: bảng `illustration_folders` và `illustrations`.

Ảnh xem trước không nằm trong database: hàng giữ đường dẫn PNG/GIF và phiên bản của ảnh
(`preview_version`); ảnh chỉ tính là có khi phiên bản đó bằng phiên bản hiện tại của hình.
"""

from __future__ import annotations

import json
from datetime import datetime

import asyncpg

from domain.illustration import (
    EXEMPLAR_FOLDER_ID,
    AlreadyExemplarError,
    CodeFinding,
    ExemplarLimitError,
    FolderNotFoundError,
    FolderTakenError,
    Illustration,
    IllustrationFilter,
    IllustrationFolder,
    IllustrationKind,
    IllustrationNameTakenError,
    IllustrationNotFoundError,
    IllustrationStatus,
    builtin_illustrations,
    detail_or_medium,
    format_time,
    kind_or_figure,
    system_folders,
)

COLUMNS = """id, name, title, folder_id, tags, description, usage, code, builtin, exemplar,
    COALESCE(source_id, ''), COALESCE(home_folder_id, ''), warnings, status, version,
    (preview_png_path <> '' AND preview_version = version), created_at, updated_at, kind, ref_id,
    detail_level"""


def _to_illustration(row: asyncpg.Record) -> Illustration:
    (id_, name, title, folder_id, tags, description, usage, code, builtin, exemplar, source_id,
     home_folder_id, warnings, status, version, has_preview, created, updated, kind, ref_id,
     detail) = tuple(row)
    findings = json.loads(warnings) if isinstance(warnings, str) else (warnings or [])
    created_at: datetime = created
    updated_at: datetime = updated
    return Illustration(
        id=id_, name=name, title=title, folder_id=folder_id, tags=tuple(tags or ()), description=description,
        usage=usage, code=code, builtin=builtin, exemplar=exemplar, source_id=source_id,
        home_folder_id=home_folder_id,
        warnings=tuple(CodeFinding(message=w["message"], line=w.get("line")) for w in findings),
        status=IllustrationStatus(status), version=version, has_preview=has_preview,
        created_at=format_time(created_at), updated_at=format_time(updated_at),
        kind=kind_or_figure(IllustrationKind(kind)), ref_id=ref_id, detail_level=detail_or_medium(detail))


def _findings_json(i: Illustration) -> str:
    return json.dumps([w.to_json() for w in i.warnings], ensure_ascii=False)


def _map_error(exc: asyncpg.PostgresError) -> Exception:
    """Đổi lỗi ràng buộc của database thành lỗi nghiệp vụ; lỗi khác giữ nguyên."""
    if isinstance(exc, asyncpg.UniqueViolationError):
        if "illustrations_name" in (exc.constraint_name or ""):
            return IllustrationNameTakenError()
        return FolderTakenError()
    if isinstance(exc, asyncpg.ForeignKeyViolationError):
        return FolderNotFoundError()
    return exc


class IllustrationRepository:
    def __init__(self, pool: asyncpg.Pool) -> None:
        self._pool = pool

    async def seed(self) -> None:
        """Nạp thư mục hệ thống và bộ hình có sẵn mỗi lần khởi động: image mới build là cả lần triển khai.
        Hình của Creator đã dùng tên của một hình có sẵn thì giữ tên đó, hình có sẵn bị bỏ qua."""
        async with self._pool.acquire() as conn:
            for f in system_folders():
                await conn.execute(
                    """
                    INSERT INTO illustration_folders (id, name, description, position, is_system)
                    VALUES ($1, $2, $3, $4, true)
                    ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description,
                        position = EXCLUDED.position, is_system = true
                    """, f.id, f.name, f.description, f.position)
            for b in builtin_illustrations():
                try:
                    await conn.execute(
                        """
                        INSERT INTO illustrations (id, name, title, folder_id, tags, description, usage, code,
                            builtin, exemplar, status, version, kind, ref_id, detail_level)
                        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, false, 'approved', 1, $9, '', $10)
                        ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, folder_id = EXCLUDED.folder_id,
                            kind = EXCLUDED.kind, detail_level = EXCLUDED.detail_level, tags = EXCLUDED.tags,
                            description = EXCLUDED.description, usage = EXCLUDED.usage, code = EXCLUDED.code,
                            version = illustrations.version
                                + (illustrations.code IS DISTINCT FROM EXCLUDED.code)::int,
                            updated_at = now()
                        """, b.id, b.name, b.title, b.folder_id, list(b.tags), b.description, b.usage, b.code,
                        b.kind.value, b.detail_level.value)
                except asyncpg.UniqueViolationError as exc:
                    if isinstance(_map_error(exc), IllustrationNameTakenError):
                        continue
                    raise

    # --- thư mục ---

    async def list_folders(self) -> list[IllustrationFolder]:
        rows = await self._pool.fetch(
            "SELECT id, name, description, position, is_system FROM illustration_folders "
            "ORDER BY position, name")
        return [IllustrationFolder(*tuple(r)) for r in rows]

    async def create_folder(self, folder: IllustrationFolder) -> IllustrationFolder:
        try:
            position = await self._pool.fetchval(
                """
                INSERT INTO illustration_folders (id, name, description, position, is_system)
                VALUES ($1, $2, $3, COALESCE((SELECT max(position) + 1 FROM illustration_folders), 1), false)
                RETURNING position
                """, folder.id, folder.name, folder.description)
        except asyncpg.PostgresError as exc:
            raise _map_error(exc) from exc
        return IllustrationFolder(
            id=folder.id, name=folder.name, description=folder.description, position=position,
            is_system=False)

    async def delete_folder(self, folder_id: str) -> None:
        await self._pool.execute(
            "DELETE FROM illustration_folders WHERE id = $1 AND NOT is_system", folder_id)

    # --- hình ---

    async def list_illustrations(self, flt: IllustrationFilter) -> list[Illustration]:
        q = "%" + flt.query.strip().lower() + "%"
        rows = await self._pool.fetch(
            f"""
            SELECT {COLUMNS} FROM illustrations
            WHERE ($1 = '' OR folder_id = $1)
              AND ($2 = '' OR status = $2)
              AND ($3 = '%%' OR lower(name) LIKE $3 OR lower(title) LIKE $3 OR lower(description) LIKE $3
                   OR lower(array_to_string(tags, ' ')) LIKE $3)
            ORDER BY builtin DESC, folder_id, title
            """, flt.folder_id, flt.status, q)
        return [_to_illustration(r) for r in rows]

    async def get_illustration(self, illustration_id: str) -> Illustration:
        row = await self._pool.fetchrow(f"SELECT {COLUMNS} FROM illustrations WHERE id = $1", illustration_id)
        if row is None:
            raise IllustrationNotFoundError()
        return _to_illustration(row)

    async def create_illustration(self, i: Illustration) -> Illustration:
        try:
            row = await self._pool.fetchrow(
                f"""
                INSERT INTO illustrations (name, title, folder_id, tags, description, usage, code, builtin,
                    status, version, warnings, kind, ref_id, detail_level)
                VALUES ($1, $2, $3, $4, $5, $6, $7, false, $8, $9, $10::jsonb, $11, $12, $13)
                RETURNING {COLUMNS}
                """, i.name, i.title, i.folder_id, list(i.tags), i.description, i.usage, i.code,
                i.status.value, i.version, _findings_json(i), kind_or_figure(i.kind).value, i.ref_id,
                detail_or_medium(i.detail_level).value)
        except asyncpg.PostgresError as exc:
            raise _map_error(exc) from exc
        return _to_illustration(row)

    async def update_illustration(self, i: Illustration) -> Illustration:
        try:
            row = await self._pool.fetchrow(
                f"""
                UPDATE illustrations SET name = $2, title = $3, folder_id = $4, tags = $5, description = $6,
                    usage = $7, code = $8, status = $9, version = $10, warnings = $11::jsonb, kind = $12,
                    ref_id = $13, detail_level = $14, updated_at = now()
                WHERE id = $1 AND NOT builtin
                RETURNING {COLUMNS}
                """, i.id, i.name, i.title, i.folder_id, list(i.tags), i.description, i.usage, i.code,
                i.status.value, i.version, _findings_json(i), kind_or_figure(i.kind).value, i.ref_id,
                detail_or_medium(i.detail_level).value)
        except asyncpg.PostgresError as exc:
            raise _map_error(exc) from exc
        if row is None:
            raise IllustrationNotFoundError()
        return _to_illustration(row)

    async def delete_illustration(self, illustration_id: str) -> None:
        await self._pool.execute("DELETE FROM illustrations WHERE id = $1 AND NOT builtin", illustration_id)

    async def create_exemplar(self, i: Illustration, limit: int) -> Illustration:
        """Khoá hàng thư mục Hình mẫu khi đếm, để hai "Đặt làm mẫu" cùng lúc không cùng vượt giới hạn."""
        async with self._pool.acquire() as conn, conn.transaction():
            await conn.execute(
                "SELECT 1 FROM illustration_folders WHERE id = $1 FOR UPDATE", EXEMPLAR_FOLDER_ID)
            count, copied = await conn.fetchrow(
                """
                SELECT count(*), COALESCE(bool_or(source_id = NULLIF($1, '')), false)
                FROM illustrations WHERE exemplar
                """, i.source_id)
            if copied:
                raise AlreadyExemplarError()
            if count >= limit:
                raise ExemplarLimitError()
            try:
                row = await conn.fetchrow(
                    f"""
                    INSERT INTO illustrations (name, title, folder_id, tags, description, usage, code,
                        builtin, exemplar, source_id, home_folder_id, status, version, warnings, kind, ref_id,
                        detail_level)
                    VALUES ($1, $2, $3, $4, $5, $6, $7, false, true, NULLIF($8, ''), NULLIF($9, ''), $10, $11,
                        $12::jsonb, $13, $14, $15)
                    RETURNING {COLUMNS}
                    """, i.name, i.title, EXEMPLAR_FOLDER_ID, list(i.tags), i.description, i.usage, i.code,
                    i.source_id, i.home_folder_id, i.status.value, i.version, _findings_json(i),
                    kind_or_figure(i.kind).value, i.ref_id, detail_or_medium(i.detail_level).value)
            except asyncpg.PostgresError as exc:
                raise _map_error(exc) from exc
        return _to_illustration(row)

    async def release_exemplar(self, illustration_id: str, folder_id: str) -> Illustration:
        try:
            row = await self._pool.fetchrow(
                f"""
                UPDATE illustrations SET exemplar = false, home_folder_id = NULL, folder_id = $2,
                    updated_at = now()
                WHERE id = $1 AND exemplar
                RETURNING {COLUMNS}
                """, illustration_id, folder_id)
        except asyncpg.PostgresError as exc:
            raise _map_error(exc) from exc
        if row is None:
            raise IllustrationNotFoundError()
        return _to_illustration(row)

    # --- ảnh xem trước ---

    async def set_preview(self, illustration_id: str, version: int, png_path: str, gif_path: str) -> None:
        await self._pool.execute(
            """
            UPDATE illustrations SET preview_png_path = $3, preview_gif_path = $4, preview_version = $2
            WHERE id = $1
            """, illustration_id, version, png_path, gif_path)

    async def get_preview(self, illustration_id: str) -> tuple[str, str]:
        row = await self._pool.fetchrow(
            "SELECT preview_png_path, preview_gif_path, preview_version = version "
            "FROM illustrations WHERE id = $1",
            illustration_id)
        if row is None:
            raise IllustrationNotFoundError()
        png_path, gif_path, current = tuple(row)
        return (png_path, gif_path) if current else ("", "")

