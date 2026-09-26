"""TypeScript check of a Creator/LLM-authored Remotion script (CR-039 FR104).

A plain `tsc --noEmit` spends about a second re-reading React's, Remotion's and
the lib's type declarations before it looks at the script, and the code
pipeline checks every chunk and every repair round. So the check runs in one
long-lived Node process (`remotion_project/tscheck.mjs`) that parses those
declarations once and keeps them; each check then costs only the script.

The script is checked as `src/__tscheck__.tsx` under the project's own
tsconfig, so its relative imports resolve exactly as they will at render time.
It is never written to disk, so concurrent checks cannot see each other.
"""

from __future__ import annotations

import json
import os
import queue
import shutil
import subprocess
import threading
import time
import uuid
from collections import deque
from dataclasses import dataclass
from pathlib import Path
from typing import IO

from adapters.rendering.remotion_renderer import _sanitize_script


class TypeScriptCheckError(RuntimeError):
    """The checker itself could not run or gave an answer nobody can read. Never PASS."""


@dataclass(frozen=True)
class TsDiagnostic:
    line: int
    column: int
    code: str
    message: str


def _pump(stream: IO[str], sink) -> None:
    for line in stream:
        sink(line)
    sink(None)


class TypeScriptChecker:
    def __init__(self, project_dir: str | Path, timeout_seconds: int = 90, node: str = "node") -> None:
        self._dir = Path(project_dir)
        self._timeout = timeout_seconds
        self._node = node
        # One process answers one request at a time; checks arrive from worker threads.
        self._lock = threading.Lock()
        self._proc: subprocess.Popen[str] | None = None
        self._replies: queue.Queue[str | None] = queue.Queue()
        self._stderr: deque[str] = deque(maxlen=40)

    # -- process -------------------------------------------------------------

    def _start(self) -> None:
        if not (self._dir / "node_modules" / "typescript").exists():
            raise TypeScriptCheckError(
                f"typescript not found under {self._dir / 'node_modules'} — is remotion_project installed?")
        script = self._dir / "tscheck.mjs"
        if not script.exists():
            raise TypeScriptCheckError(f"checker script missing: {script}")
        node = shutil.which(self._node)
        if node is None:
            raise TypeScriptCheckError(f"{self._node!r} not found on PATH")
        try:
            proc = subprocess.Popen(
                [node, str(script), str(self._dir)], cwd=self._dir,
                stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                text=True, encoding="utf-8", bufsize=1, env={**os.environ, "NO_COLOR": "1"},
            )
        except OSError as exc:
            raise TypeScriptCheckError(f"could not start the TypeScript checker: {exc}") from exc
        self._replies = queue.Queue()
        self._stderr.clear()
        threading.Thread(target=_pump, args=(proc.stdout, self._replies.put), daemon=True).start()
        threading.Thread(
            target=_pump, args=(proc.stderr, lambda s: s is not None and self._stderr.append(s)), daemon=True,
        ).start()
        self._proc = proc

    def _stop(self) -> None:
        proc, self._proc = self._proc, None
        if proc is None:
            return
        try:
            proc.kill()
            proc.wait(timeout=5)
        except (OSError, subprocess.TimeoutExpired):
            pass

    def close(self) -> None:
        with self._lock:
            self._stop()

    def warm(self) -> None:
        """Start the process and load the type declarations, so the first real
        check does not pay for it. Failures surface on that first check."""
        try:
            self.check("export {};\n")
        except TypeScriptCheckError:
            pass

    # -- check ---------------------------------------------------------------

    def check(self, script: str) -> list[TsDiagnostic]:
        with self._lock:
            if self._proc is None or self._proc.poll() is not None:
                self._start()
            assert self._proc is not None and self._proc.stdin is not None
            rid = uuid.uuid4().hex
            try:
                self._proc.stdin.write(json.dumps({"id": rid, "code": _sanitize_script(script)}) + "\n")
                self._proc.stdin.flush()
            except OSError as exc:
                self._stop()
                raise TypeScriptCheckError(f"the TypeScript checker is not accepting input: {exc}") from exc

            deadline = time.monotonic() + self._timeout
            while True:
                try:
                    line = self._replies.get(timeout=max(0.0, deadline - time.monotonic()))
                except queue.Empty:
                    self._stop()
                    raise TypeScriptCheckError(f"tsc timed out after {self._timeout}s") from None
                if line is None:
                    tail = "".join(self._stderr).strip()[-1500:]
                    self._stop()
                    raise TypeScriptCheckError(f"the TypeScript checker exited: {tail or 'no output'}")
                try:
                    reply = json.loads(line)
                except ValueError as exc:
                    self._stop()
                    raise TypeScriptCheckError(
                        f"the TypeScript checker wrote an unreadable line: {line[:300]!r}") from exc
                if reply.get("id") == rid:
                    return self._read(reply)

    @staticmethod
    def _read(reply: dict) -> list[TsDiagnostic]:
        if reply.get("error"):
            raise TypeScriptCheckError(f"the TypeScript checker failed: {str(reply['error'])[:1500]}")
        try:
            diags = [
                TsDiagnostic(int(d["line"]), int(d["col"]), str(d["code"]), str(d["message"]))
                for d in reply.get("diagnostics", [])
            ]
        except (KeyError, TypeError, ValueError) as exc:
            raise TypeScriptCheckError(
                f"the TypeScript checker answered an unreadable diagnostic: {exc}") from exc
        other = [str(o) for o in reply.get("other", [])]
        if not diags and other:
            # tsc would fail, but on nothing we can point at in the script: report, do not pass.
            raise TypeScriptCheckError("tsc reported errors outside the script: " + "; ".join(other)[:1500])
        return diags
