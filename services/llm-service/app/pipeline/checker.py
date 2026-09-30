"""Client for the Rendering service's compile check (CR-039 FR104)."""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from typing import Protocol

import httpx

COMPILE = "compile"
LAYOUT = "layout"


@dataclass
class Diagnostic:
    message: str
    line: int | None = None
    # compile: lint / tsc / Manim dry run. layout: measured on the drawn shot
    # (CR-048 T6b) — same repair loop, but the repair prompt says what it is.
    kind: str = COMPILE
    # CR-050 FR-22: the rule that failed (a layout rule, a tsc code), "" when
    # the checker has none — for statistics, not for the repair prompt.
    rule: str = ""


@dataclass
class CheckResult:
    ok: bool
    diagnostics: list[Diagnostic] = field(default_factory=list)
    # Raw tool output, kept because a Manim traceback names the failing shot
    # function on lines that carry no usable line number of their own.
    raw: str = ""
    # Findings that do not fail the check (a hero drawn too small) and notices
    # that part of it could not run (the layout check without a browser).
    warnings: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class LayoutContext:
    """What the Rendering service's layout check needs beyond the code (CR-048 T6b)."""

    # {"edge": "top" | "bottom", "px": int}: the strip burned-in subtitles
    # cover. None = nothing is burned into the frame.
    subtitle_band: dict | None = None
    video_font: str = ""  # "" = the Stage's default

    def to_json(self) -> dict:
        out: dict = {}
        if self.subtitle_band:
            out["subtitle_band"] = {"edge": self.subtitle_band["edge"], "px": int(self.subtitle_band["px"])}
        if self.video_font:
            out["video_font"] = self.video_font
        return out


class CheckerPort(Protocol):
    async def check(
        self, engine: str, code: str, scene_class_name: str, layout: LayoutContext | None = None,
    ) -> CheckResult: ...


class CheckerUnavailable(RuntimeError):
    """The Rendering service could not be reached or answered nonsense. Never
    treated as PASS: an unchecked script must not look checked."""


class RenderingChecker:
    def __init__(self, base_url: str, timeout: float, client: httpx.AsyncClient | None = None) -> None:
        self._base = base_url.rstrip("/")
        self._client = client or httpx.AsyncClient(timeout=timeout)

    async def check(
        self, engine: str, code: str, scene_class_name: str, layout: LayoutContext | None = None,
    ) -> CheckResult:
        if engine not in ("remotion", "manim"):
            raise ValueError(f"unknown engine {engine!r}")
        body: dict = {"code": code, "scene_class_name": scene_class_name}
        if layout is not None:
            body.update(layout.to_json())
        try:
            resp = await self._client.post(f"{self._base}/v1/check/{engine}", json=body)
        except httpx.HTTPError as exc:
            raise CheckerUnavailable(f"rendering check call failed: {exc}") from exc
        if resp.status_code != 200:
            raise CheckerUnavailable(f"rendering check returned {resp.status_code}: {resp.text[:500]}")
        try:
            data = resp.json()
            return CheckResult(
                ok=bool(data["ok"]),
                diagnostics=[
                    Diagnostic(d["message"], d.get("line"), d.get("kind") or COMPILE, d.get("rule") or "")
                    for d in data.get("diagnostics", [])
                ],
                raw=data.get("raw", ""),
                warnings=[str(w) for w in data.get("warnings", [])],
            )
        except (ValueError, KeyError, TypeError) as exc:
            raise CheckerUnavailable(f"rendering check returned an unreadable body: {exc}") from exc


_MANIM_SHOT_IN_TRACE = re.compile(r"\bin shot_(\d+)_(\d+)\b")
_LINE_IN_TRACE = re.compile(r'line (\d+), in (\w+)')


def shots_from_manim_trace(raw: str) -> list[tuple[str, int | None]]:
    """Every (shot id, line) a traceback passes through, innermost last."""
    out = []
    for m in _LINE_IN_TRACE.finditer(raw):
        fm = re.fullmatch(r"shot_(\d+)_(\d+)", m.group(2))
        if fm:
            out.append((f"{int(fm.group(1))}.{int(fm.group(2))}", int(m.group(1))))
    return out
