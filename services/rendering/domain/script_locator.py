"""Tìm composition id (Remotion) trong script.

Tên này là một khai báo tĩnh trong mã nguồn nên đọc bằng regex là đúng và đủ,
và nó phải được biết **trước** lượt dry vì Remotion cần biết dựng composition
nào. Không bao giờ thực thi script.
"""

from __future__ import annotations

import re
from dataclasses import dataclass

#: Một entry file (Root.tsx) đăng ký composition qua
#: `<Composition id="..." component={...} .../>` bên trong `registerRoot()`.
REMOTION_COMPOSITION_RE = re.compile(r"<Composition\b[^>]*\bid\s*=\s*[\"']([\w-]+)[\"']")
REGISTER_ROOT_RE = re.compile(r"\bregisterRoot\s*\(")

NOT_FOUND_MESSAGE = (
    "không tìm thấy Remotion composition nào (cần "
    '`<Composition id="..." ... />` bên trong `registerRoot()`)'
)


@dataclass(frozen=True)
class LocatedScene:
    """The composition the script registers."""

    scene_class_name: str


class SceneNotFoundError(ValueError):
    def __init__(self) -> None:
        super().__init__(NOT_FOUND_MESSAGE)


def locate_scene(script: str) -> LocatedScene:
    """The first composition registered inside `registerRoot()`.

    Raises:
        SceneNotFoundError: the script registers no composition.
    """
    if REGISTER_ROOT_RE.search(script):
        match = REMOTION_COMPOSITION_RE.search(script)
        if match:
            return LocatedScene(match.group(1))
    raise SceneNotFoundError()
