"""Tìm tên class Scene (Manim) hoặc composition id (Remotion) trong script.

Chuyển từ script-processing (CR-040 FR110): tên này là một khai báo tĩnh trong
mã nguồn nên đọc bằng regex là đúng và đủ, và nó phải được biết **trước** lượt
dry vì `manim` cần biết chạy class nào. Không bao giờ thực thi script.
"""

from __future__ import annotations

import re
from dataclasses import dataclass

#: Khớp cả `Scene`, `ConceptFlowScene` và các subclass khác của Manim
#: (`MovingCameraScene`...). Class đầu tiên tìm thấy là class được render.
SCENE_CLASS_RE = re.compile(r"^class\s+(\w+)\s*\([^)]*Scene[^)]*\)\s*:")

#: Remotion: một entry file (Root.tsx) đăng ký composition qua
#: `<Composition id="..." component={...} .../>` bên trong `registerRoot()`.
REMOTION_COMPOSITION_RE = re.compile(r"<Composition\b[^>]*\bid\s*=\s*[\"']([\w-]+)[\"']")
REGISTER_ROOT_RE = re.compile(r"\bregisterRoot\s*\(")

NOT_FOUND_MESSAGE = (
    "không tìm thấy class Scene nào (cần dạng `class TenScene(ConceptFlowScene):`) "
    "và cũng không tìm thấy Remotion composition nào (cần "
    '`<Composition id="..." ... />` bên trong `registerRoot()`)'
)


@dataclass(frozen=True)
class LocatedScene:
    scene_class_name: str
    engine: str  # "manim" | "remotion"


class SceneNotFoundError(ValueError):
    def __init__(self) -> None:
        super().__init__(NOT_FOUND_MESSAGE)


def locate_scene(script: str) -> LocatedScene:
    """Manim trước (khai báo `class ...Scene` không nhập nhằng), rồi Remotion."""
    for line in script.splitlines():
        match = SCENE_CLASS_RE.match(line.strip())
        if match:
            return LocatedScene(match.group(1), "manim")
    if REGISTER_ROOT_RE.search(script):
        match = REMOTION_COMPOSITION_RE.search(script)
        if match:
            return LocatedScene(match.group(1), "remotion")
    raise SceneNotFoundError()
