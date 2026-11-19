"""Bộ xương của rig mà gói Remotion vẽ được.

Nguồn duy nhất là `remotion_project/src/conceptflow-mini/skeletons.json`: TypeScript
import nó để vẽ, Python đọc để kiểm tra, nên hai bên không lệch nhau.
"""

from __future__ import annotations

import json
from pathlib import Path

SKELETON_FILE = Path("src") / "conceptflow-mini" / "skeletons.json"


def load_skeletons(project_dir: str | Path) -> dict[str, dict[str, str | None]]:
    """Tên bộ xương -> {tên xương: xương cha (None ở gốc)}."""
    return json.loads((Path(project_dir) / SKELETON_FILE).read_text(encoding="utf-8"))
