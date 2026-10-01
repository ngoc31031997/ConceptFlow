"""Backdrops reach the Remotion Engineer per shot, and a library backdrop used
as `backdrop={Name}` is pasted into the merged script."""

import json

from app import storyboard as sbm
from app.pipeline import merger
from app.pipeline.run import library_section

LIBRARY = [
    {"name": "SchoolBus", "usage": "<SchoolBus color /> — 320×200", "description": "xe buýt", "code": "x"},
    {"name": "MeadowBackdrop", "description": "đồng cỏ", "code": "", "kind": "backdrop", "shots": ["1.1", "1.2"]},
    {"name": "OrchardBackdrop", "description": "vườn sung", "code": "export function OrchardBackdrop() {}",
     "kind": "backdrop", "shots": ["2.1"]},
]


def test_backdrops_are_listed_with_their_shots_apart_from_the_figures():
    text = library_section(LIBRARY)
    figures, backdrops = text.split("## C5.")
    assert "<SchoolBus color />" in figures and "Backdrop" not in figures
    assert "- MeadowBackdrop — đồng cỏ — shot: 1.1, 1.2" in backdrops
    assert "- OrchardBackdrop — vườn sung — shot: 2.1" in backdrops
    assert "¤" not in text


def test_no_backdrop_section_without_backdrops():
    assert "C5." not in library_section(LIBRARY[:1])


def test_a_library_backdrop_used_by_a_scene_is_pasted_into_the_script():
    sb = sbm.parse(json.dumps({"hero": "h", "palette": [{"role": "a", "hex": "#000000"}],
                               "scenes": [{"id": "s", "shots": [{"id": "1.1", "visual": "v", "narration": "n"}]}]}))
    shot = {"1.1": "function Shot1_1({duration}: ShotProps) {\n"
                   "  return <Scene duration={duration} backdrop={OrchardBackdrop} />;\n}"}
    library = {"OrchardBackdrop": "import React from 'react';\nexport function OrchardBackdrop() {\n  return null;\n}\n",
               "Unused": "export function Unused() {}\n"}
    code = merger.merge_remotion(sb, "const LAYOUT = {};", shot, library=library).code
    assert "// Hình thư viện: OrchardBackdrop\nfunction OrchardBackdrop()" in code
    assert "Unused" not in code
