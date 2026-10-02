"""Code Merger — deterministic, no LLM.

Everything that must agree with the storyboard by construction is generated
here: the narrations array, the shot list and its order, the composition
registration / the Scene class. The model only writes the shot bodies, so the
"narrations.length != SHOTS.length" class of failure cannot occur.
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass

from app.frame import LANDSCAPE, Frame
from app.pipeline import naming
from app.storyboard import Storyboard

REMOTION_SHOT = re.compile(r"^function Shot(\d+)_(\d+)\s*\(")
MANIM_SHOT = re.compile(r"^def shot_(\d+)_(\d+)\s*\(self")

# Frame sections that are model-written but shared by every shot; they appear in
# `Merged.lines` next to the shot ids so an error inside them can be repaired too.
LAYOUT_KEY = "layout"
CAST_KEY = "cast"


@dataclass
class Merged:
    code: str
    # shot id -> (first line, last line), 1-based, inclusive, in the merged file
    lines: dict[str, tuple[int, int]]
    scene_class_name: str = ""

    def shot_at(self, line: int) -> str | None:
        for shot_id, (a, b) in self.lines.items():
            if a <= line <= b:
                return shot_id
        return None


def remotion_fn(shot_id: str) -> str:
    a, b = shot_id.split(".")
    return f"Shot{a}_{b}"


def manim_fn(shot_id: str) -> str:
    a, b = shot_id.split(".")
    return f"shot_{a}_{b}"


def palette_keys(sb: Storyboard) -> dict[str, str]:
    """role -> the camelCase key it gets in PALETTE."""
    keys = naming.unique([naming.camel(p.role) for p in sb.palette])
    return {p.role: k for p, k in zip(sb.palette, keys, strict=True)}


# Every component of conceptflow-mini/illustration.tsx. The prompt
# tells the model the whole kit is already imported, so the frame imports all
# of it; rendering/tests/domain/test_illustration_kit.py holds this list to the
# file's real exports.
ILLUSTRATION_KIT = (
    "Backdrop", "Panel", "Person", "Tooth", "Germ", "OpenMouth", "Toothbrush", "Toothpaste",
    "Drop", "Shield", "Heart", "Candy", "Lollipop", "Soda", "Donut", "Apple", "Clock", "Table",
    "Chair", "Window", "Plant", "House", "Tree", "Sun", "Cloud", "Lightbulb", "Coin", "Book",
    "Phone", "Magnifier", "Mark", "Sparkle", "Airplane", "Bubble",
)
# Every component of conceptflow-mini/scene.tsx: the depth layers, the camera,
# the light and the keyword text. Held to the file by the same test.
SCENE_KIT = ("Scene", "Camera", "Glow", "LightRays", "Vignette", "KeywordText")
# The building blocks a library drawing (pasted into the script) uses.
ILLUSTRATION_HELPERS = (
    "Figure", "Face", "GroundShadow", "useBlink", "phaseOf", "shadeOf", "useSvgId",
    "INK", "SHADE", "BLUSH", "WHITE",
)
# Every backdrop of conceptflow-mini/backdrops.tsx: whole-frame places a Scene
# draws in its depth layers. Held to the file by the same test.
BACKDROP_KIT = (
    "MeadowBackdrop", "RoomBackdrop", "StreetBackdrop", "InsideBodyBackdrop", "UnderwaterBackdrop",
    "SpaceBackdrop",
)
# Hooks of the scene kit a library backdrop drawing uses.
SCENE_HELPERS = ("useCamera", "useLayerBox")

_REMOTION_HEAD = """import React from 'react';
import {registerRoot, Composition, AbsoluteFill, interpolate, interpolateColors, spring, Easing, useCurrentFrame, useVideoConfig} from 'remotion';
import {calculateMetadataFromSegments, Segments} from './conceptflow-mini/segments';
import {Stage, SAFE_MARGIN, WIDTH, HEIGHT} from './conceptflow-mini/primitives';
import {LottieClip} from './conceptflow-mini/lottie';
""" + "import {" + ", ".join(ILLUSTRATION_KIT + ILLUSTRATION_HELPERS) + "} from './conceptflow-mini/illustration';\n" + (
    "import type {FigureProps, Mood, PersonPose} from './conceptflow-mini/illustration';\n"
    "import {" + ", ".join(SCENE_KIT + SCENE_HELPERS) + "} from './conceptflow-mini/scene';\n"
    "import {" + ", ".join(BACKDROP_KIT) + "} from './conceptflow-mini/backdrops';\n"
)

_REMOTION_TAIL = """
function CreatorComposition({segments = []}: {segments?: {startFrame: number; durationInFrames: number}[]}) {
  return (
    <Stage>
      <Segments segments={segments}>
        {(index, segment) => {
          const Shot = SHOTS[index];
          return Shot ? <Shot duration={segment.durationInFrames} /> : null;
        }}
      </Segments>
    </Stage>
  );
}

