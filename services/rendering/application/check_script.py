"""CheckScriptUseCase — the compile check the code-authoring pipeline runs before
anything is saved (CR-039 FR104).

Remotion: Lottie-id lint and the narration dry pass (the same gate the saga
uses), plus a real `tsc --noEmit`, then — once all of that passes — the layout
check (CR-048 T6b): every shot is drawn in headless Chromium at a dozen moments
and the measured boxes are held to the layout rules (domain/layout_rules.py).
Manim: the saga's own gate (lint + a dry run of the script). Both report
line-numbered diagnostics when they can.

Layout violations that block (safe area, subtitle zone, text over text, text
overflow, tiny font, a shot that throws) are diagnostics like a tsc error, so
the repair loop gets them; the rest are warnings. A layout check that cannot
run (no browser, crash, timeout, font missing) is a warning that says so —
never a silent pass — and the tsc result stands.
"""

from __future__ import annotations

import logging
import time
from dataclasses import dataclass, field

from adapters.rendering.layout_checker import LayoutChecker, LayoutCheckError
from adapters.rendering.typescript_checker import TypeScriptChecker
from application.validate_script import ScriptValidationError, ValidateScriptUseCase
from domain.errors import AnimationEngineError
from domain.layout_rules import SubtitleBand, evaluate
from domain.models import ScriptRenderRequest

logger = logging.getLogger(__name__)

COMPILE = "compile"
LAYOUT = "layout"

LAYOUT_DISABLED_WARNING = "Bố cục: chưa kiểm tra — kiểm tra bố cục đang tắt (LAYOUT_CHECK_ENABLED=false)."


@dataclass
class CheckDiagnostic:
    message: str
    line: int | None = None
    kind: str = COMPILE  # compile | layout
    # CR-050 FR-22: which rule failed, for statistics — the layout rule
    # (safe_area, subtitle_zone, ...) or the tsc code (TS2322). "" when the
    # source has no code of its own (lint, the Manim dry run).
    rule: str = ""


@dataclass
class CheckOutcome:
    ok: bool
    diagnostics: list[CheckDiagnostic] = field(default_factory=list)
    raw: str = ""
    # Non-blocking findings and "could not check" notices; never make ok false.
    warnings: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class LayoutContext:
    """What the layout rules need to know about the video beyond its code."""

    subtitle_band: SubtitleBand | None = None  # None = nothing burned into the frame
    video_font: str = ""  # "" = the Stage's default (Be Vietnam Pro)


class CheckScriptUseCase:
    def __init__(
        self, validate: ValidateScriptUseCase, typescript: TypeScriptChecker,
        layout: LayoutChecker | None = None,
    ) -> None:
        """`layout` None = the layout check is turned off (LAYOUT_CHECK_ENABLED=false)."""
        self._validate = validate
        self._ts = typescript
        self._layout = layout

    def _request(self, engine: str, code: str, scene_class_name: str) -> ScriptRenderRequest:
        return ScriptRenderRequest(
            project_id="check", script_content=code, scene_class_name=scene_class_name,
            narration_segments=[], engine=engine,
        )

    def check(
        self, engine: str, code: str, scene_class_name: str, layout: LayoutContext | None = None,
    ) -> CheckOutcome:
        if engine not in ("remotion", "manim"):
            raise ValueError(f"unknown engine {engine!r}")
        if not code.strip():
            raise ValueError("code is empty")
        diags: list[CheckDiagnostic] = []
        raw = ""
        try:
            self._validate.validate(self._request(engine, code, scene_class_name))
        except ScriptValidationError as exc:
            if exc.issues:
                diags += [CheckDiagnostic(i.message, i.line) for i in exc.issues]
            else:
                diags.append(CheckDiagnostic(str(exc)))
            raw = str(exc)
        except AnimationEngineError as exc:
            # A Manim traceback names the failing shot function even though it
            # carries no structured line; the caller reads it from `raw`.
            diags.append(CheckDiagnostic(str(exc)[:2000]))
            raw = str(exc)

        warnings: list[str] = []
        if engine == "remotion":
            diags += [CheckDiagnostic(f"{d.code}: {d.message}", d.line, rule=str(d.code)) for d in self._ts.check(code)]
            if not diags:
                # Only a script that compiles is worth drawing.
                layout_diags, warnings = self._check_layout(code, layout or LayoutContext())
                diags += layout_diags
        return CheckOutcome(ok=not diags, diagnostics=diags, raw=raw, warnings=warnings)

    def _check_layout(self, code: str, ctx: LayoutContext) -> tuple[list[CheckDiagnostic], list[str]]:
        if self._layout is None:
            return [], [LAYOUT_DISABLED_WARNING]
        began = time.monotonic()
        try:
            probe = self._layout.measure(code, ctx.video_font)
        except LayoutCheckError as exc:
            logger.warning("layout check could not run: %s", exc)
            return [], [f"Bố cục: KHÔNG kiểm tra được — {exc}. Kết quả biên dịch vẫn giữ nguyên."]
        font = probe.get("font") or {}
        if font.get("available") is False:
            # Every text width would be measured in a fallback font: 3-10 % off
            # (T6a), enough to invent or hide an overflow. Not a pass either.
            logger.warning(
                "layout check skipped: font %r is not installed in the probe browser", font.get("family"))
            return [], [
                f"Bố cục: KHÔNG kiểm tra được — font '{font.get('family')}' không có trong trình duyệt đo, "
                "số đo chữ sẽ sai."
            ]
        try:
            findings = evaluate(probe, ctx.subtitle_band)
        except (KeyError, TypeError, ValueError) as exc:
            logger.error("layout probe answered measurements the rules cannot read: %r", exc)
            return [], [f"Bố cục: KHÔNG kiểm tra được — số đo không đọc được ({exc!r})."]
        diags = [CheckDiagnostic(f.message, f.line, LAYOUT, f.rule) for f in findings if f.blocking]
        warnings = [
            f"Bố cục: {f.message}" + (f" (dòng {f.line})" if f.line else "")
            for f in findings if not f.blocking
        ]
        timing = probe.get("timing_ms") or {}
        logger.info(
            "layout check shots=%d samples=%s measure=%sms total=%.2fs blocking=%d warnings=%d",
            len(probe.get("shots", [])), timing.get("samples"), timing.get("measure"),
            time.monotonic() - began, len(diags), len(warnings),
        )
        return diags, warnings
