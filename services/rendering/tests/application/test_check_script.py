import pytest

from adapters.rendering.typescript_checker import TypeScriptCheckError
from application.check_script import CheckScriptUseCase
from application.validate_script import ScriptValidationError
from domain.errors import AnimationEngineError
from domain.script_lint import LintIssue


class FakeValidate:
    def __init__(self, raises=None):
        self.raises = raises
        self.requests = []

    def validate(self, request):
        self.requests.append(request)
        if self.raises:
            raise self.raises


class FakeTs:
    def __init__(self, diags=None, error=None):
        self.diags, self.error, self.seen = diags or [], error, []

    def check(self, code):
        self.seen.append(code)
        if self.error:
            raise self.error
        return self.diags


class Diag:
    def __init__(self, line, code, message):
        self.line, self.column, self.code, self.message = line, 1, code, message


def make(validate=None, ts=None):
    return CheckScriptUseCase(validate or FakeValidate(), ts or FakeTs())


def test_remotion_pass_runs_both_the_gate_and_tsc():
    validate, ts = FakeValidate(), FakeTs()
    out = make(validate, ts).check("remotion", "export const narrations = ['a'];", "creator")
    assert out.ok and out.diagnostics == []
    assert validate.requests[0].engine == "remotion" and ts.seen == ["export const narrations = ['a'];"]


def test_remotion_reports_tsc_diagnostics_with_lines():
    out = make(ts=FakeTs([Diag(12, "TS2304", "Cannot find name 'x'.")])).check("remotion", "code", "creator")
    assert not out.ok and out.diagnostics[0].line == 12 and "TS2304" in out.diagnostics[0].message


def test_gate_and_tsc_failures_are_both_reported():
    validate = FakeValidate(ScriptValidationError("bad lottie", [LintIssue(line=7, message="unknown lottie id")]))
    out = make(validate, FakeTs([Diag(3, "TS1005", "';' expected.")])).check("remotion", "code", "creator")
    assert [d.line for d in out.diagnostics] == [7, 3]


def test_manim_dry_run_failure_keeps_the_traceback_in_raw_and_never_runs_tsc():
    trace = 'Traceback\n  File "s.py", line 9, in shot_1_2\nNameError: x'
    ts = FakeTs()
    out = make(FakeValidate(AnimationEngineError(trace)), ts).check("manim", "code", "TcpScene")
    assert not out.ok and out.raw == trace and ts.seen == []


def test_a_checker_that_cannot_run_raises_instead_of_passing():
    with pytest.raises(TypeScriptCheckError):
        make(ts=FakeTs(error=TypeScriptCheckError("tsc not found"))).check("remotion", "code", "creator")


@pytest.mark.parametrize("engine,code", [("python", "x"), ("remotion", "  ")])
def test_bad_input_is_a_value_error(engine, code):
    with pytest.raises(ValueError):
        make().check(engine, code, "s")