registerRoot(() => (
  <Composition
    id="creator"
    component={CreatorComposition}
    width={WIDTH_PX}
    height={HEIGHT_PX}
    fps={30}
    durationInFrames={150}
    calculateMetadata={calculateMetadataFromSegments}
  />
));
"""


def remotion_frame_text(sb: Storyboard) -> dict[str, str]:
    """The fixed pieces the model is told already exist, so it never redeclares them."""
    keys = palette_keys(sb)
    palette = "const PALETTE = {\n" + "".join(
        f"  {keys[p.role]}: '{p.hex}', // {p.meaning or p.role}\n" for p in sb.palette
    ) + "};"
    return {"palette": palette, "head": _REMOTION_HEAD.strip()}


def _layout_number(v: int | float) -> str:
    return str(int(v)) if float(v).is_integer() else repr(float(v))


def layout_from_storyboard(sb: Storyboard) -> str | None:
    """`const LAYOUT = {...};` built from the storyboard's own `layout`, or None
    when the storyboard has none (the pipeline then asks a model for it)."""
    if sb.layout is None:
        return None
    rows = [
        f"  {key}: {{" + ", ".join(f"{f}: {_layout_number(v)}" for f, v in entry.items()) + "},"
        for key, entry in sb.layout.items()
    ]
    return "const LAYOUT = {\n" + "".join(r + "\n" for r in rows) + "};"


def remotion_stub(shot_id: str) -> str:
    """Placeholder for a shot another chunk owns, so one chunk can be
    type-checked on its own before the rest exist."""
    return f"function {remotion_fn(shot_id)}({{duration}}: ShotProps) {{\n  return null;\n}}"


_IMPORT_LINE = re.compile(r"^\s*import\b[^;]*;?\s*$", re.M)
_TAG = re.compile(r"<([A-Z][A-Za-z0-9]*)\b|backdrop=\{([A-Z][A-Za-z0-9]*)\}")
_EXPORT = re.compile(r"^\s*export\s+(function|const|type|interface)\s+([A-Za-z_$][\w$]*)", re.M)


_IMPORT_STMT = re.compile(r"^\s*import\s+(type\s+)?(.*?)\s+from\s+['\"]([^'\"]+)['\"]\s*;?\s*$")


def _used_drawings(library: dict[str, str], shots: dict[str, str]) -> list[str]:
    used = set()
    for code in shots.values():
        for tag, backdrop in _TAG.findall(code):
            used.add(tag or backdrop)
    return sorted(n for n in library if n in used)


def _local_name(spec: str) -> str:
    return spec.replace("type ", "").split(" as ")[-1].strip()


def _clause_specs(clause: str) -> tuple[list[str], list[str]]:
    """(default / namespace imports, named specifiers) of an import clause."""
    bare, named = clause, ""
    if "{" in clause:
        bare = clause[:clause.index("{")]
        named = clause[clause.index("{") + 1:clause.rindex("}")]
    return ([s.strip() for s in bare.split(",") if s.strip()],
            [s.strip() for s in named.split(",") if s.strip()])


def head_with_library_imports(head: str, library: dict[str, str], shots: dict[str, str]) -> str:
    """The frame's import lines plus every name a pasted drawing imports that
    the frame does not, since a drawing's own import lines cannot stay inside
    its namespace. A name the frame already imports is not imported twice; a
    new name joins the frame's line for the same module, or a new line."""
    lines = head.rstrip("\n").split("\n")
    have: set[str] = set()
    for line in lines:
        m = _IMPORT_STMT.match(line)
        if m:
            bare, named = _clause_specs(m.group(2))
            have.update(_local_name(s.removeprefix("* as ")) for s in bare + named)
    for name in _used_drawings(library, shots):
        for stmt in _IMPORT_LINE.findall(library[name]):
            stmt = " ".join(stmt.split())
            m = _IMPORT_STMT.match(stmt)
            if not m:
                if stmt not in lines:
                    lines.append(stmt)
                continue
            is_type, module = bool(m.group(1)), m.group(3)
            bare, named = _clause_specs(m.group(2))
            for spec in bare:
                if _local_name(spec.removeprefix("* as ")) not in have:
                    have.add(_local_name(spec.removeprefix("* as ")))
                    lines.append(f"import {'type ' if is_type else ''}{spec} from '{module}';")
            prefix, suffix = ("import type {" if is_type else "import {"), f"}} from '{module}';"
            for spec in named:
                if _local_name(spec) in have:
                    continue
                have.add(_local_name(spec))
                at = next((i for i, l in enumerate(lines) if l.startswith(prefix) and l.endswith(suffix)), None)
                if at is None:
                    lines.append(f"{prefix}{spec}{suffix}")
                else:
                    lines[at] = lines[at][:-len(suffix)] + f", {spec}" + suffix
    return "\n".join(lines) + "\n"


