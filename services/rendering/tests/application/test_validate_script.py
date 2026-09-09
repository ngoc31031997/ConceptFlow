"""Cổng kiểm tra trước TTS (CR-020 FR56)."""

from __future__ import annotations

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
