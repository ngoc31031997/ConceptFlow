"""Cổng kiểm tra trước TTS."""

from __future__ import annotations

import dataclasses

import pytest

from application.validate_script import ScriptValidationError, ValidateScriptUseCase
from domain.errors import UnsupportedEngineError
from domain.models import DryRunResult, ScriptRenderRequest
from domain.ports import ScriptRendererPort

VALID = (
    "import { registerRoot, Composition } from 'remotion';\n"
    "export const narrations: string[] = ['một'];\n"
    "const Root = () => <Composition id=\"DemoVideo\" component={X} />;\n"
    "registerRoot(Root);\n"
)


class FakeRenderer(ScriptRendererPort):
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
        scene_class_name="DemoVideo",
        narration_segments=[],
    )


def test_tra_ve_loi_thoai_tu_luot_dry():
    renderer = FakeRenderer(DryRunResult(narrations=["một", "hai"], beats=[(0, "hook")]))

    result = ValidateScriptUseCase(renderer).validate(make_request())

    assert result.dry_run.narrations == ["một", "hai"]
    assert result.dry_run.beats == [(0, "hook")]
    assert len(renderer.dry_runs) == 1


def test_tu_choi_script_rong():
    with pytest.raises(ScriptValidationError):
        ValidateScriptUseCase(FakeRenderer()).validate(make_request("   "))


def test_engine_khac_remotion_bi_tu_choi_truoc_luot_dry():
    renderer = FakeRenderer()

    with pytest.raises(UnsupportedEngineError, match="manim"):
        ValidateScriptUseCase(renderer).validate(dataclasses.replace(make_request(), engine="manim"))

    assert renderer.dry_runs == []


REMOTION_WITH_CLIP = (
    "export const narrations: string[] = ['một'];\n"
    "const A = () => <LottieClip id=\"cat.idle\" />;\n"
)


def test_chan_lottie_id_ngoai_danh_muc_truoc_luot_dry():
    renderer = FakeRenderer()
    use_case = ValidateScriptUseCase(renderer, lambda: {"cat.thinking"})
    with pytest.raises(ScriptValidationError, match="cat.idle"):
        use_case.validate(make_request(REMOTION_WITH_CLIP))
    assert renderer.dry_runs == []


def test_cho_qua_khi_id_da_duyet():
    use_case = ValidateScriptUseCase(FakeRenderer(), lambda: {"cat.idle"})
    assert use_case.validate(make_request(REMOTION_WITH_CLIP)).dry_run is not None


def test_tu_tim_ten_composition_khi_request_de_trong():
    """Không có bước parse_script, nên tên composition rỗng là bình thường."""
    renderer = FakeRenderer()

    result = ValidateScriptUseCase(renderer).validate(
        dataclasses.replace(make_request(), scene_class_name="")
    )

    assert result.scene_class_name == "DemoVideo" and result.engine == "remotion"
    assert renderer.dry_runs[0].scene_class_name == "DemoVideo"


def test_khong_tim_thay_composition_la_loi_validate():
    renderer = FakeRenderer()

    with pytest.raises(ScriptValidationError, match="không tìm thấy Remotion composition"):
        ValidateScriptUseCase(renderer).validate(
            dataclasses.replace(make_request("x = 1\n"), scene_class_name="")
        )

    assert renderer.dry_runs == []
