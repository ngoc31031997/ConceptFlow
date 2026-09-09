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


class ManimScriptParser(ScriptParserPort):
    def parse(self, raw_script: str) -> ParsedScript:
        scene_class_name = self._find_scene_class(raw_script.splitlines())
        if scene_class_name is None:
            raise ScriptSyntaxError(
                None,
                "không tìm thấy class Scene nào "
                "(cần dạng `class TenScene(ConceptFlowScene):`)",
            )
        return ParsedScript(scene_class_name=scene_class_name)

    @staticmethod
    def _find_scene_class(lines: list[str]) -> str | None:
        for line in lines:
            match = SCENE_CLASS_RE.match(line.strip())
            if match:
                return match.group(1)
        return None
