"""Marks the tests driven by the tooth mockup assets, so a run can deselect them
with `-m "not tooth_mockup"`.
"""

from pathlib import Path

import pytest

# Tests driven by the tooth mockup assets (layout-probe samples, illustration kit).
TOOTH_MOCKUP_FILES = (
    "test_layout_checker.py",
    "test_illustration_kit.py",
    "test_illustration_previewer.py",
)


def pytest_configure(config):
    config.addinivalue_line("markers", "tooth_mockup: driven by the tooth mockup assets")


def pytest_collection_modifyitems(items):
    for item in items:
        if Path(str(item.fspath)).name in TOOTH_MOCKUP_FILES:
            item.add_marker(pytest.mark.tooth_mockup)
