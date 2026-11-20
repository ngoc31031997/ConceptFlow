"""Luật của tài sản dự án: kiểm trường, đổi trạng thái, mô tả ghép cho bộ vẽ."""

from __future__ import annotations

import pytest

from domain.illustration import IllustrationError
from domain.project_asset import (
    AssetNoDetailError,
    AssetStateError,
    AssetStatus,
    ProjectAsset,
    apply_edit,
    describe_for_drawing,
    ensure_can_approve,
    ensure_can_draw,
    format_asset_id,
    normalize_asset,
    validate_asset_id,
)


def asset(**kw) -> ProjectAsset:
    base = {
        "project_id": "p1", "concept_id": "C1", "name": "Space", "portrayal": "Vũ trụ tối, vài ngôi sao",
        "must_not_imply": (), "tier": "background", "kind": "backdrop",
    }
    base.update(kw)
    return ProjectAsset(**base)


def test_asset_id_is_the_uppercased_name_with_a_three_digit_number():
    assert format_asset_id("SchoolBus", 2) == "AST-SCHOOLBUS-002"
    validate_asset_id("AST-SPACE-001")
    with pytest.raises(IllustrationError):
        validate_asset_id("../x")


@pytest.mark.parametrize("change", [
    {"name": "bad name"}, {"concept_id": " "}, {"kind": "sprite"}, {"tier": "epic"},
    {"project_id": "../p"},
])
def test_normalize_rejects_invalid_fields(change):
    with pytest.raises(IllustrationError):
        normalize_asset(asset(**change))


def test_normalize_trims_and_dedupes_lists():
    out = normalize_asset(asset(palette_tokens=(" a ", "a", "", "b")))
    assert out.palette_tokens == ("a", "b")


def test_approving_needs_a_drawn_asset_with_code():
    with pytest.raises(AssetStateError):
        ensure_can_approve(asset())
    ensure_can_approve(asset(status=AssetStatus.DRAWN, code="export function Space() {}"))


def test_editing_the_portrayal_of_a_drawn_or_approved_asset_sets_it_back_to_planned():
    for status in (AssetStatus.DRAWN, AssetStatus.APPROVED):
        existing = asset(status=status, code="x", version=2)
        edited = apply_edit(existing, existing.with_(portrayal="Vũ trụ sáng"))
        assert edited.status == AssetStatus.PLANNED and edited.code == "x" and edited.version == 2


def test_editing_only_library_fields_keeps_the_status():
    existing = asset(status=AssetStatus.APPROVED, code="x")
    edited = apply_edit(existing, existing.with_(to_library=False, library_folder_id="do-vat"))
    assert edited.status == AssetStatus.APPROVED and not edited.to_library


def test_description_joins_portrayal_constraints_palette_expressions_and_tier():
    text = describe_for_drawing(asset(
        must_not_imply=("trời ban ngày",), palette_tokens=("navy",), expressions=("vui",), tier="hero"))
    for part in ("Vũ trụ tối", "trời ban ngày", "navy", "vui", "hero"):
        assert part in text


def test_a_listed_asset_has_no_detail_until_it_has_a_portrayal():
    listed = normalize_asset(asset(portrayal="", units=(" VP-001 ", "VP-002", "VP-001", "")))
    assert listed.units == ("VP-001", "VP-002") and not listed.has_detail
    assert listed.to_json()["has_detail"] is False and listed.to_json()["units"] == ["VP-001", "VP-002"]
    assert not asset(portrayal="  ").has_detail and asset().has_detail
    with pytest.raises(AssetNoDetailError) as caught:
        ensure_can_draw(listed)
    assert caught.value.code == "asset_no_detail" and isinstance(caught.value, AssetStateError)
    ensure_can_draw(asset())


def test_editing_the_units_of_a_drawn_or_approved_asset_sets_it_back_to_planned():
    for status in (AssetStatus.DRAWN, AssetStatus.APPROVED):
        existing = asset(status=status, code="x", version=2, units=("VP-001",))
        edited = apply_edit(existing, existing.with_(units=("VP-001", "VP-002")))
        assert edited.status == AssetStatus.PLANNED and edited.code == "x"
        assert edited.units == ("VP-001", "VP-002")
    same = asset(status=AssetStatus.DRAWN, code="x", units=("VP-001",))
    assert apply_edit(same, same.with_(units=("VP-001",))).status == AssetStatus.DRAWN
