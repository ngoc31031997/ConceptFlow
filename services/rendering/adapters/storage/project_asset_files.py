"""Ảnh xem trước tài sản của dự án: `{root}/{project_id}/storyboard/assets/{asset_id}/v{n}.png`.

Ghi file tạm rồi đổi tên nên người đọc không thấy file dở. Mỗi tài sản giữ ảnh của phiên bản hiện tại.
"""

from __future__ import annotations

import os
import shutil
from pathlib import Path

from domain.project_asset import validate_asset_id, validate_project_id

SHARED_ROOT = "/shared"


class ProjectAssetFiles:
    def __init__(self, root: str | Path = SHARED_ROOT) -> None:
        self._root = Path(root)

    def save(self, project_id: str, asset_id: str, version: int, png: bytes) -> str:
        """Ghi PNG của `version`, xoá ảnh các phiên bản cũ, trả đường dẫn."""
        folder = self._folder(project_id, asset_id)
        folder.mkdir(parents=True, exist_ok=True)
        target = folder / f"v{version}.png"
        tmp = target.with_name(target.name + ".tmp")
        tmp.write_bytes(png)
        os.replace(tmp, target)
        for old in folder.iterdir():
            if old.name != target.name:
                old.unlink(missing_ok=True)
        return str(target)

    def read(self, path: str) -> bytes:
        resolved = Path(path).resolve()
        if not resolved.is_relative_to(self._root.resolve()):
            raise ValueError(f"đường dẫn ảnh nằm ngoài {self._root}: {path}")
        return resolved.read_bytes()

    def remove(self, project_id: str, asset_id: str) -> None:
        shutil.rmtree(self._folder(project_id, asset_id), ignore_errors=True)

    def _folder(self, project_id: str, asset_id: str) -> Path:
        validate_project_id(project_id)
        validate_asset_id(asset_id)
        return self._root / project_id / "storyboard" / "assets" / asset_id
