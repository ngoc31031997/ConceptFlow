"""Ảnh xem trước của thư viện hình trên volume dùng chung: `{root}/{id}/v{version}.png|.gif`.

Ghi file tạm rồi đổi tên, nên người đọc không bao giờ thấy file dở. Mỗi hình giữ ảnh của phiên bản
hiện tại; ảnh phiên bản cũ bị xoá sau khi ảnh mới đã nằm đúng chỗ.
"""

from __future__ import annotations

import os
import shutil
from pathlib import Path

ILLUSTRATIONS_ROOT = "/shared/illustrations"


class IllustrationFiles:
    def __init__(self, root: str | Path = ILLUSTRATIONS_ROOT) -> None:
        self._root = Path(root)

    def save(self, illustration_id: str, version: int, png: bytes, gif: bytes) -> tuple[str, str]:
        folder = self._folder(illustration_id)
        folder.mkdir(parents=True, exist_ok=True)
        png_path = self._write(folder / f"v{version}.png", png)
        gif_path = self._write(folder / f"v{version}.gif", gif) if gif else ""
        keep = {Path(p).name for p in (png_path, gif_path) if p}
        for old in folder.iterdir():
            if old.name not in keep:
                old.unlink(missing_ok=True)
        return png_path, gif_path

    def read(self, path: str) -> bytes:
        return self._inside(path).read_bytes()

    def remove(self, illustration_id: str) -> None:
        shutil.rmtree(self._folder(illustration_id), ignore_errors=True)

    def _folder(self, illustration_id: str) -> Path:
        # Mã hình do database sinh (uuid) hoặc "builtin-<Tên>"; không cho dấu phân cách đường dẫn đi qua.
        bad = not illustration_id or "/" in illustration_id or "\\" in illustration_id
        if bad or illustration_id in (".", ".."):
            raise ValueError(f"mã hình không hợp lệ: {illustration_id!r}")
        return self._root / illustration_id

    def _inside(self, path: str) -> Path:
        resolved = Path(path).resolve()
        if not resolved.is_relative_to(self._root.resolve()):
            raise ValueError(f"đường dẫn ảnh nằm ngoài {self._root}: {path}")
        return resolved

    @staticmethod
    def _write(target: Path, body: bytes) -> str:
        tmp = target.with_name(target.name + ".tmp")
        tmp.write_bytes(body)
        os.replace(tmp, target)
        return str(target)
