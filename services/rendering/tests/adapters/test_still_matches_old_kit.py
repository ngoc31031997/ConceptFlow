"""A still drawn from the kit object holding the old `phatsang` numbers equals the still the retired
kit table drew. Needs the Remotion bundle (node_modules) and a browser, like the other real-render tests."""

import hashlib
import os
import shutil
from pathlib import Path

import pytest

from adapters.rendering.illustration_previewer import IllustrationPreviewer
from application.stills import StillsUseCase
from tests.kit_fixtures import PHATSANG_SAMPLE_FIG_WIDE_SHA256, phatsang_kit

PROJECT = Path(__file__).parents[2] / "remotion_project"
needs_browser = pytest.mark.skipif(
    not (PROJECT / "node_modules" / "@remotion" / "renderer").exists()
    or shutil.which("node") is None
    or not os.environ.get("REMOTION_BROWSER"),
    reason="cần remotion_project/node_modules và REMOTION_BROWSER",
)


@needs_browser
def test_the_phatsang_kit_object_draws_the_same_still_as_the_old_kit_table(tmp_path):
    previewer = IllustrationPreviewer(PROJECT, timeout_seconds=240)
    try:
        # The old baseline was drawn with an extra, ignored prop to bypass the cache.
        out = StillsUseCase(previewer, tmp_path).render(
            "sample-fig-wide", phatsang_kit(), {"baseline_cr116": 1})
    finally:
        previewer.close()
    assert hashlib.sha256((tmp_path / out.file).read_bytes()).hexdigest() == PHATSANG_SAMPLE_FIG_WIDE_SHA256
