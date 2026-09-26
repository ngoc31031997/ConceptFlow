"""Client for the Rendering service's compile check (CR-039 FR104)."""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from typing import Protocol

import httpx


@dataclass
class Diagnostic:
    message: str
    line: int | None = None


@dataclass
class CheckResult:
    ok: bool
    diagnostics: list[Diagnostic] = field(default_factory=list)
    # Raw tool output, kept because a Manim traceback names the failing shot
    # function on lines that carry no usable line number of their own.
    raw: str = ""


class CheckerPort(Protocol):
    async def check(self, engine: str, code: str, scene_class_name: str) -> CheckResult: ...


class CheckerUnavailable(RuntimeError):
    """The Rendering service could not be reached or answered nonsense. Never
    treated as PASS: an unchecked script must not look checked."""


class RenderingChecker:
    def __init__(self, base_url: str, timeout: float, client: httpx.AsyncClient | None = None) -> None:
        self._base = base_url.rstrip("/")
        self._client = client or httpx.AsyncClient(timeout=timeout)

    async def check(self, engine: str, code: str, scene_class_name: str) -> CheckResult:
        if engine not in ("remotion", "manim"):
            raise ValueError(f"unknown engine {engine!r}")
        try:
            resp = await self._client.post(
                f"{self._base}/v1/check/{engine}",
                json={"code": code, "scene_class_name": scene_class_name},
            )
        except httpx.HTTPError as exc:
            raise CheckerUnavailable(f"rendering check call failed: {exc}") from exc
        if resp.status_code != 200:
            raise CheckerUnavailable(f"rendering check returned {resp.status_code}: {resp.text[:500]}")
        try:
            data = resp.json()
            return CheckResult(
                ok=bool(data["ok"]),
                diagnostics=[Diagnostic(d["message"], d.get("line")) for d in data.get("diagnostics", [])],
                raw=data.get("raw", ""),
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
