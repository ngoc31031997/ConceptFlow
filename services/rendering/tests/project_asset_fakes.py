"""Bản giả trong bộ nhớ của kho và ảnh tài sản dự án, dùng cho test use case và route."""

from __future__ import annotations

from domain.project_asset import AssetNotFoundError, ProjectAsset, format_asset_id, name_key


class FakeAssetRepo:
    def __init__(self) -> None:
        self.rows: dict[tuple[str, str], ProjectAsset] = {}
        self.counters: dict[tuple[str, str], int] = {}

    async def list_assets(self, project_id):
        return [a for (p, _), a in self.rows.items() if p == project_id]

    async def get_asset(self, project_id, asset_id):
        if (project_id, asset_id) not in self.rows:
            raise AssetNotFoundError()
        return self.rows[(project_id, asset_id)]

    async def create_asset(self, asset):
        key = (asset.project_id, name_key(asset.name))
        self.counters[key] = self.counters.get(key, 0) + 1
        row = asset.with_(
            asset_id=format_asset_id(asset.name, self.counters[key]), created_at="2026-11-19T00:00:00Z",
            updated_at="2026-11-19T00:00:00Z")
        self.rows[(row.project_id, row.asset_id)] = row
        return row

    async def update_asset(self, asset):
        if (asset.project_id, asset.asset_id) not in self.rows:
            raise AssetNotFoundError()
        self.rows[(asset.project_id, asset.asset_id)] = asset
        return asset

    async def delete_asset(self, project_id, asset_id):
        self.rows.pop((project_id, asset_id), None)


class FakeAssetFiles:
    def __init__(self) -> None:
        self.files: dict[str, bytes] = {}

    def save(self, project_id, asset_id, version, png):
        prefix = f"/shared/{project_id}/storyboard/assets/{asset_id}/"
        for path in [p for p in self.files if p.startswith(prefix)]:
            del self.files[path]
        path = f"{prefix}v{version}.png"
        self.files[path] = png
        return path

    def read(self, path):
        return self.files[path]

    def remove(self, project_id, asset_id):
        prefix = f"/shared/{project_id}/storyboard/assets/{asset_id}/"
        for path in [p for p in self.files if p.startswith(prefix)]:
            del self.files[path]
