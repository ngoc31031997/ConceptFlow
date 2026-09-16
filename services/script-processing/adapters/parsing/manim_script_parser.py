"""ManimScriptParser — implements ScriptParserPort.

Sau CR-018, service này chỉ còn tìm **tên class Scene**. Lời thoại, beat và
chapter không đọc được từ text nữa: chúng do lượt dry của Rendering thu thập
theo thứ tự chạy thật (`self.narrate(...)` có thể nằm trong vòng lặp, trong
nhánh điều kiện, trong hàm helper — những chỗ mà quét comment không bao giờ
đếm đúng).

Tên class thì ngược lại: nó là một khai báo tĩnh trong mã nguồn, đọc bằng regex
là đúng và đủ. Và nó phải được biết **trước** lượt dry, vì `manim` cần biết chạy
class nào.

Service này vẫn không bao giờ thực thi script.
"""

from __future__ import annotations

import re

from domain.errors import ScriptSyntaxError
from domain.models import ParsedScript
from domain.ports import ScriptParserPort

#: Khớp cả `Scene`, `ConceptFlowScene` và các subclass khác của Manim
#: (`MovingCameraScene`...). Class đầu tiên tìm thấy là class được render.
SCENE_CLASS_RE = re.compile(r"^class\s+(\w+)\s*\([^)]*Scene[^)]*\)\s*:")

#: Remotion: một entry file (Root.tsx) đăng ký composition qua
#: `<Composition id="..." component={...} .../>` bên trong `registerRoot()`
#: (https://www.remotion.dev/docs/the-fundamentals). Chỉ regex-confirm hai
#: dấu hiệu này tồn tại — không parse sâu JSX/TSX (ngoài phạm vi CR này).
REMOTION_COMPOSITION_RE = re.compile(r"<Composition\b[^>]*\bid\s*=\s*[\"']([\w-]+)[\"']")
REGISTER_ROOT_RE = re.compile(r"\bregisterRoot\s*\(")


class ManimScriptParser(ScriptParserPort):
    """Detects and parses either a Manim Python script or a Remotion
    entry file (CR: Remotion rendering engine).

    Manim is tried first — its `class ... (...Scene...):` declaration is
    unambiguous and was the only grammar this service ever knew. A script
    that instead registers a Remotion `<Composition>` inside
    `registerRoot()` is the new, second grammar; anything matching
    neither is a syntax error exactly as before.
    """

    def parse(self, raw_script: str) -> ParsedScript:
        scene_class_name = self._find_scene_class(raw_script.splitlines())
        if scene_class_name is not None:
            return ParsedScript(scene_class_name=scene_class_name, engine="manim")

        composition_id = self._find_remotion_composition(raw_script)
        if composition_id is not None:
            return ParsedScript(scene_class_name=composition_id, engine="remotion")

        raise ScriptSyntaxError(
            None,
            "không tìm thấy class Scene nào (cần dạng "
            "`class TenScene(ConceptFlowScene):`) và cũng không tìm thấy "
            "Remotion composition nào (cần `<Composition id=\"...\" ... />` "
            "bên trong `registerRoot()`)",
        )

    @staticmethod
    def _find_scene_class(lines: list[str]) -> str | None:
        for line in lines:
            match = SCENE_CLASS_RE.match(line.strip())
            if match:
                return match.group(1)
        return None

    @staticmethod
    def _find_remotion_composition(raw_script: str) -> str | None:
        if not REGISTER_ROOT_RE.search(raw_script):
            return None
        match = REMOTION_COMPOSITION_RE.search(raw_script)
        if match is None:
            return None
        return match.group(1)
