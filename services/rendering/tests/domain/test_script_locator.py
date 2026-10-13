"""locate_scene: tìm composition id của một entry file Remotion."""

from __future__ import annotations

import pytest

from domain.script_locator import SceneNotFoundError, locate_scene

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
    assert locate_scene(REMOTION_ENTRY).scene_class_name == "MyComp"


def test_bao_loi_khi_khong_co_composition():
    with pytest.raises(SceneNotFoundError):
        locate_scene("x = 1\n")


def test_class_scene_kieu_python_khong_con_duoc_nhan():
    """Một script Python với class Scene không phải entry file Remotion."""
    script = "class DemoScene(Scene):\n    def construct(self):\n        pass\n"
    with pytest.raises(SceneNotFoundError):
        locate_scene(script)


def test_composition_khong_trong_registerroot_thi_khong_tinh():
    """Chỉ có `<Composition id="...">` mà không có `registerRoot(...)` thì
    không phải một entry file Remotion hợp lệ — báo lỗi."""
    script = '<Composition id="MyComp" component={X} />\n'
    with pytest.raises(SceneNotFoundError):
        locate_scene(script)


def test_registerroot_khong_co_composition_thi_khong_tinh():
    script = "registerRoot(RemotionRoot);\n"
    with pytest.raises(SceneNotFoundError):
        locate_scene(script)
