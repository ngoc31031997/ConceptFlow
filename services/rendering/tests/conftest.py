"""Marks the tests CI deselects for now (Creator decision, 2026-09-28): they pass
locally, and the parts they cover may be dropped later. `make check` still runs
them; CI passes `-m "not manim and not tooth_mockup"` (.github/workflows/ci.yml).
"""

from pathlib import Path

import pytest

# The ConceptFlow design system renders through real manim.
MANIM_DIRS = ("conceptflow",)
# Tests driven by the tooth mockup assets (layout-probe samples, illustration kit).
TOOTH_MOCKUP_FILES = (
    "test_layout_checker.py",
    "test_illustration_kit.py",
    "test_illustration_previewer.py",
)


def pytest_configure(config):
    config.addinivalue_line("markers", "manim: needs the manim-based ConceptFlow design system")
    config.addinivalue_line("markers", "tooth_mockup: driven by the tooth mockup assets")


def pytest_collection_modifyitems(items):
    tests_root = Path(__file__).parent
    for item in items:
        path = Path(str(item.fspath))
        rel = path.relative_to(tests_root) if path.is_relative_to(tests_root) else path
        if rel.parts and rel.parts[0] in MANIM_DIRS:
            item.add_marker(pytest.mark.manim)
        if path.name in TOOTH_MOCKUP_FILES:
            item.add_marker(pytest.mark.tooth_mockup)
