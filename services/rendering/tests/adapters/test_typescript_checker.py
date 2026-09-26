"""The warm TypeScript checker (remotion_project/tscheck.mjs + its Python client).

Process behaviour is exercised against a stand-in checker run by this Python
interpreter, so these tests need no Node. The real checker is exercised when a
remotion_project with node_modules is available (the Rendering image, or
TSCHECK_PROJECT_DIR pointing at an installed copy).
"""

from __future__ import annotations

import os
import shutil
import sys
import textwrap
import time
from pathlib import Path

import pytest

from adapters.rendering.typescript_checker import TsDiagnostic, TypeScriptChecker, TypeScriptCheckError

# --- reply parsing ------------------------------------------------------------


def test_reads_diagnostics_of_the_script():
    diags = TypeScriptChecker._read({"id": "a", "diagnostics": [
        {"line": 12, "col": 5, "code": "TS2304", "message": "Cannot find name 'x'."},
    ], "other": []})
    assert diags == [TsDiagnostic(12, 5, "TS2304", "Cannot find name 'x'.")]


def test_no_errors_is_a_pass():
    assert TypeScriptChecker._read({"id": "a", "diagnostics": [], "other": []}) == []


def test_errors_outside_the_script_are_never_a_pass():
    with pytest.raises(TypeScriptCheckError, match="outside the script"):
        TypeScriptChecker._read({"id": "a", "diagnostics": [], "other": ["TS5023: Unknown compiler option"]})


def test_script_errors_are_reported_even_when_others_exist():
    diags = TypeScriptChecker._read({"id": "a", "diagnostics": [
        {"line": 1, "col": 1, "code": "TS1005", "message": "';' expected."}], "other": ["elsewhere"]})
    assert [d.code for d in diags] == ["TS1005"]


def test_a_checker_error_is_an_error():
    with pytest.raises(TypeScriptCheckError, match="checker failed"):
        TypeScriptChecker._read({"id": "a", "error": "boom"})


# --- process handling (stand-in checker) -----------------------------------------

STAND_IN = {
    # Answers every request: an error on line 1 when the code mentions BAD.
    "echo": """
        import json, sys
        for line in sys.stdin:
            req = json.loads(line)
            bad = "BAD" in req["code"]
            diags = [{"line": 1, "col": 1, "code": "TS2304", "message": "bad"}] if bad else []
            print(json.dumps({"id": req["id"], "diagnostics": diags, "other": []}), flush=True)
    """,
    "crash": """
        import sys
        sys.stdin.readline()
        sys.stderr.write("TypeError: kaput\\n")
        sys.exit(1)
    """,
    "hang": """
        import sys, time
        sys.stdin.readline()
        time.sleep(60)
    """,
}


def project(tmp_path: Path, kind: str) -> TypeScriptChecker:
    (tmp_path / "node_modules" / "typescript").mkdir(parents=True)
    (tmp_path / "tscheck.mjs").write_text(textwrap.dedent(STAND_IN[kind]))
    return TypeScriptChecker(tmp_path, timeout_seconds=2, node=sys.executable)


def test_one_process_answers_many_checks(tmp_path):
    ts = project(tmp_path, "echo")
    try:
        assert ts.check("const a = 1;") == []
        pid = ts._proc.pid
        assert [d.code for d in ts.check("BAD")] == ["TS2304"]
        assert ts.check("const b = 2;") == []
        assert ts._proc.pid == pid
    finally:
        ts.close()


def test_a_crashed_checker_is_an_error_and_is_restarted(tmp_path):
    ts = project(tmp_path, "crash")
    try:
        with pytest.raises(TypeScriptCheckError, match="kaput"):
            ts.check("const a = 1;")
        (tmp_path / "tscheck.mjs").write_text(textwrap.dedent(STAND_IN["echo"]))
        assert ts.check("const a = 1;") == []
    finally:
        ts.close()


def test_a_hung_checker_times_out_and_is_killed(tmp_path):
    ts = project(tmp_path, "hang")
    began = time.monotonic()
    with pytest.raises(TypeScriptCheckError, match="timed out"):
        ts.check("const a = 1;")
    assert time.monotonic() - began < 10
    assert ts._proc is None


def test_missing_typescript_is_reported_not_passed(tmp_path):
    with pytest.raises(TypeScriptCheckError, match="typescript not found"):
        TypeScriptChecker(tmp_path).check("const a = 1;")


# --- the real checker ----------------------------------------------------------

REAL = Path(os.environ.get("TSCHECK_PROJECT_DIR", Path(__file__).parents[2] / "remotion_project"))
needs_real = pytest.mark.skipif(
    not (REAL / "node_modules" / "typescript").exists() or shutil.which("node") is None,
    reason="remotion_project/node_modules is not installed here",
)

SCRIPT = """import React from 'react';
import {AbsoluteFill, interpolate, useCurrentFrame} from 'remotion';
import {Stage} from './conceptflow-mini/primitives';

const PALETTE = {accent: '#F5B841'};

export function Shot1_1({duration}: {duration: number}) {
  const frame = useCurrentFrame();
  const t = interpolate(frame, [0, duration], [0, 1]);
  return <Stage><AbsoluteFill style={{backgroundColor: PALETTE.accent, opacity: t}} /></Stage>;
}
"""


@needs_real
def test_real_checker_passes_good_code_and_points_at_bad_lines():
    if REAL.resolve() != (Path(__file__).parents[2] / "remotion_project").resolve():
        shutil.copy(Path(__file__).parents[2] / "remotion_project" / "tscheck.mjs", REAL / "tscheck.mjs")
    ts = TypeScriptChecker(REAL, timeout_seconds=60)
    try:
        assert ts.check(SCRIPT) == []
        bad = SCRIPT.replace("PALETTE.accent", "PALETTE.nope").replace("const t =", "const t: string =")
        diags = ts.check(bad)
        assert sorted((d.line, d.code) for d in diags) == [(9, "TS2322"), (10, "TS2339")]
        assert ts.check(SCRIPT) == []  # nothing of the bad script lingers
        assert [d.code for d in ts.check("const x = ;")] == ["TS1109"]
    finally:
        ts.close()
