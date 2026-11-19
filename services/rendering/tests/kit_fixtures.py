"""The kit values the retired `phatsang` table held, kept as a fixture: tests that need a valid kit use it."""

import copy
import json
from pathlib import Path

FIXTURES = Path(__file__).parent / "fixtures"

#: SHA-256 of the PNG the old bundle drew for `sample-fig-wide` (frame 10) with the kit key `phatsang`.
PHATSANG_SAMPLE_FIG_WIDE_SHA256 = "8bb8dd6d3a63f13a309332589b19e792b681f86ad25cc9acf9d97aad4e410daa"


def phatsang_kit() -> dict:
    """A fresh copy of the phatsang kit, safe to edit."""
    return copy.deepcopy(json.loads((FIXTURES / "phatsang_kit.json").read_text(encoding="utf-8")))
