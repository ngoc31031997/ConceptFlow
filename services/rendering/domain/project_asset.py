"""Tài sản của một dự án ở bước Storyboard: một hình hoặc nền mà các shot sẽ đặt vào khung.

Tài sản đi từ `planned` (mới lập, chưa có hình) qua `drawn` (AI đã vẽ, có code và ảnh xem trước) tới
`approved` (Creator duyệt, shot được dùng). Sửa phần mô tả của tài sản đã vẽ hay đã duyệt đưa nó về
`planned`, vì hình đang có không còn là thứ mô tả nói. Tài sản mới lập chỉ có phần khung (khái niệm, tên,
loại, hạng, các unit nó xuất hiện); phần chi tiết (`portrayal` và các trường đi kèm) được viết sau, và chỉ khi
đã có `portrayal` thì tài sản mới vẽ được. Mã `AST-<TÊN>-<số>` do máy cấp từ bộ đếm theo dự án
và không bao giờ dùng lại.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field, replace
from enum import StrEnum

from domain.illustration import (
    TIER_BACKGROUND,
    TIER_HERO,
    TIER_SUPPORT,
    CodeFinding,
    IllustrationError,
    IllustrationKind,
    validate_illustration_name,
)

ASSET_ID_RE = re.compile(r"^AST-[A-Z0-9]{2,41}-\d{3,}$")
PROJECT_ID_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$")
TIERS = (TIER_HERO, TIER_SUPPORT, TIER_BACKGROUND)
KINDS = (IllustrationKind.FIGURE.value, IllustrationKind.BACKDROP.value)
#: Thư mục thư viện nhận hình của dự án khi Creator không chọn thư mục khác.
PROJECT_FOLDER_ID = "tu-du-an"
PROJECT_FOLDER_NAME = "Từ dự án"
#: Độ dài mã dự án gắn vào tên hình trong thư viện.
PROJECT_SHORT_ID_LEN = 8


class AssetStatus(StrEnum):
    PLANNED = "planned"
    DRAWN = "drawn"
    APPROVED = "approved"


class AssetNotFoundError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("asset not found")


class AssetStateError(IllustrationError):
    """Thao tác không hợp lệ với trạng thái hiện tại của tài sản (vd duyệt khi chưa vẽ)."""


class AssetNoDetailError(AssetStateError):
    """Vẽ tài sản chưa có chi tiết (`portrayal` trống)."""

    code = "asset_no_detail"

    def __init__(self) -> None:
        super().__init__("tài sản chưa có chi tiết — viết chi tiết (portrayal) trước khi vẽ")


@dataclass(frozen=True)
class ProjectAsset:
    project_id: str
    asset_id: str = ""
    concept_id: str = ""
    name: str = ""  # component được export, PascalCase
    kind: str = IllustrationKind.FIGURE.value
    tier: str = TIER_SUPPORT
    portrayal: str = ""
    must_not_imply: tuple[str, ...] = ()
    palette_tokens: tuple[str, ...] = ()
    expressions: tuple[str, ...] = ()
    base_illustration_id: str = ""
    #: Mã unit kế hoạch hình (VP-…) mà tài sản xuất hiện; Creator sửa được.
    units: tuple[str, ...] = ()
    code: str = ""
    version: int = 0
    preview_path: str = ""
    status: AssetStatus = AssetStatus.PLANNED
    to_library: bool = True
    library_folder_id: str = ""
    library_illustration_id: str = ""
    warnings: tuple[CodeFinding, ...] = field(default_factory=tuple)
    created_at: str = ""
    updated_at: str = ""

    @property
    def has_detail(self) -> bool:
        """Đã có chi tiết khi `portrayal` không trống; chỉ khi đó mới vẽ được."""
        return bool(self.portrayal.strip())

    def with_(self, **changes: object) -> ProjectAsset:
        return replace(self, **changes)

    def to_json(self) -> dict:
        return {
            "project_id": self.project_id, "asset_id": self.asset_id, "concept_id": self.concept_id,
            "name": self.name, "kind": self.kind, "tier": self.tier, "portrayal": self.portrayal,
            "must_not_imply": list(self.must_not_imply), "palette_tokens": list(self.palette_tokens),
            "expressions": list(self.expressions), "base_illustration_id": self.base_illustration_id,
            "units": list(self.units), "has_detail": self.has_detail, "code": self.code,
            "version": self.version, "preview_path": self.preview_path,
            "status": self.status.value, "to_library": self.to_library,
            "library_folder_id": self.library_folder_id,
            "library_illustration_id": self.library_illustration_id,
            "warnings": [w.to_json() for w in self.warnings],
            "created_at": self.created_at, "updated_at": self.updated_at,
        }


#: Trường mà đổi chúng làm hình đang có không còn khớp mô tả.
DRAWING_FIELDS = (
    "name", "kind", "tier", "portrayal", "must_not_imply", "palette_tokens", "expressions",
    "base_illustration_id", "units",
)


def validate_project_id(project_id: str) -> None:
    if not PROJECT_ID_RE.match(project_id):
        raise IllustrationError(f"mã dự án không hợp lệ: {project_id!r}")


def validate_asset_id(asset_id: str) -> None:
    if not ASSET_ID_RE.match(asset_id):
        raise IllustrationError(f"mã tài sản không hợp lệ: {asset_id!r}")


def name_key(name: str) -> str:
    """Khoá bộ đếm và phần `<TÊN>` của mã tài sản: tên component viết hoa."""
    return name.upper()


def format_asset_id(name: str, number: int) -> str:
    return f"AST-{name_key(name)}-{number:03d}"


def _clean_list(values: list[str] | tuple[str, ...]) -> tuple[str, ...]:
    """Cắt khoảng trắng, bỏ mục trống và mục trùng, giữ thứ tự."""
    out: list[str] = []
    for v in values:
        v = v.strip()
        if v and v not in out:
            out.append(v)
    return tuple(out)


def normalize_asset(asset: ProjectAsset) -> ProjectAsset:
    """Chuẩn hoá và kiểm các trường người dùng nhập; ném IllustrationError nêu trường sai."""
    validate_project_id(asset.project_id)
    name = asset.name.strip()
    validate_illustration_name(name)
    if not asset.concept_id.strip():
        raise IllustrationError("concept_id là bắt buộc")
    if asset.kind not in KINDS:
        raise IllustrationError(f'kind "{asset.kind}" phải là figure hoặc backdrop')
    if asset.tier not in TIERS:
        raise IllustrationError(f'tier "{asset.tier}" phải là hero, support hoặc background')
    return asset.with_(
        name=name, concept_id=asset.concept_id.strip(), portrayal=asset.portrayal.strip(),
        must_not_imply=_clean_list(asset.must_not_imply), palette_tokens=_clean_list(asset.palette_tokens),
        expressions=_clean_list(asset.expressions), units=_clean_list(asset.units),
        base_illustration_id=asset.base_illustration_id.strip(),
        library_folder_id=asset.library_folder_id.strip())


def apply_edit(existing: ProjectAsset, edited: ProjectAsset) -> ProjectAsset:
    """Áp phần sửa của Creator lên `existing`. Đổi một trường mô tả của tài sản đã vẽ hay đã duyệt đưa nó
    về `planned`; code, phiên bản và ảnh giữ nguyên tới lần vẽ lại."""
    edited = normalize_asset(edited)
    merged = existing.with_(
        concept_id=edited.concept_id, name=edited.name, kind=edited.kind, tier=edited.tier,
        portrayal=edited.portrayal, must_not_imply=edited.must_not_imply,
        palette_tokens=edited.palette_tokens, expressions=edited.expressions,
        base_illustration_id=edited.base_illustration_id, units=edited.units, to_library=edited.to_library,
        library_folder_id=edited.library_folder_id)
    drawing_changed = any(getattr(merged, f) != getattr(existing, f) for f in DRAWING_FIELDS)
    if drawing_changed and existing.status != AssetStatus.PLANNED:
        merged = merged.with_(status=AssetStatus.PLANNED)
    return merged


def ensure_can_draw(asset: ProjectAsset) -> None:
    if not asset.has_detail:
        raise AssetNoDetailError()


def ensure_can_approve(asset: ProjectAsset) -> None:
    if asset.status == AssetStatus.PLANNED or not asset.code.strip():
        raise AssetStateError("tài sản chưa được vẽ — vẽ xong mới duyệt được")


def describe_for_drawing(asset: ProjectAsset) -> str:
    """Mô tả cho bộ vẽ AI, ghép từ cách thể hiện, điều không được gợi ra, màu, biểu cảm và hạng."""
    tier_text = {
        TIER_HERO: "nhân vật chính của khung, vẽ công phu",
        TIER_SUPPORT: "hình phụ, vẽ vừa phải",
        TIER_BACKGROUND: "phông, vẽ đơn giản",
    }[asset.tier]
    lines = [asset.portrayal]
    if asset.must_not_imply:
        lines.append("Không được gợi ra: " + "; ".join(asset.must_not_imply) + ".")
    if asset.palette_tokens:
        lines.append("Chỉ dùng các màu của bảng màu dự án: " + ", ".join(asset.palette_tokens) + ".")
    if asset.expressions:
        lines.append("Cần các biểu cảm/trạng thái: " + ", ".join(asset.expressions) + ".")
    lines.append(f"Hạng trong video: {asset.tier} ({tier_text}).")
    return "\n".join(lines)
