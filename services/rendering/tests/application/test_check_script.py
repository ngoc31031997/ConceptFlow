import pytest

from adapters.rendering.layout_checker import LayoutCheckError, LayoutScriptError
from adapters.rendering.typescript_checker import TypeScriptCheckError
from application.check_script import LAYOUT_DISABLED_WARNING, CheckScriptUseCase, LayoutContext
from application.validate_script import ScriptValidationError
from domain.errors import AnimationEngineError
from domain.layout_rules import SubtitleBand
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
    assert out.diagnostics[0].rule == "TS2304"  # CR-050 FR-22


def test_gate_and_tsc_failures_are_both_reported():
    validate = FakeValidate(
        ScriptValidationError("bad lottie", [LintIssue(line=7, message="unknown lottie id")])
    )
    out = make(validate, FakeTs([Diag(3, "TS1005", "';' expected.")])).check("remotion", "code", "creator")
    assert [d.line for d in out.diagnostics] == [7, 3]
    assert [d.rule for d in out.diagnostics] == ["", "TS1005"]  # lint issues carry no rule code


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


# --- CR-048 T6b: the layout check after tsc -------------------------------------------


def box(x, y, w, h):
    return {"x": x, "y": y, "w": w, "h": h}


def probe(elements, font_ok=True, line=10):
    samples = [{"pct": p, "frame": round(p * 149), "elements": elements} for p in (0, 0.5, 127 / 149, 1)]
    return {"version": 1, "composition": {"width": 1920, "height": 1080, "fps": 30},
            "font": {"family": "Be Vietnam Pro", "available": font_ok}, "shots_line": 30,
            "shots": [{"index": 0, "id": "1.1", "component": "Shot1_1", "line": line, "samples": samples}]}


TOOTH_OUT = {"kind": "kit", "component": "Tooth", "line": 12, "rect": box(1500, 300, 420, 420), "opacity": 1}
SMALL_APPLE = {"kind": "kit", "component": "Apple", "line": 13, "rect": box(800, 400, 200, 200), "opacity": 1}


class FakeLayout:
    def __init__(self, result=None, error=None):
        self.result, self.error, self.seen = result, error, []

    def measure(self, code, font=""):
        self.seen.append((code, font))
        if self.error:
            raise self.error
        return self.result


def test_blocking_layout_violations_become_line_numbered_layout_diagnostics():
    layout = FakeLayout(probe([TOOTH_OUT]))
    out = CheckScriptUseCase(FakeValidate(), FakeTs(), layout).check("remotion", "code", "creator")
    assert not out.ok and out.warnings == []
    [d] = out.diagnostics
    assert d.kind == "layout" and d.line == 12 and d.rule == "safe_area"
    assert d.message == "Shot 1.1, mọi frame đo: hình Tooth ra ngoài vùng an toàn (phải x=1920 > 1824)"


def test_non_blocking_findings_are_warnings_and_the_check_passes():
    out = CheckScriptUseCase(FakeValidate(), FakeTs(), FakeLayout(probe([SMALL_APPLE]))).check(
        "remotion", "code", "creator")
    assert out.ok and out.diagnostics == []
    assert out.warnings == ["Bố cục: Shot 1.1: vật lớn nhất (hình Apple) chỉ chiếm 19% chiều khung (< 30%) — "
                            "khung dễ thành nền trống với vật nhỏ lọt thỏm (dòng 13)"]


def test_the_subtitle_band_and_font_reach_the_layout_check():
    layout = FakeLayout(probe([{**TOOTH_OUT, "rect": box(700, 600, 420, 300)}]))
    ctx = LayoutContext(subtitle_band=SubtitleBand("bottom", 240), video_font="Montserrat")
    out = CheckScriptUseCase(FakeValidate(), FakeTs(), layout).check("remotion", "code", "creator", ctx)
    assert layout.seen == [("code", "Montserrat")]
    assert [d.message.split(": ", 1)[1] for d in out.diagnostics] == [
        "hình Tooth lấn vùng phụ đề ở mép dưới khung (mép dưới y=900 > 840)"]


@pytest.mark.parametrize("error", [
    LayoutCheckError("bộ đo bố cục không mở được trình duyệt: Executable doesn't exist"),
    LayoutCheckError("đo bố cục quá 60s (đo)"),
    LayoutScriptError("script không nạp được vào trang đo: the script never called registerRoot()"),
])
def test_a_layout_check_that_cannot_run_is_an_explicit_warning_and_tsc_stands(error):
    use_case = CheckScriptUseCase(FakeValidate(), FakeTs(), FakeLayout(error=error))
    out = use_case.check("remotion", "code", "creator")
    assert out.ok and out.diagnostics == []
    [w] = out.warnings
    assert w.startswith("Bố cục: KHÔNG kiểm tra được — ") and str(error) in w


def test_measurements_the_rules_cannot_read_are_not_a_pass():
    broken = probe([{"kind": "kit", "component": "Tooth", "line": 12, "opacity": 1}])  # no rect
    use_case = CheckScriptUseCase(FakeValidate(), FakeTs(), FakeLayout(broken))
    out = use_case.check("remotion", "code", "creator")
    assert out.ok and out.diagnostics == []
    assert "KHÔNG kiểm tra được — số đo không đọc được" in out.warnings[0]


def test_a_missing_font_means_not_checked_rather_than_numbers_in_the_wrong_font():
    out = CheckScriptUseCase(FakeValidate(), FakeTs(), FakeLayout(probe([TOOTH_OUT], font_ok=False))).check(
        "remotion", "code", "creator")
    assert out.ok and out.diagnostics == []
    assert "font 'Be Vietnam Pro' không có trong trình duyệt đo" in out.warnings[0]


def test_layout_is_not_measured_when_tsc_fails_and_never_for_manim():
    layout = FakeLayout(probe([TOOTH_OUT]))
    out = CheckScriptUseCase(FakeValidate(), FakeTs([Diag(3, "TS1005", "';' expected.")]), layout).check(
        "remotion", "code", "creator")
    assert [d.kind for d in out.diagnostics] == ["compile"] and layout.seen == []
    out = CheckScriptUseCase(FakeValidate(), FakeTs(), layout).check("manim", "code", "XScene")
    assert out.ok and out.warnings == [] and layout.seen == []


def test_a_disabled_layout_check_says_so():
    out = make().check("remotion", "code", "creator")
    assert out.ok and out.warnings == [LAYOUT_DISABLED_WARNING]
