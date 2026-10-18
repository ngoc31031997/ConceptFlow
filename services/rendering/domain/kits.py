"""Bộ kit (ngôn ngữ hình) mà gói Remotion vẽ được, và bộ xương của rig.

Nguồn duy nhất là hai file JSON nằm cạnh mã TypeScript
(`remotion_project/src/conceptflow-mini/kits.json`, `skeletons.json`): TypeScript
import chúng để vẽ, Python đọc chúng để kiểm tra, nên hai bên không lệch nhau.
"""

from __future__ import annotations

import json
from pathlib import Path

KIT_DIR = Path("src") / "conceptflow-mini"


class UnknownKitError(ValueError):
    """`kit_key` không có trong bộ kit."""


def _read(project_dir: str | Path, name: str) -> dict:
    return json.loads((Path(project_dir) / KIT_DIR / name).read_text(encoding="utf-8"))


def load_kits(project_dir: str | Path) -> dict[str, int]:
    """Mã kit -> phiên bản của kit đó."""
    return {key: int(meta["version"]) for key, meta in _read(project_dir, "kits.json").items()}


def load_skeletons(project_dir: str | Path) -> dict[str, dict[str, str | None]]:
    """Tên bộ xương -> {tên xương: xương cha (None ở gốc)}."""
    return _read(project_dir, "skeletons.json")


def require_kit(kits: dict[str, int], kit_key: str) -> int:
    """Phiên bản của `kit_key`; thiếu hoặc sai thì ném lỗi nói rõ."""
    if not kit_key:
        raise UnknownKitError("thiếu kit_key: project chưa ghim ngôn ngữ hình")
    if kit_key not in kits:
        raise UnknownKitError(f"kit_key {kit_key!r} không có trong bộ kit ({', '.join(sorted(kits))})")
    return kits[kit_key]