def library_block(library: dict[str, str], shots: dict[str, str]) -> str:
    """The library drawings the shots actually use, pasted into the
    script so it renders without any file beside it. Their import lines are
    dropped here; `head_with_library_imports` carries what they import into
    the frame's own import lines.

    Each drawing is written as a self-contained module, free to name its own
    top-level helpers (`LAND`, `Gear`, ...), so it is pasted inside its own
    namespace: two drawings using the same helper name cannot clash. Only
    the names it exports are lifted into the script, as before."""
    blocks = []
    for name in _used_drawings(library, shots):
        code = _IMPORT_LINE.sub("", library[name]).strip("\n")
        scope = f"Library_{name}"
        lifted = [
            f"type {export} = {scope}.{export};" if kind in ("type", "interface")
            else f"const {export} = {scope}.{export};"
            for kind, export in _EXPORT.findall(code)
        ]
        blocks.append(
            f"// Hình thư viện: {name}\nnamespace {scope} {{\n{code}\n}}\n" + "".join(l + "\n" for l in lifted)
        )
    return "\n".join(blocks)


def merge_remotion(
    sb: Storyboard, layout: str, shots: dict[str, str], *, stub_missing: bool = False,
    library: dict[str, str] | None = None, canvas: Frame = LANDSCAPE,
) -> Merged:
    """The whole Remotion script: the fixed frame around `layout` and the shot
    functions, in storyboard order, registered as a `canvas`-sized composition."""
    ordered = [sh.id for _, sh in sb.all_shots()]
    missing = [i for i in ordered if i not in shots]
    if missing and not stub_missing:
        raise ValueError(f"cannot merge: no code for shots {missing}")

    layout = layout.strip()
    palette = remotion_frame_text(sb)["palette"]
    head = head_with_library_imports(_REMOTION_HEAD, library or {}, shots)
    parts: list[str] = [head, palette, "", layout, ""]
    # Derive the layout's line range from the text actually emitted, not from
    # arithmetic on the pieces.
    first = "\n".join(parts[:3]).count("\n") + 2
    lines: dict[str, tuple[int, int]] = {LAYOUT_KEY: (first, first + layout.count("\n"))}
    parts.append("const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'} as const;\n")
    narrations = ",\n".join("  " + json.dumps(sh.narration, ensure_ascii=False) for _, sh in sb.all_shots())
    parts.append(f"export const narrations: string[] = [\n{narrations},\n];\n")
    parts.append("type ShotProps = {duration: number};\n")
    drawings = library_block(library or {}, shots)
    if drawings:
        parts.append(drawings)

    text = "\n".join(parts) + "\n"
    line = text.count("\n") + 1
    for sid in ordered:
        code = shots[sid].strip("\n") if sid in shots else remotion_stub(sid)
        n = code.count("\n") + 1
        lines[sid] = (line, line + n - 1)
        text += code + "\n\n"
        line += n + 1
    names = ", ".join(remotion_fn(s) for s in ordered)
    tail = _REMOTION_TAIL.replace("WIDTH_PX", str(canvas.width)).replace("HEIGHT_PX", str(canvas.height))
    text += f"const SHOTS: React.FC<ShotProps>[] = [{names}];\n" + tail
    return Merged(code=text, lines=lines)


def manim_scene_class(topic: str) -> str:
    return naming.pascal(topic, "Video") + "Scene"


def merge_manim(sb: Storyboard, topic: str, cast: str, shots: dict[str, str]) -> Merged:
    ordered = [sh.id for _, sh in sb.all_shots()]
    missing = [i for i in ordered if i not in shots]
    if missing:
        raise ValueError(f"cannot merge: no code for shots {missing}")
    cls = manim_scene_class(topic)

    def indent(code: str) -> str:
        return "\n".join(("    " + ln) if ln.strip() else "" for ln in code.strip("\n").splitlines())

    out = ["from conceptflow import *", "", "", f"class {cls}(ConceptFlowScene):", "    def construct(self):"]
    out.append("        self.setup_cast()")
    for scene in sb.scenes:
        out.append(f"        self.beat({json.dumps(scene.id)})")
        for shot in scene.shots:
            out.append(f"        self.{manim_fn(shot.id)}()")
    out.append("")
    text = "\n".join(out) + "\n"
    line = text.count("\n") + 1

    cast_block = indent(cast)
    lines: dict[str, tuple[int, int]] = {CAST_KEY: (line, line + cast_block.count("\n"))}
    text += cast_block + "\n\n"
    line += cast_block.count("\n") + 2

    for sid in ordered:
        block = indent(shots[sid])
        n = block.count("\n") + 1
        lines[sid] = (line, line + n - 1)
        text += block + "\n\n"
        line += n + 1
    return Merged(code=text.rstrip("\n") + "\n", lines=lines, scene_class_name=cls)
