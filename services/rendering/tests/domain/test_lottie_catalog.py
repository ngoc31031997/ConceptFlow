"""Danh mục Lottie (CR-038): manifest, cổng giấy phép, khối prompt, lint id."""

import json

import pytest

from domain.lottie_catalog import (
    APPROVED,
    CANDIDATE,
    CatalogError,
    LottieAsset,
    approved,
    credits,
    extract_palette,
    lint_lottie_ids,
    load_manifest,
    render_prompt_block,
    validate,
)

LOTTIE_MIN = {"v": "5.7.0", "fr": 30, "ip": 0, "op": 60, "w": 512, "h": 512, "layers": []}


def asset(**over) -> LottieAsset:
    base = dict(
        id="cat.idle",
        title="Mèo ngồi",
        description="Mèo mướp ngồi yên, đuôi vẫy nhẹ",
        license="CC0",
        source_url="https://example.test/cat",
        status=APPROVED,
        license_checked="2026-09-25",
    )
    base.update(over)
    return LottieAsset(**base)


def write_manifest(tmp_path, entries):
    path = tmp_path / "manifest.json"
    path.write_text(json.dumps({"assets": entries}), encoding="utf-8")
    return path


def test_thieu_manifest_la_danh_muc_rong(tmp_path):
    assert load_manifest(tmp_path / "khong-co.json") == []


def test_manifest_thieu_truong_bat_buoc(tmp_path):
    path = write_manifest(tmp_path, [{"id": "cat.idle", "title": "x"}])
    with pytest.raises(CatalogError, match="thiếu"):
        load_manifest(path)


def test_manifest_mac_dinh_la_candidate(tmp_path):
    path = write_manifest(
        tmp_path,
        [{"id": "cat.idle", "title": "t", "description": "d", "license": "CC0", "source_url": "u"}],
    )
    (only,) = load_manifest(path)
    assert only.status == CANDIDATE
    assert approved([only]) == []


def test_validate_chan_giay_phep_la_va_thieu_ghi_cong(tmp_path):
    (tmp_path / "cat.idle.json").write_text(json.dumps(LOTTIE_MIN))
    problems = validate([asset(license="All Rights Reserved")], tmp_path)
    assert any("giấy phép" in p for p in problems)
    problems = validate([asset(license="CC-BY")], tmp_path)
    assert any("attribution" in p for p in problems)
    assert validate([asset(license="CC-BY", attribution="Tác giả A")], tmp_path) == []


def test_validate_thieu_file_va_file_khong_phai_lottie(tmp_path):
    assert any("thiếu file" in p for p in validate([asset()], tmp_path))
    (tmp_path / "cat.idle.json").write_text(json.dumps({"hello": 1}))
    assert any("không giống file Lottie" in p for p in validate([asset()], tmp_path))


def test_validate_id_trung_va_sai_dang(tmp_path):
    (tmp_path / "cat.idle.json").write_text(json.dumps(LOTTIE_MIN))
    problems = validate([asset(), asset()], tmp_path)
    assert any("trùng" in p for p in problems)
    assert any("id phải" in p for p in validate([asset(id="Cat Idle")], tmp_path))


def test_khoi_prompt_rong_noi_ro_khong_co_clip():
    assert "Chưa có clip" in render_prompt_block([])
    assert "Chưa có clip" in render_prompt_block([asset(status=CANDIDATE)])


def test_khoi_prompt_chi_gom_clip_da_duyet_va_sap_xep():
    block = render_prompt_block(
        [
            asset(id="cat.thinking", description="Mèo nghiêng đầu"),
            asset(id="cat.candidate", status=CANDIDATE),
            asset(id="cat.idle", palette=("#F5B841",), loop=False),
        ]
    )
    assert "cat.candidate" not in block
    assert block.index("cat.idle") < block.index("cat.thinking")
    assert "#F5B841" in block and "chạy một lần" in block


def test_credits_chi_clip_can_ghi_cong():
    lines = credits(
        [asset(id="a.x", license="CC-BY", attribution="Tác giả A"), asset(id="b.y", license="CC0")]
    )
    assert len(lines) == 1 and "Tác giả A" in lines[0]


def test_extract_palette_tinh_va_keyframe():
    data = {
        "layers": [
            {"shapes": [{"ty": "fl", "c": {"a": 0, "k": [1, 0.5, 0, 1]}}]},
            {"shapes": [{"ty": "st", "c": {"a": 1, "k": [{"s": [0, 0, 1, 1]}, {"s": [1, 1, 1, 1]}]}}]},
        ]
    }
    assert extract_palette(data) == {"#FF8000", "#0000FF", "#FFFFFF"}


def test_lint_id_ngoai_danh_muc_bi_chan_kem_so_dong():
    script = 'const a = 1;\n<LottieClip id="cat.idle" x={1} />\n<LottieClip id="cat.ghost" />\n'
    issues = lint_lottie_ids(script, {"cat.idle"})
    assert len(issues) == 1
    assert issues[0][0] == 3 and "cat.ghost" in issues[0][1]


def test_lint_nhan_ca_dau_nhay_don_va_dang_bieu_thuc():
    script = "<LottieClip id='a.b' />\n<LottieClip x={1} id={\"c.d\"} />\n"
    assert {i[1].split("'")[1] for i in lint_lottie_ids(script, set())} == {"a.b", "c.d"}


def test_lint_danh_muc_rong_thi_moi_clip_deu_sai():
    ((_, message),) = lint_lottie_ids('<LottieClip id="cat.idle" />', set())
    assert "danh mục đang rỗng" in message


def test_clip_da_duyet_phai_ghi_ngay_kiem_giay_phep(tmp_path):
    (tmp_path / "cat.idle.json").write_text(json.dumps(LOTTIE_MIN))
    assert any("license_checked" in p for p in validate([asset(license_checked="")], tmp_path))
    # Ứng viên chưa cần: kiểm giấy phép là điều kiện để DUYỆT.
    assert validate([asset(status=CANDIDATE, license_checked="")], tmp_path) == []
