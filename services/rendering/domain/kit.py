"""Bộ thông số vẽ (kit): mọi lượt vẽ đều nhận một object kit, không có bảng kit dựng sẵn.

Hình dạng khớp `KitTokens` trong `remotion_project/src/conceptflow-mini/kits.ts` và
kit lưu ở visual-service. Mọi số nằm trong 0..1, trừ `dof` (bán kính làm mờ, px >= 0) và
`previewLook.grade` (-1..1); màu là `#RRGGBB`; `previewLook.focus` là một trong năm lớp chiều sâu.
"""

from __future__ import annotations

import math
import re
from typing import Any

FOCUS_LAYERS = ("sky", "far", "mid", "near", "figures")

_UNIT = "unit"
_SIGNED = "signed"
_PX = "px"
_COLOR = "color"
_FOCUS = "focus"

# Cây hình dạng của kit: lá là kiểu giá trị, nhánh là object con.
_SHAPE: dict[str, Any] = {
    "outline": _UNIT, "radius": _UNIT, "rimWidth": _UNIT, "rimStrength": _UNIT,
    "shadowStrength": _UNIT, "aoStrength": _UNIT,
    "specular": {"glossy": _UNIT, "metal": _UNIT, "glass": _UNIT},
    "texture": {
        "fur": _UNIT, "fabric": _UNIT, "wood": _UNIT, "stone": _UNIT, "organic": _UNIT, "leaf": _UNIT,
    },
    "sss": _UNIT, "glowBlur": _UNIT, "bloom": _UNIT,
    "haze": {"far": _UNIT, "mid": _UNIT},
    "dof": {"sky": _PX, "far": _PX, "mid": _PX, "near": _PX},
    "grain": _UNIT, "vignette": _UNIT, "ink": _COLOR,
    "previewLook": {
        "light": {"x": _UNIT, "y": _UNIT, "color": _COLOR, "intensity": _UNIT},
        "ambient": _COLOR,
        "atmosphere": {"color": _COLOR, "density": _UNIT},
        "focus": _FOCUS,
        "grade": {"warmth": _SIGNED, "contrast": _SIGNED, "saturation": _SIGNED},
    },
}

_HEX_COLOR = re.compile(r"^#[0-9A-Fa-f]{6}$")


class InvalidKitError(ValueError):
    """Kit thiếu hoặc sai; thông báo nêu trường sai."""


def _is_number(value: object) -> bool:
    return isinstance(value, int | float) and not isinstance(value, bool) and math.isfinite(value)


def _check_leaf(path: str, kind: str, value: object) -> None:
    if kind == _COLOR:
        if not isinstance(value, str) or not _HEX_COLOR.match(value):
            raise InvalidKitError(f"kit.{path} phải là màu #RRGGBB, nhận {value!r}")
    elif kind == _FOCUS:
        if value not in FOCUS_LAYERS:
            raise InvalidKitError(f"kit.{path} phải là một trong {', '.join(FOCUS_LAYERS)}, nhận {value!r}")
    elif not _is_number(value):
        raise InvalidKitError(f"kit.{path} phải là số, nhận {value!r}")
    elif kind == _UNIT and not 0 <= value <= 1:
        raise InvalidKitError(f"kit.{path} phải trong khoảng 0..1, nhận {value!r}")
    elif kind == _SIGNED and not -1 <= value <= 1:
        raise InvalidKitError(f"kit.{path} phải trong khoảng -1..1, nhận {value!r}")
    elif kind == _PX and value < 0:
        raise InvalidKitError(f"kit.{path} (px) phải >= 0, nhận {value!r}")


def _check(path: str, shape: dict[str, Any], value: object) -> None:
    if not isinstance(value, dict):
        raise InvalidKitError(f"kit{'.' + path if path else ''} phải là object, nhận {type(value).__name__}")
    for key, sub in shape.items():
        child = f"{path}.{key}" if path else key
        if key not in value:
            raise InvalidKitError(f"kit thiếu trường {child}")
        if isinstance(sub, dict):
            _check(child, sub, value[key])
        else:
            _check_leaf(child, sub, value[key])
    extra = sorted(set(value) - set(shape))
    if extra:
        raise InvalidKitError(f"kit{'.' + path if path else ''} có trường lạ: {', '.join(extra)}")


def validate_kit(kit: object) -> dict:
    """Trả lại `kit` nếu hợp lệ; thiếu, sai kiểu hay ngoài khoảng thì ném InvalidKitError nêu trường."""
    if kit is None or kit == {}:
        raise InvalidKitError("thiếu kit: dự án chưa có bộ thông số vẽ")
    _check("", _SHAPE, kit)
    assert isinstance(kit, dict)
    return kit
