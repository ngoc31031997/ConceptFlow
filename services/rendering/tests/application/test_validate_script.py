"""Cổng kiểm tra trước TTS (CR-020 FR56)."""

from __future__ import annotations

import dataclasses

import pytest

from application.validate_script import ScriptValidationError, ValidateScriptUseCase
from domain.models import DryRunResult, ScriptRenderRequest
from domain.ports import ManimScriptRendererPort

VALID = (
    "from conceptflow import *\n\n"
    "class DemoScene(ConceptFlowScene):\n"
    "    def construct(self):\n"
    '        self.narrate("một")\n'
)


class FakeRenderer(ManimScriptRendererPort):
    def __init__(self, result: DryRunResult | None = None) -> None:
        self.dry_runs: list[ScriptRenderRequest] = []
        self._result = result or DryRunResult(narrations=["một"])

    def dry_run(self, request):
        self.dry_runs.append(request)
        return self._result

    def render(self, request, output_path):  # pragma: no cover - không dùng ở đây
        raise AssertionError("validate không được gọi render")


def make_request(script: str = VALID) -> ScriptRenderRequest:
    return ScriptRenderRequest(
        project_id="proj-1",
        script_content=script,
        scene_class_name="DemoScene",
        narration_segments=[],
    )


def test_tra_ve_loi_thoai_tu_luot_dry():
    renderer = FakeRenderer(DryRunResult(narrations=["một", "hai"], beats=[(0, "hook")]))

    result = ValidateScriptUseCase(renderer).validate(make_request())

    assert result.dry_run.narrations == ["một", "hai"]
    assert result.dry_run.beats == [(0, "hook")]
    assert len(renderer.dry_runs) == 1


def test_loi_lint_chan_truoc_khi_chay_luot_dry():
    """Chạy một subprocess Manim cho script đã biết là sai chỉ tốn thêm thời
    gian để nhận cùng câu trả lời."""
    renderer = FakeRenderer()
    script = VALID + "\nr = Rectangle(width=2)\n"

    with pytest.raises(ScriptValidationError, match="Rectangle"):
        ValidateScriptUseCase(renderer).validate(make_request(script))

    assert renderer.dry_runs == []


def test_canh_bao_khong_chan_va_duoc_tra_ve():
    """Đường thoát hiểm ra API thô của Manim là hợp lệ (CR-017 FR46.3)."""
    renderer = FakeRenderer()
    script = "from conceptflow import *\nfrom manim import Arrow\n" + VALID.split("\n", 1)[1]

    result = ValidateScriptUseCase(renderer).validate(make_request(script))

    assert [w.severity for w in result.warnings] == ["warning"]
    assert len(renderer.dry_runs) == 1


def test_tu_choi_script_rong():
    with pytest.raises(ScriptValidationError):
        ValidateScriptUseCase(FakeRenderer()).validate(make_request("   "))


REMOTION_WITH_CLIP = (
    "export const narrations: string[] = ['một'];\n"
    "const A = () => <LottieClip id=\"cat.idle\" />;\n"
)


def make_remotion_request(script: str) -> ScriptRenderRequest:
    return dataclasses.replace(make_request(script), engine="remotion")


def test_remotion_chan_lottie_id_ngoai_danh_muc():
    use_case = ValidateScriptUseCase(FakeRenderer(), lambda: {"cat.thinking"})
    with pytest.raises(ScriptValidationError, match="cat.idle"):
        use_case.validate(make_remotion_request(REMOTION_WITH_CLIP))


def test_remotion_cho_qua_khi_id_da_duyet():
    use_case = ValidateScriptUseCase(FakeRenderer(), lambda: {"cat.idle"})
    assert use_case.validate(make_remotion_request(REMOTION_WITH_CLIP)).dry_run is not None


def test_manim_khong_bi_lint_lottie():
    # Script Manim không bao giờ chứa LottieClip; lint chỉ chạy cho engine remotion.
    use_case = ValidateScriptUseCase(FakeRenderer(), lambda: set())
    use_case.validate(make_request())
