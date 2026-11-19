"""Biến và đoạn chữ của prompt bộ vẽ; domain thư viện hình."""

from __future__ import annotations

from domain import drawer_prompts as dp
from domain.illustration import (
    BUILTIN_DETAIL,
    DetailLevel,
    IllustrationKind,
    builtin_illustrations,
    exemplar_copy_name,
    normalize_tags,
    rename_export,
    system_folders,
    tier_for_detail,
)
from tests.illustration_fakes import STYLE


def test_style_values_take_the_reference_styles_own_words():
    values = dp.style_values(STYLE)
    assert set(values) == set(dp.STYLE_VARIABLES)
    assert values["style_name"] == "Khoa học phát sáng" and values["drawing_shading"] == "theo nguồn sáng"
    assert values["layout_persistent_elements"] == ""


def test_a_style_with_no_detail_gives_empty_values_not_errors():
    values = dp.style_values({"id": "R01", "name": "Tên"})
    assert (
        values["style_name"] == "Tên" and values["drawing_medium"] == "" and values["layout_structure"] == ""
    )


def test_fill_replaces_only_known_placeholders_and_keeps_code_braces():
    assert dp.fill("a {{x}} b {{color, ground}} {{y}}", {"x": "1", "y": "2"}) == "a 1 b {{color, ground}} 2"
    assert dp.placeholders("{{a}} {{b_1}} {{color, ground}}") == {"a", "b_1"}


def test_roles_variables_and_helpers_follow_the_kind():
    assert dp.role_for(IllustrationKind.FIGURE) == "illustration_drawer"
    assert dp.role_for(IllustrationKind.BACKDROP) == "illustration_backdrop_drawer"
    assert "tier" in dp.variables_for(IllustrationKind.FIGURE)
    assert "tier" not in dp.variables_for(IllustrationKind.BACKDROP)
    assert "<Figure" in dp.helpers_text(IllustrationKind.FIGURE)
    assert dp.helpers_text(IllustrationKind.BACKDROP) != dp.helpers_text(IllustrationKind.FIGURE)


def test_the_tier_instruction_names_the_role_of_the_drawing():
    assert "HẠNG CHÍNH (hero)" in dp.tier_instruction("hero")
    assert "HẠNG PHÔNG (background)" in dp.tier_instruction("background")
    assert "HẠNG PHỤ (support)" in dp.tier_instruction("support")
    assert "HẠNG PHỤ (support)" in dp.tier_instruction("lạ")


def test_system_prompts_load_from_role_named_files_and_report_the_missing_ones(tmp_path):
    (tmp_path / "illustration_drawer.txt").write_text("TEMPLATE", encoding="utf-8")
    found, missing = dp.load_system_prompts(tmp_path)
    assert [(p.role, p.template_text) for p in found] == [("illustration_drawer", "TEMPLATE")]
    assert found[0].name == dp.SYSTEM_PROMPT_NAMES["illustration_drawer"]
    assert missing == ["illustration_backdrop_drawer"]


def test_both_approved_system_prompts_ship_and_use_only_known_variables():
    directory = dp.Path(dp.__file__).parent / "system_prompts"
    found, missing = dp.load_system_prompts(directory)
    assert missing == [] and {p.role for p in found} == set(dp.SYSTEM_PROMPT_NAMES)
    allowed = {
        dp.ROLE_FIGURE: set(dp.FIGURE_VARIABLES), dp.ROLE_BACKDROP: set(dp.BACKDROP_VARIABLES)}
    for prompt in found:
        assert dp.placeholders(prompt.template_text) <= allowed[prompt.role], prompt.role


def test_builtins_are_the_34_figures_and_6_backdrops_of_the_kit():
    rows = builtin_illustrations()
    assert len(rows) == 40 and sum(r.kind == IllustrationKind.BACKDROP for r in rows) == 6
    assert len({r.name for r in rows}) == 40 and all(r.builtin and r.id == "builtin-" + r.name for r in rows)
    folders = {f.id for f in system_folders()}
    assert {r.folder_id for r in rows} <= folders
    assert {r.name: r.detail_level for r in rows if r.name in BUILTIN_DETAIL} == BUILTIN_DETAIL
    assert next(r for r in rows if r.name == "Person").detail_level == DetailLevel.RICH


def test_every_builtin_is_exported_by_the_kit_source():
    from pathlib import Path

    project = Path(__file__).resolve().parents[2] / "remotion_project" / "src" / "conceptflow-mini"
    source = "".join(p.read_text(encoding="utf-8") for p in project.glob("*.tsx"))
    missing = [
        r.name
        for r in builtin_illustrations()
        if f"export function {r.name}" not in source and f"export const {r.name}" not in source
    ]
    assert missing == []


def test_system_folders_are_numbered_from_one_with_the_exemplar_folder_first():
    folders = system_folders()
    assert [f.position for f in folders] == list(range(1, 15)) and folders[0].id == "hinh-mau"
    assert all(f.is_system for f in folders)


def test_tags_are_trimmed_lowercased_and_deduplicated_in_order():
    assert normalize_tags([" Xe ", "xe", "", "Máy Bay", "xe"]) == ("xe", "máy bay")


def test_exemplar_copy_names_count_up_and_stay_within_the_name_length():
    assert exemplar_copy_name("Bus", set()) == "BusMau"
    assert exemplar_copy_name("Bus", {"BusMau", "BusMau2"}) == "BusMau3"
    long = "A" * 41
    assert len(exemplar_copy_name(long, set())) == 41


def test_rename_export_changes_the_function_and_the_usage_tags_only():
    code, usage = rename_export(
        "export function Bus({color}) {}\n// Bus", "<Bus color /> <BusStop />", "BusMau"
    )
    assert code == "export function BusMau({color}) {}\n// Bus"
    assert usage == "<BusMau color /> <BusStop />"


def test_tier_for_detail_maps_every_level():
    assert [tier_for_detail(d) for d in ("rich", "medium", "silhouette", "", "x")] == [
        "hero",
        "support",
        "background",
        "support",
        "support",
    ]
