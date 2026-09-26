"""locate_scene (chuyển từ script-processing, CR-040 FR110).

Các test cũ về thứ tự `# NARRATION:` và gom `# CHAPTER:` đã bỏ: hai thứ đó
không đọc từ text nữa, chúng do lượt dry của Rendering thu theo thứ tự chạy
thật. Xem `services/rendering/tests/conceptflow/test_narration.py`.
"""

from __future__ import annotations

import pytest

from domain.script_locator import SceneNotFoundError, locate_scene

SCRIPT = (
    "from conceptflow import *\n\n"
    "class ForLoopScene(ConceptFlowScene):\n"
    "    def construct(self):\n"
    '        self.narrate("một")\n'
)


def test_tim_ten_class_scene():
    assert locate_scene(SCRIPT).scene_class_name == "ForLoopScene"


def test_lay_class_dau_tien_khi_co_nhieu_class():
    """Rendering chỉ chạy một class; class đầu tiên trong file là class đó."""
    script = SCRIPT + "\n\nclass PhuScene(ConceptFlowScene):\n    pass\n"
    assert locate_scene(script).scene_class_name == "ForLoopScene"


@pytest.mark.parametrize("base", ["Scene", "ConceptFlowScene", "MovingCameraScene", "ThreeDScene"])
def test_nhan_moi_base_class_co_chu_scene(base):
    script = f"class DemoScene({base}):\n    pass\n"
    assert locate_scene(script).scene_class_name == "DemoScene"


def test_bao_loi_khi_khong_co_class_scene():
    with pytest.raises(SceneNotFoundError):
        locate_scene("x = 1\n")


def test_khong_con_bat_buoc_co_narration_trong_text():
    """Một script không có chữ 'narrate' nào trong nguồn vẫn hợp lệ ở bước này:
    lời thoại có thể nằm trong hàm helper của thư viện, và chỉ lượt dry mới
    biết. Bắt lỗi ở đây sẽ chặn nhầm đúng loại script mà CR-018 mở ra."""
    script = "class DemoScene(ConceptFlowScene):\n    def construct(self):\n        pass\n"
    assert locate_scene(script).scene_class_name == "DemoScene"


REMOTION_ENTRY = (
    "import { registerRoot, Composition } from 'remotion';\n"
    "import { MyVideo } from './MyVideo';\n\n"
    "export const RemotionRoot: React.FC = () => {\n"
    "  return (\n"
    "    <Composition\n"
    '      id="MyComp"\n'
    "      component={MyVideo}\n"
    "      durationInFrames={150}\n"
    "      fps={30}\n"
    "      width={1920}\n"
    "      height={1080}\n"
    "    />\n"
    "  );\n"
    "};\n\n"
    "registerRoot(RemotionRoot);\n"
)


def test_nhan_dien_remotion_composition_trong_registerroot():
    parsed = locate_scene(REMOTION_ENTRY)
    assert parsed.scene_class_name == "MyComp"
    assert parsed.engine == "remotion"


def test_manim_van_uu_tien_khi_ca_hai_dang_deu_co_the_khop():
    """Class Scene được thử trước — grammar Manim đã tồn tại từ trước và
    không được lùi bước trước grammar Remotion mới thêm."""
    parsed = locate_scene(SCRIPT)
    assert parsed.engine == "manim"


def test_composition_khong_trong_registerroot_thi_khong_tinh():
    """Chỉ có `<Composition id="...">` mà không có `registerRoot(...)` thì
    không phải một entry file Remotion hợp lệ — báo lỗi như trước đây."""
    script = '<Composition id="MyComp" component={X} />\n'
    with pytest.raises(SceneNotFoundError):
        locate_scene(script)


def test_registerroot_khong_co_composition_thi_khong_tinh():
    script = "registerRoot(RemotionRoot);\n"
    with pytest.raises(SceneNotFoundError):
        locate_scene(script)
