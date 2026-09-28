"""The warm layout checker (remotion_project/layout_check.mjs + its Python client), CR-048 T6b.

Process behaviour is exercised against stand-in checkers run by this Python
interpreter, so those tests need neither Node nor a browser. The real checker
runs when remotion_project/node_modules and a Chromium headless shell are
present (the rendering image; or here, with Playwright's browsers installed).
"""

from __future__ import annotations

import json
import os
import shutil
import sys
import textwrap
import time
from pathlib import Path

import pytest

from adapters.rendering.layout_checker import LayoutChecker, LayoutCheckError, LayoutScriptError

PROBE = {"version": 1, "composition": {"width": 1920, "height": 1080, "fps": 30}, "shots": [],
         "font": {"family": "Be Vietnam Pro", "available": True}}

STAND_IN = {
    "echo": f"""
        import json, sys
        print(json.dumps({{"ready": True, "browser": {{"version": "141"}}}}), flush=True)
        for line in sys.stdin:
            req = json.loads(line)
            if "LOADFAIL" in req["code"]:
                print(json.dumps({{"id": req["id"], "error": "the script never called registerRoot()", "stage": "load"}}), flush=True)
            elif "MEASUREFAIL" in req["code"]:
                print(json.dumps({{"id": req["id"], "error": "Target closed", "stage": "measure"}}), flush=True)
            elif "GARBAGE" in req["code"]:
                print(json.dumps({{"id": req["id"], "result": {{"version": 7}}}}), flush=True)
            else:
                out = dict({PROBE!r}, font={{"family": req.get("font", ""), "available": True}})
                print(json.dumps({{"id": req["id"], "result": out}}), flush=True)
    """,
    "fatal": """
        import json
        print(json.dumps({"fatal": "could not start Chromium: Executable doesn't exist"}), flush=True)
    """,
    "crash": """
        import json, sys
        print(json.dumps({"ready": True}), flush=True)
        sys.stdin.readline()
        sys.stderr.write("Error: page crashed\\n")
        sys.exit(1)
    """,
    "hang": """
        import json, sys, time
        print(json.dumps({"ready": True}), flush=True)
        sys.stdin.readline()
        time.sleep(60)
    """,
    "silent_start": """
        import time
        time.sleep(60)
    """,
}


def project(tmp_path: Path, kind: str, timeout: int = 2) -> LayoutChecker:
    for dep in ("playwright-core", "@remotion/player", "esbuild", "typescript"):
        (tmp_path / "node_modules" / dep).mkdir(parents=True, exist_ok=True)
    (tmp_path / "layout_check.mjs").write_text(textwrap.dedent(STAND_IN[kind]))
    return LayoutChecker(tmp_path, timeout_seconds=timeout, node=sys.executable)


def test_one_process_answers_many_checks_and_passes_the_font(tmp_path):
    lc = project(tmp_path, "echo")
    try:
        assert lc.measure("const a = 1;", "Montserrat")["font"]["family"] == "Montserrat"
        pid = lc._proc.pid
        assert lc.measure("const b = 2;")["version"] == 1
        assert lc._proc.pid == pid and lc.browser_version == "141"
    finally:
        lc.close()


def test_a_script_that_does_not_load_is_its_own_error(tmp_path):
    lc = project(tmp_path, "echo")
    try:
        with pytest.raises(LayoutScriptError, match="registerRoot"):
            lc.measure("LOADFAIL")
        with pytest.raises(LayoutCheckError, match=r"thất bại \(measure\)"):
            lc.measure("MEASUREFAIL")
        with pytest.raises(LayoutCheckError, match="không đúng dạng"):
            lc.measure("GARBAGE")
    finally:
        lc.close()


def test_a_browser_that_cannot_start_is_an_error_never_a_pass(tmp_path):
    with pytest.raises(LayoutCheckError, match="không mở được trình duyệt.*Executable doesn't exist"):
        project(tmp_path, "fatal").measure("const a = 1;")


def test_a_crashed_checker_is_an_error_and_is_restarted(tmp_path):
    lc = project(tmp_path, "crash")
    try:
        with pytest.raises(LayoutCheckError, match="page crashed"):
            lc.measure("const a = 1;")
        (tmp_path / "layout_check.mjs").write_text(textwrap.dedent(STAND_IN["echo"]))
        assert lc.measure("const a = 1;")["version"] == 1
    finally:
        lc.close()


@pytest.mark.parametrize("kind", ["hang", "silent_start"])
def test_a_hung_checker_times_out_and_is_killed(tmp_path, kind):
    lc = project(tmp_path, kind, timeout=1)
    began = time.monotonic()
    with pytest.raises(LayoutCheckError, match="quá 1s"):
        lc.measure("const a = 1;")
    assert time.monotonic() - began < 10
    assert lc._proc is None


def test_warm_swallows_a_start_failure_that_the_first_check_then_reports(tmp_path):
    lc = project(tmp_path, "fatal")
    lc.warm()
    assert lc._proc is None
    with pytest.raises(LayoutCheckError):
        lc.measure("const a = 1;")


