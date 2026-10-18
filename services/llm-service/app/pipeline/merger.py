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

# Frame sections that are model-written but shared by every shot; they appear in
# `Merged.lines` next to the shot ids so an error inside them can be repaired too.
LAYOUT_KEY = "layout"


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
    "Figure", "HeldAt", "figurePoint", "useBlink", "phaseOf", "shadeOf", "useSvgId",
    "INK", "SHADE", "BLUSH", "WHITE", "FIGURE_DETAIL",
)
# Every backdrop of conceptflow-mini/backdrops.tsx: whole-frame places a Scene
# draws in its depth layers. Held to the file by the same test.
BACKDROP_KIT = (
    "MeadowBackdrop", "RoomBackdrop", "StreetBackdrop", "InsideBodyBackdrop", "UnderwaterBackdrop",
    "SpaceBackdrop",
)
# Hooks of the scene kit a library backdrop drawing uses.
SCENE_HELPERS = ("useCamera", "useLayerBox")
# The remotion API every shot may use. The rule book tells the model to draw
# randomness with remotion's seeded `random`, so it is imported here too.
REMOTION_API = (
    "registerRoot", "Composition", "AbsoluteFill", "interpolate", "interpolateColors", "spring", "Easing",
    "useCurrentFrame", "useVideoConfig", "random",
)
# conceptflow-mini/primitives names the rule book offers the shots.
PRIMITIVES = ("Stage", "SAFE_MARGIN", "WIDTH", "HEIGHT", "BACKGROUND", "useFrameBox")
# conceptflow-mini/segments: the composition's timing, plus the helpers a shot
# uses to time a change to one of its narration lines.
SEGMENT_API = ("calculateMetadataFromSegments", "Segments", "evenLines", "lineSpan")
# Every component of conceptflow-mini/rig.tsx: close-up body parts that act.
# Held to the file by the same test as the other kits.
RIG_KIT = ("ReachingHand",)
# conceptflow-mini/film.tsx: the film look every shot is wrapped in.
FILM_KIT = ("Shot",)
FILM_HELPERS = ("LookProvider", "useLook", "parseHex", "darken", "mix", "withAlpha")
# conceptflow-mini/material.tsx: forms lit by the scene's light.
MATERIAL_KIT = ("Shape", "GroundShadow")
MATERIAL_HELPERS = ("FigureFrameProvider", "lightDirection", "capsulePath")
# conceptflow-mini/procedural.tsx: noise-driven scenery generators.
PROCEDURAL_KIT = (
    "Ridges", "Hills", "Dunes", "Canopy", "Grass", "Clouds", "Stars", "Rocks", "Coral", "Waves", "Cells",
    "Particles",
)
PROCEDURAL_HELPERS = ("noise1", "ridgeLine", "voronoiCells")
# conceptflow-mini/shader.tsx: WebGL shader layers.
SHADER_KIT = ("ShaderLayer",)
SHADER_HELPERS = ("shaderSource", "SHADER_NAMES")
# conceptflow-mini/skeleton.tsx: bones, poses and machines.
SKELETON_KIT = ("Rig", "Bone", "Eyes", "Brows", "Mouth", "Mechanism", "Part")
SKELETON_HELPERS = (
    "SKELETONS", "boneDepth", "HUMANOID_POSES", "POSES", "restPoseName", "cycleAngles", "ikTwoBone",
    "rotateAbout", "elbowPosition", "wrap180", "easeAction", "poseAt", "solveRig", "lookAngle", "isBlinking",
    "partTransform",
)
# conceptflow-mini/kits.ts
KITS_HELPERS = ("KITS", "KIT_VERSIONS", "kitKeys", "useKit")


def _named_import(names: tuple[str, ...], module: str) -> str:
    return "import {" + ", ".join(names) + "} from '" + module + "';\n"


_REMOTION_HEAD = (
    "import React from 'react';\n"
    + _named_import(REMOTION_API, "remotion")
    + _named_import(SEGMENT_API, "./conceptflow-mini/segments")
    + _named_import(PRIMITIVES, "./conceptflow-mini/primitives")
    + "import {LottieClip} from './conceptflow-mini/lottie';\n"
    + _named_import(ILLUSTRATION_KIT + ILLUSTRATION_HELPERS, "./conceptflow-mini/illustration")
    + "import type {FigureProps, Mood, PersonPose} from './conceptflow-mini/illustration';\n"
    + _named_import(SCENE_KIT + SCENE_HELPERS, "./conceptflow-mini/scene")
    + _named_import(BACKDROP_KIT, "./conceptflow-mini/backdrops")
    + _named_import(RIG_KIT, "./conceptflow-mini/rig")
    + _named_import(FILM_KIT + FILM_HELPERS, "./conceptflow-mini/film")
    + _named_import(MATERIAL_KIT + MATERIAL_HELPERS, "./conceptflow-mini/material")
    + _named_import(PROCEDURAL_KIT + PROCEDURAL_HELPERS, "./conceptflow-mini/procedural")
    + _named_import(SHADER_KIT + SHADER_HELPERS, "./conceptflow-mini/shader")
    + _named_import(SKELETON_KIT + SKELETON_HELPERS, "./conceptflow-mini/skeleton")
    + "import type {Look, MaterialName} from './conceptflow-mini/kits';\n"
)


