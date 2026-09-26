"""Storyboard JSON (CR-039 FR101).

The Visual Director now returns a JSON document instead of prose so that the
code step can split it by shot without re-reading prose. This module parses,
validates and renders it back to the readable `CẢNH n / n.m | MÁY | HÌNH |
THOẠI` form the Creator edits.
"""

from __future__ import annotations

import json
import re

from pydantic import BaseModel, ConfigDict, ValidationError, field_validator

HEX = re.compile(r"^#[0-9A-Fa-f]{6}$")
SHOT_ID = re.compile(r"^\d+\.\d+$")
_FENCE = re.compile(r"```[a-zA-Z0-9]*\r?\n(.*?)\r?\n?```", re.DOTALL)


class StoryboardError(ValueError):
    """The document is unusable; `problems` lists everything wrong with it."""

    def __init__(self, problems: list[str]) -> None:
        super().__init__("; ".join(problems))
        self.problems = problems


class PaletteEntry(BaseModel):
    model_config = ConfigDict(extra="ignore")
    role: str
    hex: str
    meaning: str = ""

    @field_validator("hex")
    @classmethod
    def _hex(cls, v: str) -> str:
        if not HEX.match(v):
            raise ValueError(f"hex must look like #RRGGBB, got {v!r}")
        return v.upper()


class Shot(BaseModel):
    model_config = ConfigDict(extra="ignore")
    id: str
    camera: str = ""
    visual: str
    narration: str

    @field_validator("id")
    @classmethod
    def _id(cls, v: str) -> str:
        if not SHOT_ID.match(v):
            raise ValueError(f"shot id must look like n.m, got {v!r}")
        return v

    @field_validator("narration", "visual")
    @classmethod
    def _non_empty(cls, v: str) -> str:
        if not v.strip():
            raise ValueError("must not be empty")
        return v


class Scene(BaseModel):
    model_config = ConfigDict(extra="ignore")
    id: str
    title: str = ""
    invariant: str = ""
    transition_in: str | None = None
    mood: str = ""
    end_frame: str = ""
    shots: list[Shot]


class Storyboard(BaseModel):
    model_config = ConfigDict(extra="ignore")
    hero: str | None = None
    world: str | None = None
    palette: list[PaletteEntry]
    scenes: list[Scene]

    def all_shots(self) -> list[tuple[Scene, Shot]]:
        return [(sc, sh) for sc in self.scenes for sh in sc.shots]


def _extract_json(text: str) -> str:
    text = text.strip()
    m = _FENCE.search(text)
    if m:
        return m.group(1).strip()
    if text.startswith("{"):
        return text
    # Tolerate a sentence of chatter around a bare object, nothing more.
    start, end = text.find("{"), text.rfind("}")
    if start != -1 and end > start:
        return text[start : end + 1]
    return text


def parse(content: str) -> Storyboard:
    """Parse and validate. Raises StoryboardError listing every problem."""
    try:
        data = json.loads(_extract_json(content))
    except json.JSONDecodeError as exc:
        raise StoryboardError([f"not valid JSON: {exc}"]) from exc
    if not isinstance(data, dict):
        raise StoryboardError(["top level must be a JSON object"])
    try:
        sb = Storyboard.model_validate(data)
    except ValidationError as exc:
        raise StoryboardError(
            [f"{'.'.join(str(p) for p in e['loc'])}: {e['msg']}" for e in exc.errors()]
        ) from exc

    problems: list[str] = []
    if not sb.hero and not sb.world:
        problems.append("one of hero / world is required")
    if not sb.palette:
        problems.append("palette must not be empty")
    roles = [p.role for p in sb.palette]
    if len(set(roles)) != len(roles):
        problems.append("palette roles must be unique")
    if not sb.scenes:
        problems.append("scenes must not be empty")
    seen: set[str] = set()
    for sc in sb.scenes:
        if not sc.shots:
            problems.append(f"scene {sc.id!r} has no shots")
        for sh in sc.shots:
            if sh.id in seen:
                problems.append(f"duplicate shot id {sh.id}")
            seen.add(sh.id)
    if problems:
        raise StoryboardError(problems)
    return sb


def dumps(sb: Storyboard) -> str:
    return json.dumps(sb.model_dump(), ensure_ascii=False, indent=2)


def to_prose(sb: Storyboard) -> str:
    lines: list[str] = []
    if sb.hero:
        lines.append(f"NHÂN VẬT CHÍNH: {sb.hero}")
    else:
        lines.append(f"THẾ GIỚI: {sb.world}")
    lines.append("BẢNG MÀU:")
    for p in sb.palette:
        lines.append(f"  {p.role} — {p.hex} — {p.meaning}")
    for i, sc in enumerate(sb.scenes, 1):
        lines += ["", f"CẢNH {i} — {sc.title or sc.id}"]
        if sc.invariant:
            lines.append(f"Ý nghĩa bất biến: {sc.invariant}")
        if sc.transition_in:
            lines.append(f"Chuyển cảnh vào: {sc.transition_in}")
        if sc.mood:
            lines.append(f"Không khí: {sc.mood}")
        lines.append("Các shot:")
        for sh in sc.shots:
            lines.append(f'  {sh.id} | MÁY: {sh.camera} | HÌNH: {sh.visual} | THOẠI: "{sh.narration}"')
        if sc.end_frame:
            lines.append(f"Kết cảnh: {sc.end_frame}")
    return "\n".join(lines)


def fix_prompt(broken: str, problems: list[str]) -> str:
    """One repair turn for a storyboard that failed validation."""
    listed = "\n".join(f"- {p}" for p in problems)
    return (
        "The storyboard JSON below failed validation. Return the SAME storyboard, corrected, as one JSON "
        "object and nothing else (no markdown fence, no commentary). Do not change, add or drop any "
        "narration, shot or scene beyond what the problems require.\n\n"
        f"Problems:\n{listed}\n\nStoryboard:\n{broken}"
    )
