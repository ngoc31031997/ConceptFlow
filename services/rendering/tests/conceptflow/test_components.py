"""Component dựng cảnh (CR-017 FR45).

`importorskip` giữ đúng tính chất mà test suite của Rendering đang có: chạy được
mà không cần cài manim (README). Ai có manim thì được kiểm thêm phần này; CI
trong image thì luôn có.
"""

import pytest

pytest.importorskip("manim", reason="component cần manim thật để đo bounding box")

from conceptflow import (  # noqa: E402
    BarChart,
    Callout,
    DataTable,
    FlowDiagram,
    FunctionPlot,
    Timeline,
    CodePanel,
    ComparisonSplit,
    Recap,
    StepList,
    TitleCard,
)
from conceptflow.api import PUBLIC_NAMES  # noqa: E402
from conceptflow.layout import Box, overflow  # noqa: E402

CASES = [
    lambda: TitleCard("Vòng lặp for trong Java", "Khởi tạo · Điều kiện · Bước nhảy"),
    lambda: Callout("Đừng quên tăng biến đếm", tone="warning"),
    lambda: CodePanel("for i in range(5):\n    print(i)", "python"),
    # `++`/`<=` là ligature lập trình của JetBrains Mono (qua OpenType "calt",
    # thứ Manim's disable_ligatures không tắt được) — Pango gộp 2 ký tự thành 1
    # glyph, làm `Text._gen_chars` lệch số và crash IndexError. Ca C-style loop
    # này (không phải Python for-range) là ca thật Creator gặp; giữ nó ở đây để
    # không lặp lại lỗ hổng coverage cũ.
    lambda: CodePanel(
        "for (int i = 1; i <= 5; i++) {\n    System.out.println(i);\n}", "java"
    ),
    lambda: StepList(["Khởi tạo biến đếm", "Kiểm tra điều kiện", "Tăng biến đếm"]),
    lambda: ComparisonSplit("while", "Kiểm tra trước", "do-while", "Chạy trước"),
    lambda: Recap(["for gồm ba phần", "Quên tăng biến đếm là lặp vô hạn"]),
    lambda: FlowDiagram(["Request", "Controller", "Service", "Database"]),
    lambda: FlowDiagram(["Viết code", "Biên dịch", "Chạy"], direction="down"),
    lambda: BarChart(["O(1)", "O(log n)", "O(n)", "O(n²)"], [1, 3, 8, 64]),
    lambda: FunctionPlot(lambda x: x * x, (-3, 3), label="y = x²"),
    lambda: DataTable(["Kiểu", "Kích thước"], [["int", "4 byte"], ["long", "8 byte"]]),
    lambda: Timeline([("1991", "Python ra đời"), ("2008", "Python 3"), ("2020", "Hết hỗ trợ Python 2")]),
]


@pytest.mark.parametrize("build", CASES, ids=lambda b: b().__class__.__name__)
def test_component_nam_gon_trong_khung_an_toan(build):
    assert overflow(Box.from_mobject(build())) == ()


@pytest.mark.parametrize("length", [1, 4, 10])
def test_step_list_dai_bao_nhieu_cung_khong_tran(length):
    """Nội dung dài phải được co lại, không được tràn ra ngoài khung."""
    items = [f"Bước số {i} trong quy trình xử lý" for i in range(length)]
    assert overflow(Box.from_mobject(StepList(items))) == ()


def test_tieu_de_rat_dai_van_lot_khung():
    title = "Một tiêu đề cực kỳ dài dùng để kiểm tra xem component có tự co lại hay không"
    assert overflow(Box.from_mobject(TitleCard(title))) == ()


def test_api_surface_khop_voi_whitelist_cua_lint():
    """Lint đọc `api.PUBLIC_NAMES`, script đọc `__all__`. Lệch nhau thì lint sẽ
    từ chối đúng thứ mà thư viện vừa cấp cho script."""
    import conceptflow

    assert set(conceptflow.__all__) == set(PUBLIC_NAMES)


def test_tone_la_cua_callout_roi_ve_accent():
    """Không raise: một chú thích sai màu vẫn là khung hình xem được."""
    assert overflow(Box.from_mobject(Callout("x", tone="khong-ton-tai"))) == ()


def test_template_khoi_dau_phai_qua_duoc_lint():
    """Template mà chính lint từ chối là cách chắc chắn nhất để Creator mất
    niềm tin vào cả hệ thống. Đây là bản gốc của template trong scriptTemplates.ts."""
    import pathlib

    from domain.script_lint import blocking_issues, lint_manim_script

    source = (
        pathlib.Path(__file__).parent.parent / "fixtures" / "conceptflow_template.py"
    ).read_text(encoding="utf-8")

    assert blocking_issues(lint_manim_script(source)) == []
    # CR-018: lời thoại là self.narrate(...), không phải marker + wait rời rạc.
    # Một template còn dùng chuẩn cũ sẽ có narration_segments rỗng khi render
    # thật (render_script.py từ chối) — lint không bắt được việc này vì đây là
    # quy ước ngữ nghĩa, không phải API sai.
    assert "self.narrate(" in source
    assert "# NARRATION:" not in source
    assert "self.wait(AUTO)" not in source


@pytest.mark.parametrize("length", [2, 6, 12])
def test_flow_diagram_nhieu_buoc_van_lot_khung(length):
    steps = [f"Bước {i}" for i in range(length)]
    assert overflow(Box.from_mobject(FlowDiagram(steps))) == ()


def test_bar_chart_cot_thap_dung_chung_duong_day():
    """Cột thấp phải đứng trên cùng đường đáy với cột cao, không lơ lửng."""
    chart = BarChart(["a", "b", "c"], [1, 10, 4])
    bottoms = {round(float(bar.get_bottom()[1]), 4) for bar in chart.bars}
    assert len(bottoms) == 1


@pytest.mark.parametrize(
    "build",
    [
        lambda: BarChart([], []),
        lambda: BarChart(["a"], [-1]),
        lambda: DataTable(["a", "b"], [["chỉ một ô"]]),
        lambda: FunctionPlot(lambda x: x, (2, 1)),
        lambda: FlowDiagram([]),
    ],
)
def test_dau_vao_sai_bao_loi_ro_rang(build):
    with pytest.raises(ValueError):
        build()


def test_function_plot_bo_qua_diem_khong_xac_dinh():
    """1/x không xác định tại 0 — trục y vẫn phải tính được từ các điểm còn lại."""
    from conceptflow.components.function_plot import _y_bounds

    low, high = _y_bounds(lambda x: 1 / x, -2, 2)
    assert low < 0 < high


def test_scene_methods_khop_voi_method_that_cua_scene():
    """`api.SCENE_METHODS` là thứ prompt liệt kê cho LLM; lệch với scene thật thì
    LLM hoặc gọi method không tồn tại, hoặc không biết method mới có."""
    from conceptflow import ConceptFlowScene
    from conceptflow.api import SCENE_METHODS

    own = {
        name
        for name, value in vars(ConceptFlowScene).items()
        if callable(value) and not name.startswith("_")
    }
    # `play` là override của Manim; `stage_mobjects`, `is_zoomed`,
    # `added_during_zoom` là helper cho QC/narration, không phải thứ script cần gọi.
    own -= {"play", "stage_mobjects", "is_zoomed", "added_during_zoom"}
    assert own == set(SCENE_METHODS)
