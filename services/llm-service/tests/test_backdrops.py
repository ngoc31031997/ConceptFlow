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


def test_a_library_figure_shows_the_prop_types_its_code_declares():
    door = {"name": "NarrowDoor", "usage": "<NarrowDoor color open />", "description": "cửa hẹp",
            "code": "type DoorOpen = 'closed' | 'ajar' | 'open';\n"
                    "export function NarrowDoor({color = '#E63946', open = 'ajar', ...fig}: FigureProps & {\n"
                    "  color?: string;\n  open?: DoorOpen;\n}) {\n  return null;\n}\n"}
    text = library_section([door, LIBRARY[0]])
    assert "- <NarrowDoor color open /> — cửa hẹp\n  props: color?: string; open?: 'closed' | 'ajar' | 'open'\n" in text
    # A drawing whose props cannot be read keeps its usage line alone.
    assert text.rstrip().endswith("- <SchoolBus color /> — 320×200 — xe buýt")


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
    assert "// Hình thư viện: OrchardBackdrop\nnamespace Library_OrchardBackdrop {\nexport function OrchardBackdrop()" in code
    assert "const OrchardBackdrop = Library_OrchardBackdrop.OrchardBackdrop;" in code
    assert "Unused" not in code


def _import_lines(code: str) -> list[str]:
    return [line for line in code.splitlines() if line.startswith("import ")]


def test_a_pasted_drawing_brings_the_imports_the_frame_lacks_without_duplicates():
    sb = sbm.parse(json.dumps({"hero": "h", "palette": [{"role": "a", "hex": "#000000"}],
                               "scenes": [{"id": "s", "shots": [{"id": "1.1", "visual": "v", "narration": "n"}]}]}))
    shot = {"1.1": "function Shot1_1({duration}: ShotProps) {\n"
                   "  return <Scene duration={duration} backdrop={StarBackdrop} />;\n}"}
    library = {"StarBackdrop": "import React from 'react';\n"
                               "import {useVideoConfig, random, measureSpring} from 'remotion';\n"
                               "import {\n  shadeOf,\n  useSvgId,\n} from './conceptflow-mini/illustration';\n"
                               "import {useLayerBox} from './conceptflow-mini/scene';\n"
                               "export function StarBackdrop() {\n  return random('star') > 2 ? <g /> : null;\n}\n"}
    imports = _import_lines(merger.merge_remotion(sb, "const LAYOUT = {};", shot, library=library).code)
    remotion = [line for line in imports if line.endswith("from 'remotion';")]
    assert len(remotion) == 1
    assert remotion[0].endswith(", useVideoConfig, random, measureSpring} from 'remotion';")
    assert remotion[0].count("useVideoConfig") == 1 and remotion[0].count("random") == 1
    # Every other line is the frame's own: nothing the frame imports is imported twice.
    assert [l for l in imports if l not in remotion] == \
        [l for l in merger._REMOTION_HEAD.strip().split("\n") if not l.endswith("from 'remotion';")]


def test_a_drawing_import_from_a_module_the_frame_lacks_gets_its_own_line():
    head = "import React from 'react';\n"
    library = {"Clock": "import {clsx} from 'clsx';\nimport Lottie from 'lottie';\nexport function Clock() {}\n"}
    out = merger.head_with_library_imports(head, library, {"1.1": "<Clock />"})
    assert out == "import React from 'react';\nimport {clsx} from 'clsx';\nimport Lottie from 'lottie';\n"
