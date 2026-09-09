"""ManimScriptParser — sau CR-018 chỉ còn tìm tên class Scene.

Các test cũ về thứ tự `# NARRATION:` và gom `# CHAPTER:` đã bỏ: hai thứ đó
không đọc từ text nữa, chúng do lượt dry của Rendering thu theo thứ tự chạy
thật. Xem `services/rendering/tests/conceptflow/test_narration.py`.
"""

from __future__ import annotations

import pytest

from adapters.parsing.manim_script_parser import ManimScriptParser
from domain.errors import ScriptSyntaxError

SCRIPT = (
    "from conceptflow import *\n\n"
    "class ForLoopScene(ConceptFlowScene):\n"
    "    def construct(self):\n"
    '        self.narrate("một")\n'
)


def test_tim_ten_class_scene():
    assert ManimScriptParser().parse(SCRIPT).scene_class_name == "ForLoopScene"


def test_lay_class_dau_tien_khi_co_nhieu_class():
    """Rendering chỉ chạy một class; class đầu tiên trong file là class đó."""
    script = SCRIPT + "\n\nclass PhuScene(ConceptFlowScene):\n    pass\n"
    assert ManimScriptParser().parse(script).scene_class_name == "ForLoopScene"


@pytest.mark.parametrize("base", ["Scene", "ConceptFlowScene", "MovingCameraScene", "ThreeDScene"])
def test_nhan_moi_base_class_co_chu_scene(base):
    script = f"class DemoScene({base}):\n    pass\n"
    assert ManimScriptParser().parse(script).scene_class_name == "DemoScene"


def test_bao_loi_khi_khong_co_class_scene():
    with pytest.raises(ScriptSyntaxError):
        ManimScriptParser().parse("x = 1\n")


def test_khong_con_bat_buoc_co_narration_trong_text():
    """Một script không có chữ 'narrate' nào trong nguồn vẫn hợp lệ ở bước này:
    lời thoại có thể nằm trong hàm helper của thư viện, và chỉ lượt dry mới
    biết. Bắt lỗi ở đây sẽ chặn nhầm đúng loại script mà CR-018 mở ra."""
    script = "class DemoScene(ConceptFlowScene):\n    def construct(self):\n        pass\n"
    assert ManimScriptParser().parse(script).scene_class_name == "DemoScene"
