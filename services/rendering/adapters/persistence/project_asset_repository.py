"""PostgreSQL cho tài sản dự án: bảng `project_assets` và bộ đếm mã `project_asset_counters`."""

from __future__ import annotations

import json

import asyncpg

from domain.illustration import CodeFinding, format_time
from domain.project_asset import (
    AssetNotFoundError,
    AssetStatus,
    ProjectAsset,
    format_asset_id,
    name_key,
)

COLUMNS = """project_id, asset_id, concept_id, name, kind, tier, portrayal, must_not_imply, palette_tokens,
    expressions, base_illustration_id, units, code, version, preview_path, status, to_library,
    library_folder_id, library_illustration_id, warnings, created_at, updated_at"""


def _to_asset(row: asyncpg.Record) -> ProjectAsset:
    (project_id, asset_id, concept_id, name, kind, tier, portrayal, must_not_imply, palette_tokens,
     expressions, base_id, units, code, version, preview_path, status, to_library, folder_id, library_id,
     warnings, created, updated) = tuple(row)
    findings = json.loads(warnings) if isinstance(warnings, str) else (warnings or [])
    return ProjectAsset(
        project_id=project_id, asset_id=asset_id, concept_id=concept_id, name=name, kind=kind, tier=tier,
        portrayal=portrayal, must_not_imply=tuple(must_not_imply or ()),
        palette_tokens=tuple(palette_tokens or ()), expressions=tuple(expressions or ()),
        base_illustration_id=base_id, units=tuple(units or ()), code=code, version=version,
        preview_path=preview_path,
        status=AssetStatus(status), to_library=to_library, library_folder_id=folder_id,
        library_illustration_id=library_id,
        warnings=tuple(CodeFinding(message=w["message"], line=w.get("line")) for w in findings),
        created_at=format_time(created), updated_at=format_time(updated))


def _warnings_json(asset: ProjectAsset) -> str:
    return json.dumps([w.to_json() for w in asset.warnings], ensure_ascii=False)


class ProjectAssetRepository:
    def __init__(self, pool: asyncpg.Pool) -> None:
        self._pool = pool

    async def list_assets(self, project_id: str) -> list[ProjectAsset]:
        rows = await self._pool.fetch(
            f"SELECT {COLUMNS} FROM project_assets WHERE project_id = $1 ORDER BY created_at, asset_id",
            project_id)
        return [_to_asset(r) for r in rows]

    async def get_asset(self, project_id: str, asset_id: str) -> ProjectAsset:
        row = await self._pool.fetchrow(
            f"SELECT {COLUMNS} FROM project_assets WHERE project_id = $1 AND asset_id = $2",
            project_id, asset_id)
        if row is None:
            raise AssetNotFoundError()
        return _to_asset(row)

    async def create_asset(self, asset: ProjectAsset) -> ProjectAsset:
        """Cấp mã từ bộ đếm của `(dự án, tên)` rồi chèn hàng, cùng một giao dịch; mã bị bỏ qua khi chèn
        hỏng vẫn không được cấp lại."""
        async with self._pool.acquire() as conn, conn.transaction():
            number = await conn.fetchval(
                """
                INSERT INTO project_asset_counters (project_id, name_key, last_number) VALUES ($1, $2, 1)
                ON CONFLICT (project_id, name_key)
                DO UPDATE SET last_number = project_asset_counters.last_number + 1
                RETURNING last_number
                """, asset.project_id, name_key(asset.name))
            row = await conn.fetchrow(
                f"""
                INSERT INTO project_assets (project_id, asset_id, concept_id, name, kind, tier, portrayal,
                    must_not_imply, palette_tokens, expressions, base_illustration_id, units, status,
                    to_library, library_folder_id)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'planned', $13, $14)
                RETURNING {COLUMNS}
                """, asset.project_id, format_asset_id(asset.name, number), asset.concept_id, asset.name,
                asset.kind, asset.tier, asset.portrayal, list(asset.must_not_imply),
                list(asset.palette_tokens), list(asset.expressions), asset.base_illustration_id,
                list(asset.units), asset.to_library, asset.library_folder_id)
        return _to_asset(row)

    async def update_asset(self, asset: ProjectAsset) -> ProjectAsset:
        row = await self._pool.fetchrow(
            f"""
            UPDATE project_assets SET concept_id = $3, name = $4, kind = $5, tier = $6, portrayal = $7,
                must_not_imply = $8, palette_tokens = $9, expressions = $10, base_illustration_id = $11,
                units = $12, code = $13, version = $14, preview_path = $15, status = $16, to_library = $17,
                library_folder_id = $18, library_illustration_id = $19, warnings = $20::jsonb,
                updated_at = now()
            WHERE project_id = $1 AND asset_id = $2
            RETURNING {COLUMNS}
            """, asset.project_id, asset.asset_id, asset.concept_id, asset.name, asset.kind, asset.tier,
            asset.portrayal, list(asset.must_not_imply), list(asset.palette_tokens), list(asset.expressions),
            asset.base_illustration_id, list(asset.units), asset.code, asset.version, asset.preview_path,
            asset.status.value, asset.to_library, asset.library_folder_id, asset.library_illustration_id,
            _warnings_json(asset))
        if row is None:
            raise AssetNotFoundError()
        return _to_asset(row)

    async def delete_asset(self, project_id: str, asset_id: str) -> None:
        """Xoá hàng; bộ đếm giữ nguyên nên mã không được cấp lại."""
        await self._pool.execute(
            "DELETE FROM project_assets WHERE project_id = $1 AND asset_id = $2", project_id, asset_id)
