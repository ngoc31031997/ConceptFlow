"""CheckScriptUseCase — the compile check the code-authoring pipeline runs before
anything is saved (CR-039 FR104).

Remotion: Lottie-id lint and the narration dry pass (the same gate the saga
uses), plus a real `tsc --noEmit`. Manim: the saga's own gate (lint + a dry
run of the script). Both report line-numbered diagnostics when they can.
"""

from __future__ import annotations

from dataclasses import dataclass, field

from adapters.rendering.typescript_checker import TypeScriptChecker
from application.validate_script import ScriptValidationError, ValidateScriptUseCase
from domain.errors import AnimationEngineError
from domain.models import ScriptRenderRequest


@dataclass
class CheckDiagnostic:
    message: str
    line: int | None = None


@dataclass
class CheckOutcome:
    ok: bool
    diagnostics: list[CheckDiagnostic] = field(default_factory=list)
    raw: str = ""


class CheckScriptUseCase:
    def __init__(self, validate: ValidateScriptUseCase, typescript: TypeScriptChecker) -> None:
        self._validate = validate
        self._ts = typescript

    def _request(self, engine: str, code: str, scene_class_name: str) -> ScriptRenderRequest:
        return ScriptRenderRequest(
            project_id="check", script_content=code, scene_class_name=scene_class_name,
            narration_segments=[], engine=engine,
        )

    def check(self, engine: str, code: str, scene_class_name: str) -> CheckOutcome:
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

        if engine == "remotion":
            diags += [CheckDiagnostic(f"{d.code}: {d.message}", d.line) for d in self._ts.check(code)]
        return CheckOutcome(ok=not diags, diagnostics=diags, raw=raw)
