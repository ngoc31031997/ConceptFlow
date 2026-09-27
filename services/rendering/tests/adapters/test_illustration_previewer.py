"""Dựng xem trước thật một hình (CR-044). Cần node_modules và một trình duyệt."""

import base64
import os
import shutil
from pathlib import Path

import pytest

from adapters.rendering.illustration_previewer import IllustrationPreviewer
from adapters.rendering.typescript_checker import TypeScriptChecker
from application.preview_illustration import PreviewIllustrationUseCase
from tests.domain.test_illustration_asset import GOOD

PROJECT = Path(__file__).parents[2] / "remotion_project"
needs_browser = pytest.mark.skipif(
    not (PROJECT / "node_modules" / "@remotion" / "renderer").exists()
    or shutil.which("node") is None
    or not os.environ.get("REMOTION_BROWSER"),
    reason="cần remotion_project/node_modules và REMOTION_BROWSER",
)

PNG_MAGIC = b"\x89PNG"
GIF_MAGIC = b"GIF8"


@needs_browser
def test_dung_hinh_moi_hinh_co_san_va_bao_loi_code_hong():
    ts = TypeScriptChecker(PROJECT, timeout_seconds=90)
    previewer = IllustrationPreviewer(PROJECT, timeout_seconds=240)
    uc = PreviewIllustrationUseCase(ts, previewer)
    try:
        out = uc.run("SchoolBus", GOOD)
        assert out.ok, out.diagnostics
        assert base64.b64decode(out.png)[:4] == PNG_MAGIC
        assert base64.b64decode(out.gif)[:4] == GIF_MAGIC

        builtin = uc.run("Tooth", props={"decay": 0.6}, gif=False)
        assert builtin.ok and builtin.gif == ""

        bad = uc.run("SchoolBus", GOOD.replace("fill={color}", "fill={colour}"))
        assert not bad.ok and bad.diagnostics[0]["line"] == 9
    finally:
        ts.close()
        previewer.close()
