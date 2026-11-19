"""Prompt của bộ vẽ AI: vai (role), biến template và các đoạn chữ điền vào biến.

Template nằm ở llm-service (owner `rendering`); rendering chỉ điền biến và gọi model. Hai đoạn
chữ dùng chung `helpers` (linh kiện của kit) nằm cạnh code trong `illustration_helpers/`.
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path

from domain.illustration import (
    TIER_BACKGROUND,
    TIER_HERO,
    IllustrationKind,
    tier_or_support,
)

PROMPT_OWNER = "rendering"
ROLE_FIGURE = "illustration_drawer"
ROLE_BACKDROP = "illustration_backdrop_drawer"
STEP_ILLUSTRATIONS = "illustrations"

#: Tên hiển thị của hai prompt hệ thống ở trang Prompt.
SYSTEM_PROMPT_NAMES = {
    ROLE_FIGURE: "Bộ vẽ hình minh hoạ",
    ROLE_BACKDROP: "Bộ vẽ nền minh hoạ",
}

#: Biến mà code điền từ phong cách tham chiếu (`drawing_*` và `layout_*` là trường cùng tên của phong cách).
STYLE_VARIABLES = (
    "style_name", "style_summary",
    "drawing_medium", "drawing_shape", "drawing_outline", "drawing_fill", "drawing_shading",
    "drawing_light", "drawing_depth", "drawing_perspective", "drawing_texture",
    "layout_structure", "layout_layering", "layout_persistent_elements",
)
#: Mọi biến mà một template của bộ vẽ được dùng.
FIGURE_VARIABLES = (*STYLE_VARIABLES, "tier", "helpers", "references")
BACKDROP_VARIABLES = (*STYLE_VARIABLES, "helpers", "references")

_PLACEHOLDER_RE = re.compile(r"\{\{([a-z][a-z0-9_]*)\}\}")
_HELPERS_DIR = Path(__file__).parent / "illustration_helpers"

_TIER_TEXT = {
    TIER_HERO: "HẠNG CHÍNH (hero): hình này là vật hoặc nhân vật mà video nói về — vẽ kỹ, 80–400 hình con, "
               "chia bộ phận (<Part>, <Bone>).",
    TIER_BACKGROUND: "HẠNG PHÔNG (background): hình này chỉ là vật xa hoặc trang trí — 3–15 hình con, một "
                     "khối rõ dáng, một vật liệu, không mặt.",
}
_TIER_SUPPORT_TEXT = ("HẠNG PHỤ (support): hình này có vai nhỏ — 20–80 hình con, đủ để nhận ra ngay, vài bộ "
                      "phận chuyển động.")


def role_for(kind: IllustrationKind) -> str:
    return ROLE_BACKDROP if kind == IllustrationKind.BACKDROP else ROLE_FIGURE


def variables_for(kind: IllustrationKind) -> tuple[str, ...]:
    return BACKDROP_VARIABLES if kind == IllustrationKind.BACKDROP else FIGURE_VARIABLES


def tier_instruction(tier: object) -> str:
    """Cho bộ vẽ biết hình đóng vai gì trong video."""
    return _TIER_TEXT.get(tier_or_support(tier), _TIER_SUPPORT_TEXT)


def helpers_text(kind: IllustrationKind) -> str:
    """Mô tả linh kiện của kit mà một hình (hay nền) mới được dựng từ đó."""
    name = "backdrop_helpers_vi.txt" if kind == IllustrationKind.BACKDROP else "illustration_helpers_vi.txt"
    return (_HELPERS_DIR / name).read_text(encoding="utf-8").strip()


def style_values(style: dict) -> dict[str, str]:
    """Giá trị các biến phong cách, lấy nguyên chữ của phong cách tham chiếu (trường trống thì rỗng)."""
    out = {"style_name": str(style.get("name") or ""), "style_summary": str(style.get("summary") or "")}
    for group in ("drawing", "layout"):
        fields = style.get(group) or {}
        keys = (
            ("medium", "shape", "outline", "fill", "shading", "light", "depth", "perspective", "texture")
            if group == "drawing" else ("structure", "layering", "persistent_elements"))
        for key in keys:
            out[f"{group}_{key}"] = str(fields.get(key) or "").strip()
    return out


def placeholders(template: str) -> set[str]:
    return set(_PLACEHOLDER_RE.findall(template))


def fill(template: str, values: dict[str, str]) -> str:
    """Thay mỗi `{{biến}}` của template bằng giá trị của nó; chữ khác (kể cả `{{color, ground}}` của
    code) giữ nguyên."""
    return _PLACEHOLDER_RE.sub(lambda m: values[m.group(1)], template)


@dataclass(frozen=True)
class SystemPrompt:
    role: str
    name: str
    template_text: str


def load_system_prompts(directory: Path) -> tuple[list[SystemPrompt], list[str]]:
    """Đọc prompt hệ thống từ `<directory>/<role>.txt`. Trả các prompt có file và các role chưa có file."""
    found: list[SystemPrompt] = []
    missing: list[str] = []
    for role, name in SYSTEM_PROMPT_NAMES.items():
        path = directory / f"{role}.txt"
        if path.is_file():
            found.append(SystemPrompt(role=role, name=name, template_text=path.read_text(encoding="utf-8")))
        else:
            missing.append(role)
    return found, missing