def test_missing_packages_are_reported(tmp_path):
    (tmp_path / "layout_check.mjs").write_text("")
    with pytest.raises(LayoutCheckError, match="thiếu gói"):
        LayoutChecker(tmp_path, node=sys.executable).measure("const a = 1;")


# --- the real checker, end to end through the compile check ---------------------------

REAL = Path(__file__).parents[2] / "remotion_project"
SAMPLES = REAL / "layout_probe_samples"


def _browser_available() -> bool:
    explicit = os.environ.get("LAYOUT_PROBE_BROWSER")
    if explicit:
        return Path(explicit).exists()
    root = Path(os.environ.get("PLAYWRIGHT_BROWSERS_PATH") or Path.home() / ".cache" / "ms-playwright")
    return any(root.glob("chromium_headless_shell-*/chrome-linux/headless_shell"))


needs_real = pytest.mark.skipif(
    not (REAL / "node_modules" / "playwright-core").exists() or shutil.which("node") is None
    or not _browser_available(),
    reason="cần remotion_project/node_modules và Chromium headless shell của Playwright",
)


class PassingGate:
    def validate(self, request):
        return None


class PassingTsc:
    def check(self, code):
        return []


@pytest.fixture(scope="module")
def real_checker():
    lc = LayoutChecker(REAL, timeout_seconds=60)
    yield lc
    lc.close()


def _owner(lines: dict, line: int) -> str | None:
    return next((sid for sid, (a, b) in lines.items() if a <= line <= b), None)


def _check(real_checker, name: str, band=None):
    from application.check_script import CheckScriptUseCase, LayoutContext

    use_case = CheckScriptUseCase(PassingGate(), PassingTsc(), real_checker)
    code = (SAMPLES / f"{name}.tsx").read_text(encoding="utf-8")
    lines = json.loads((SAMPLES / f"{name}.lines.json").read_text(encoding="utf-8"))
    return use_case.check("remotion", code, "creator", LayoutContext(subtitle_band=band)), lines


@needs_real
def test_real_problems_script_gives_the_three_planted_faults_on_the_right_shots(real_checker):
    out, lines = _check(real_checker, "problems")
    assert not out.ok and out.warnings == []
    got = sorted((_owner(lines, d.line), d.kind, d.message.split(": ", 1)[1].split(" (")[0]) for d in out.diagnostics)
    assert got == [
        ("1.2", "layout", "nhãn 'Lớp men răng bảo vệ' tràn khung chữ"),
        ("1.3", "layout", "hình Germ ra ngoài vùng an toàn"),
        ("1.3", "layout", "hình Tooth ra ngoài vùng an toàn"),
        ("2.1", "layout", "nhãn 'Đánh răng hai lần mỗi ngày'"),
    ]


@needs_real
def test_real_clean_script_passes_with_no_warning(real_checker):
    out, _ = _check(real_checker, "clean")
    assert out.ok and out.diagnostics == [] and out.warnings == []


@needs_real
def test_real_overshoot_is_caught(real_checker):
    out, lines = _check(real_checker, "overshoot")
    [d] = out.diagnostics
    assert _owner(lines, d.line) == "1.1" and "hình Sun ra ngoài vùng an toàn" in d.message


@needs_real
def test_real_thirty_shot_video_is_measured_well_under_the_timeout(real_checker):
    code = (SAMPLES / "timing30.tsx").read_text(encoding="utf-8")
    began = time.monotonic()
    probe = real_checker.measure(code)
    took = time.monotonic() - began
    assert len(probe["shots"]) == 30 and all(len(s["samples"]) == 12 for s in probe["shots"])
    assert probe["font"]["available"] is True
    print(f"\n30 shots x 12 samples: {took:.2f}s (measure {probe['timing_ms']['measure']} ms)")
    assert took < 30


@needs_real
def test_real_one_chunk_check_skips_the_mergers_stubs(real_checker):
    # llm-service checks each chunk against `return null` stubs for the shots
    # other chunks own (merger.remotion_stub); drawing them would cost ~12 ms a sample for nothing.
    code = (SAMPLES / "problems.tsx").read_text(encoding="utf-8").split("\n")
    lines = json.loads((SAMPLES / "problems.lines.json").read_text(encoding="utf-8"))
    a, b = lines["1.3"]
    stubbed = code[: a - 1] + ["function Shot1_3({duration}: ShotProps) {", "  return null;", "}"] + code[b:]
    probe = real_checker.measure("\n".join(stubbed))
    by_id = {s["id"]: s for s in probe["shots"]}
    assert by_id["1.3"].get("skipped") == "stub" and by_id["1.3"]["samples"] == []
    assert all(len(s["samples"]) == 12 for sid, s in by_id.items() if sid != "1.3")


@needs_real
def test_real_script_that_throws_while_loading_is_reported_as_not_checked(real_checker):
    from application.check_script import CheckScriptUseCase

    out = CheckScriptUseCase(PassingGate(), PassingTsc(), real_checker).check(
        "remotion", "export const narrations: string[] = [];\n", "creator")
    assert out.ok and out.diagnostics == []
    assert len(out.warnings) == 1 and "KHÔNG kiểm tra được" in out.warnings[0] and "registerRoot" in out.warnings[0]
