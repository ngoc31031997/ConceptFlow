"""Bộ avatar mèo `cat.*` (CR-038): dựng lại được, hợp lệ, và khớp manifest."""

import importlib.util
import json
from pathlib import Path

import pytest

from domain import lottie_catalog as catalog

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location("build_avatar", ROOT / "tools" / "build_avatar.py")
build_avatar = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(build_avatar)

PUBLIC = ROOT / "remotion_project" / "public" / "lottie"
MANIFEST = ROOT / "remotion_project" / "lottie" / "manifest.json"


@pytest.fixture(scope="module")
def rig():
    return build_avatar.build_rig()


@pytest.mark.parametrize("state", list(build_avatar.STATES))
def test_moi_trang_thai_la_lottie_hop_le(rig, state):
    data = build_avatar.STATES[state][0](rig)
    assert all(k in data for k in ("v", "fr", "ip", "op", "layers"))
    assert data["op"] > 0
    assert all(layer["op"] == data["op"] for layer in data["layers"])
    json.dumps(data)  # tuần tự hoá được


@pytest.mark.parametrize("state", list(build_avatar.STATES))
def test_moi_trang_thai_co_ba_soc_muop(rig, state):
    data = build_avatar.STATES[state][0](rig)
    stripes = next(layer for layer in data["layers"] if layer["nm"] == "Stripes")
    lines = [i for i in stripes["shapes"][0]["it"] if i["ty"] == "sh"]
    assert len(lines) == 3


def test_chi_push_co_chan_va_ly(rig):
    for state, (fn, _, _) in build_avatar.STATES.items():
        names = {layer["nm"] for layer in fn(rig)["layers"]}
        assert ({"Lapa", "cup"} <= names) == (state == "push")


def test_duoi_nam_duoi_than_va_moi_lop_co_cha_null(rig):
    data = build_avatar.STATES["idle"][0](rig)
    order = [layer["nm"] for layer in data["layers"]]
    assert order.index("body") < order.index("tale")
    movers = [layer for layer in data["layers"] if layer["ty"] == 3]
    assert len(movers) == 1
    assert all(layer.get("parent") == movers[0]["ind"] for layer in data["layers"] if layer["ty"] != 3)


def test_dung_lai_cho_ket_qua_giong_nhau():
    a = build_avatar.STATES["happy"][0](build_avatar.build_rig())
    b = build_avatar.STATES["happy"][0](build_avatar.build_rig())
    assert a == b


def test_khong_sua_ban_goc_rig(rig):
    before = json.dumps(rig, sort_keys=True)
    for fn, _, _ in build_avatar.STATES.values():
        fn(rig)
    assert json.dumps(rig, sort_keys=True) == before


def test_file_da_commit_khop_trinh_dung(rig):
    """File trong public/lottie phải là đúng thứ trình dựng sinh ra (không bị sửa tay)."""
    for state, (fn, _, _) in build_avatar.STATES.items():
        expected = json.dumps(fn(rig), separators=(",", ":"))
        assert (PUBLIC / f"cat.{state}.json").read_text(encoding="utf-8") == expected, state


def test_manifest_that_hop_le_va_khop_bo_avatar():
    assets = catalog.load_manifest(MANIFEST)
    assert catalog.validate(assets, PUBLIC) == []
    assert {a.id for a in assets if a.id.startswith("cat.")} == {f"cat.{s}" for s in build_avatar.STATES}


def test_avatar_da_duyet_phai_ghi_ngay_kiem_giay_phep():
    """Clip đã duyệt là do Creator xác nhận giấy phép riêng, có ngày kiểm."""
    assets = [a for a in catalog.load_manifest(MANIFEST) if a.id.startswith("cat.")]
    assert assets and all(a.is_approved and a.license_checked for a in assets)


def test_avatar_da_duyet_vao_khoi_prompt():
    assets = catalog.load_manifest(MANIFEST)
    block = catalog.render_prompt_block(assets)
    for state in build_avatar.STATES:
        assert f"cat.{state}" in block


def test_file_prompt_cua_orchestrator_khop_manifest():
    """lottie_catalog_vi.txt là file SINH RA; sửa manifest mà quên chạy `tools/lottie_catalog.py prompt`
    thì prompt đang chạy sẽ lệch danh mục thật."""
    target = ROOT.parent / "orchestrator" / "internal" / "domain" / "prompts" / "lottie_catalog_vi.txt"
    if not target.exists():  # trong image rendering không có mã orchestrator
        pytest.skip("không có mã orchestrator bên cạnh")
    expected = catalog.render_prompt_block(catalog.load_manifest(MANIFEST), "vi")
    assert target.read_text(encoding="utf-8") == expected