def available_names_text() -> str:
    """The names the frame imports for the shots to use, one module per
    item, for the chunk prompt's "already in the file" list. Built from the
    same tuples as the frame's import lines, so the two cannot disagree."""
    return "; ".join([
        "react",
        "remotion: " + ", ".join(REMOTION_API),
        "./conceptflow-mini/segments: " + ", ".join(SEGMENT_API),
        "./conceptflow-mini/primitives: " + ", ".join(PRIMITIVES),
        "./conceptflow-mini/lottie: LottieClip",
        "./conceptflow-mini/illustration: " + ", ".join(ILLUSTRATION_KIT),
        "./conceptflow-mini/scene: " + ", ".join(SCENE_KIT),
        "./conceptflow-mini/backdrops: " + ", ".join(BACKDROP_KIT),
        "./conceptflow-mini/rig: " + ", ".join(RIG_KIT),
        "./conceptflow-mini/film: " + ", ".join(FILM_KIT),
        "./conceptflow-mini/material: " + ", ".join(MATERIAL_KIT),
        "./conceptflow-mini/procedural: " + ", ".join(PROCEDURAL_KIT),
        "./conceptflow-mini/shader: " + ", ".join(SHADER_KIT),
        "./conceptflow-mini/skeleton: " + ", ".join(SKELETON_KIT),
    ])


def frame_names() -> frozenset[str]:
    """Every name the Remotion frame puts in scope for the shot functions:
    the imports of the head and the constants the frame declares. Built from
    the same tuples as the head, so a name the head imports is always here."""
    return frozenset((
        "React", *REMOTION_API, *SEGMENT_API, *PRIMITIVES, "LottieClip",
        *ILLUSTRATION_KIT, *ILLUSTRATION_HELPERS, "FigureProps", "Mood", "PersonPose",
        *SCENE_KIT, *SCENE_HELPERS, *BACKDROP_KIT, *RIG_KIT,
        *FILM_KIT, *FILM_HELPERS, *MATERIAL_KIT, *MATERIAL_HELPERS, *PROCEDURAL_KIT, *PROCEDURAL_HELPERS,
        *SHADER_KIT, *SHADER_HELPERS, *SKELETON_KIT, *SKELETON_HELPERS, "Look", "MaterialName",
        "PALETTE", "LAYOUT", "LOOKS", "clamp", "ShotProps",
    ))

_REMOTION_TAIL = """
function CreatorComposition({segments = []}: {segments?: {startFrame: number; durationInFrames: number; lines?: number[]}[]}) {
  return (
    <Stage>
      <Segments segments={segments}>
        {(index, segment) => {
          const Shot = SHOTS[index];
          const lines = segment.lines ?? evenLines(segment.durationInFrames, shotLineCounts[index] ?? 1);
          return Shot ? <Shot duration={segment.durationInFrames} lines={lines} /> : null;
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
    return {"palette": palette, "looks": looks_text(sb), "head": _REMOTION_HEAD.strip()}


def looks_text(sb: Storyboard) -> str:
    """`const LOOKS = {...};`: the light of every scene as a `Look` for `<Shot look=...>`.

    A key is `scene<n>`, `n` being the first part of the ids of the scene's shots
    (shot 2.3 belongs to `LOOKS.scene2`). Every colour is a PALETTE entry, so a
    scene's colours are the video's palette and nothing else."""
    keys = palette_keys(sb)
    rows: list[str] = []
    for sc in sb.scenes:
        li = sc.light
        g = li.grade
        n = _layout_number
        body = (
            f"light: {{x: {n(li.x)}, y: {n(li.y)}, color: PALETTE.{keys[li.color]}, intensity: {n(li.intensity)}}}, "
            f"ambient: PALETTE.{keys[li.ambient]}, "
            f"atmosphere: {{color: PALETTE.{keys[li.atmosphere]}, density: {n(li.haze)}}}, "
            f"focus: '{li.focus}', "
            f"grade: {{warmth: {n(g.warmth)}, contrast: {n(g.contrast)}, saturation: {n(g.saturation)}}}"
        )
        for prefix in dict.fromkeys(sh.id.split(".")[0] for sh in sc.shots):
            note = " ".join(f"{sc.title or sc.id}: {li.source}".split()).rstrip(": ")
            rows.append(f"  scene{prefix}: {{{body}}}, // {note}")
    return "const LOOKS: Record<string, Look> = {\n" + "\n".join(rows) + "\n};"


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
    return f"function {remotion_fn(shot_id)}({{duration, lines}}: ShotProps) {{\n  return null;\n}}"


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
    parts: list[str] = [head, palette, "", layout, "", looks_text(sb), ""]
    # Derive the layout's line range from the text actually emitted, not from
    # arithmetic on the pieces.
    first = "\n".join(parts[:3]).count("\n") + 2
    lines: dict[str, tuple[int, int]] = {LAYOUT_KEY: (first, first + layout.count("\n"))}
    parts.append("const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'} as const;\n")
    # One narration entry per line: each is synthesized as its own clip, and
    # shotLineCounts tells rendering which lines make up which shot.
    spoken = [said for _, sh in sb.all_shots() for said in sh.spoken_lines()]
    narrations = ",\n".join("  " + json.dumps(said, ensure_ascii=False) for said in spoken)
    parts.append(f"export const narrations: string[] = [\n{narrations},\n];\n")
    counts = ", ".join(str(len(sh.spoken_lines())) for _, sh in sb.all_shots())
    parts.append(f"export const shotLineCounts: number[] = [{counts}];\n")
    parts.append("type ShotProps = {duration: number; lines: number[]};\n")
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
