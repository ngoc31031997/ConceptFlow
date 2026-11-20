"""Dựng một khung storyboard nhiều hình + chữ. Cần node_modules và một trình duyệt."""

import asyncio
import json
import os
import shutil
import struct
from pathlib import Path

import pytest

from adapters.rendering.illustration_previewer import IllustrationPreviewer, IllustrationPreviewError
from adapters.storage.storyboard_files import StoryboardFiles
from application.storyboard import Storyboard
from domain.project_asset import AssetStatus, ProjectAsset
from domain.storyboard import build_frame_request
from tests.project_asset_fakes import FakeAssetRepo
from tests.storyboard_fakes import FakeStoryboardRepo

PROJECT = Path(__file__).parents[2] / "remotion_project"
KIT = json.loads((Path(__file__).parents[1] / "fixtures" / "phatsang_kit.json").read_text(encoding="utf-8"))
needs_browser = pytest.mark.skipif(
    not (PROJECT / "node_modules" / "@remotion" / "renderer").exists()
    or shutil.which("node") is None
    or not os.environ.get("REMOTION_BROWSER"),
    reason="cần remotion_project/node_modules và REMOTION_BROWSER",
)

# Hai hình thư viện (mã TSX được esbuild biên dịch như hình xem trước).
PERSON = "export {Person} from './conceptflow-mini/illustration';"
GERM = "export {Germ} from './conceptflow-mini/illustration';"


def _spec(**over) -> dict:
    spec = {
        "kit": KIT,
        "background": {"color": "#0F1B3D"},
        "placements": [
            {"code": PERSON, "name": "Person", "position": "lower_left", "tier": "hero"},
            {"code": GERM, "name": "Germ", "position": "upper_right", "tier": "support"},
        ],
        "camera_view": "wide",
        "on_screen_text": [
            {"text": "Vi khuẩn tấn công", "min_px": 44}, {"text": "Cơ thể phản ứng", "min_px": 32}],
        "text_color": "#FFFFFF",
    }
    spec.update(over)
    return spec


def _png_size(path: Path) -> tuple[int, int]:
    data = path.read_bytes()
    assert data[:4] == b"\x89PNG"
    return struct.unpack(">II", data[16:24])


@needs_browser
def test_khung_hai_hinh_va_chu_ra_png_1280x720_va_tu_choi_gia_tri_la(tmp_path):
    previewer = IllustrationPreviewer(PROJECT, timeout_seconds=240)
    try:
        out = tmp_path / "frame.png"
        previewer.frame(_spec(), str(out))
        assert _png_size(out) == (1280, 720)

        with pytest.raises(IllustrationPreviewError, match="position"):
            previewer.frame(_spec(placements=[{"code": GERM, "name": "Germ", "position": "nowhere"}]),
                            str(tmp_path / "bad.png"))
        with pytest.raises(IllustrationPreviewError, match="camera_view"):
            previewer.frame(_spec(camera_view="drone"), str(tmp_path / "bad2.png"))
    finally:
        previewer.close()


@needs_browser
async def test_use_case_draws_a_real_frame_from_approved_assets_and_caches_it(tmp_path):
    assets = FakeAssetRepo()
    drawings = (("AST-PERSON-001", "Person", PERSON, "hero"), ("AST-GERM-001", "Germ", GERM, "support"))
    for asset_id, name, code, tier in drawings:
        assets.rows[("p1", asset_id)] = ProjectAsset(
            project_id="p1", asset_id=asset_id, concept_id="C1", name=name, code=code, tier=tier, version=1,
            status=AssetStatus.APPROVED)
    previewer = IllustrationPreviewer(PROJECT, timeout_seconds=240)
    files = StoryboardFiles(tmp_path)
    storyboard = Storyboard(FakeStoryboardRepo(), assets, files, previewer, asyncio.Semaphore(1))
    request = build_frame_request(
        "p1", "SH-001", "VP-001", "B1", "c" * 64, KIT,
        [{"asset_id": "AST-PERSON-001", "role": "subject", "position": "lower_left"},
         {"asset_id": "AST-GERM-001", "role": "main", "position": "upper_right"}],
        "close", {"color": "#0F1B3D"}, [{"text": "Vi khuẩn tấn công", "min_px": 44}], "#FFFFFF")
    try:
        first = await storyboard.draw_frame(request)
        assert not first.cache_hit and _png_size(tmp_path / "p1" / first.shot.frame_path) == (1280, 720)
        assert len(first.shot.frame_hash) == 64
        assert (await storyboard.draw_frame(request)).cache_hit
    finally:
        previewer.close()
