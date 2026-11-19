"""Ảnh tĩnh dùng chung: một khung của composition mẫu, dựng một lần và cache.

Tên file là SHA-256 của mọi thứ làm ảnh đổi: composition, nội dung kit, props
và khung hình. Cùng nội dung thì trả lại file cũ (`cache_hit`), kit đổi một
con số thì ra ảnh mới. Dọn theo danh sách giữ lại: file không có trong
`keep` và đủ cũ thì xoá.
"""

from __future__ import annotations

import hashlib
import json
import os
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Protocol

from domain.kit import validate_kit

HASH_LENGTH = 64
SECONDS_PER_HOUR = 3600


class StillRenderer(Protocol):
    """Dựng một khung của composition mẫu ra file."""

    def still(
        self, kit: dict, composition: str, output: str, frame: int = ..., props: dict | None = ...,
    ) -> None: ...


@dataclass(frozen=True)
class StillResult:
    """File ảnh (`file` là tên trong thư mục ảnh tĩnh) và có lấy từ cache không."""

    file: str
    cache_hit: bool


class StillsUseCase:
    def __init__(self, renderer: StillRenderer, directory: str | Path) -> None:
        self._renderer = renderer
        self._dir = Path(directory)

    def still_hash(self, composition: str, kit: dict, props: dict, frame: int) -> str:
        """Hash của một ảnh: đổi bất kỳ thành phần nào cũng ra hash khác."""
        payload = json.dumps(
            {"composition": composition, "kit": kit, "props": props, "frame": frame},
            sort_keys=True, separators=(",", ":"), ensure_ascii=False,
        )
        return hashlib.sha256(payload.encode("utf-8")).hexdigest()

    def render(self, composition: str, kit: dict, props: dict | None = None, frame: int = 10) -> StillResult:
        validate_kit(kit)
        props = props or {}
        name = f"{self.still_hash(composition, kit, props, frame)}.png"
        target = self._dir / name
        if target.exists():
            return StillResult(name, True)
        self._dir.mkdir(parents=True, exist_ok=True)
        # Dựng ra file tạm rồi đổi tên: một yêu cầu khác không bao giờ thấy ảnh dang dở.
        partial = self._dir / f"{name}.{os.getpid()}.part"
        try:
            self._renderer.still(kit, composition, str(partial), frame, props)
            os.replace(partial, target)
        finally:
            partial.unlink(missing_ok=True)
        return StillResult(name, False)

    def prune(self, keep: list[str], min_age_hours: float) -> list[str]:
        """Xoá ảnh không có trong `keep` và cũ hơn `min_age_hours`; trả tên các file đã xoá."""
        if min_age_hours < 0:
            raise ValueError("min_age_hours không được âm")
        if not self._dir.exists():
            return []
        kept = {k for k in keep}
        cutoff = time.time() - min_age_hours * SECONDS_PER_HOUR
        removed: list[str] = []
        for path in sorted(self._dir.glob("*.png")):
            if path.name in kept or len(path.stem) != HASH_LENGTH:
                continue
            if path.stat().st_mtime <= cutoff:
                path.unlink()
                removed.append(path.name)
        return removed
