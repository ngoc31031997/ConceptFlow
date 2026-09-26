"""`tsc --noEmit` over a Creator/LLM-authored Remotion script (CR-039 FR104).

The script is written next to the real conceptflow-mini sources so its relative
imports resolve exactly as they will at render time, checked under the
project's own tsconfig, and removed afterwards. Each call gets its own
throwaway tsconfig that includes only that file, so concurrent checks never see
each other's scripts.
"""

from __future__ import annotations

import os
import re
import subprocess
import uuid
from dataclasses import dataclass
from pathlib import Path

from adapters.rendering.remotion_renderer import _sanitize_script

# src/_check_ab12.tsx(12,5): error TS2304: Cannot find name 'x'.
_DIAG_RE = re.compile(r"^(?P<file>[^\s(][^(]*)\((?P<line>\d+),(?P<col>\d+)\):\s+error\s+(?P<code>TS\d+):\s+(?P<msg>.*)$")


class TypeScriptCheckError(RuntimeError):
    """tsc itself could not be run or gave output nobody can read. Never PASS."""


@dataclass(frozen=True)
class TsDiagnostic:
    line: int
    column: int
    code: str
    message: str


class TypeScriptChecker:
    def __init__(self, project_dir: str | Path, timeout_seconds: int = 90) -> None:
        self._dir = Path(project_dir)
        self._timeout = timeout_seconds

    def _tsc(self) -> str:
        local = self._dir / "node_modules" / ".bin" / "tsc"
        if not local.exists():
            raise TypeScriptCheckError(f"tsc not found at {local} — is remotion_project installed?")
        return str(local)

    def check(self, script: str) -> list[TsDiagnostic]:
        name = f"_check_{uuid.uuid4().hex[:12]}"
        entry = self._dir / "src" / f"{name}.tsx"
        config = self._dir / f"tsconfig.{name}.json"
        try:
            entry.write_text(_sanitize_script(script), encoding="utf-8")
            config.write_text(
                '{"extends": "./tsconfig.json", "include": ["src/%s.tsx"]}' % name, encoding="utf-8"
            )
            try:
                proc = subprocess.run(
                    [self._tsc(), "--noEmit", "--pretty", "false", "-p", str(config)],
                    cwd=self._dir, capture_output=True, text=True, timeout=self._timeout,
                    env={**os.environ, "NO_COLOR": "1"},
                )
            except subprocess.TimeoutExpired as exc:
                raise TypeScriptCheckError(f"tsc timed out after {self._timeout}s") from exc
            return self._parse(proc.returncode, proc.stdout, proc.stderr, name)
        finally:
            for path in (entry, config):
                path.unlink(missing_ok=True)

    @staticmethod
    def _parse(returncode: int, stdout: str, stderr: str, name: str) -> list[TsDiagnostic]:
        if returncode == 0:
            return []
        diags: list[TsDiagnostic] = []
        for raw in stdout.splitlines():
            m = _DIAG_RE.match(raw.strip())
            if m and name in m.group("file"):
                diags.append(TsDiagnostic(int(m.group("line")), int(m.group("col")), m.group("code"), m.group("msg")))
        if not diags:
            # Non-zero with nothing we can attribute to the script: report, do not pass.
            raise TypeScriptCheckError(f"tsc exited {returncode} with no diagnostics for the script: "
                                       f"{(stdout + stderr).strip()[:1500]}")
        return diags
