"""Code Merger (CR-039 FR103) — deterministic, no LLM.

Everything that must agree with the storyboard by construction is generated
here: the narrations array, the shot list and its order, the composition
registration / the Scene class. The model only writes the shot bodies, so the
"narrations.length != SHOTS.length" class of failure cannot occur.
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass

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


_REMOTION_HEAD = """import React from 'react';
import {registerRoot, Composition, AbsoluteFill, interpolate, interpolateColors, spring, Easing, useCurrentFrame, useVideoConfig} from 'remotion';
import {calculateMetadataFromSegments, Segments} from './conceptflow-mini/segments';
import {Stage, SAFE_MARGIN, WIDTH, HEIGHT} from './conceptflow-mini/primitives';
import {LottieClip} from './conceptflow-mini/lottie';
"""

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
    width={1920}
    height={1080}
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


def merge_remotion(sb: Storyboard, layout: str, shots: dict[str, str]) -> Merged:
    ordered = [sh.id for _, sh in sb.all_shots()]
    missing = [i for i in ordered if i not in shots]
    if missing:
        raise ValueError(f"cannot merge: no code for shots {missing}")

    layout = layout.strip()
    palette = remotion_frame_text(sb)["palette"]
    parts: list[str] = [_REMOTION_HEAD, palette, "", layout, ""]
    # Derive the layout's line range from the text actually emitted, not from
    # arithmetic on the pieces.
    first = "\n".join(parts[:3]).count("\n") + 2
    lines: dict[str, tuple[int, int]] = {LAYOUT_KEY: (first, first + layout.count("\n"))}
    parts.append("const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'} as const;\n")
    narrations = ",\n".join("  " + json.dumps(sh.narration, ensure_ascii=False) for _, sh in sb.all_shots())
    parts.append(f"export const narrations: string[] = [\n{narrations},\n];\n")
    parts.append("type ShotProps = {duration: number};\n")

    text = "\n".join(parts) + "\n"
    line = text.count("\n") + 1
    for sid in ordered:
        code = shots[sid].strip("\n")
        n = code.count("\n") + 1
        lines[sid] = (line, line + n - 1)
        text += code + "\n\n"
        line += n + 1
    names = ", ".join(remotion_fn(s) for s in ordered)
    text += f"const SHOTS: React.FC<ShotProps>[] = [{names}];\n" + _REMOTION_TAIL
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
