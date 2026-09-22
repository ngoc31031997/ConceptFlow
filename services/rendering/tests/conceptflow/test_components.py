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
    ConceptFlowScene,
    DataTable,
    FlowDiagram,
    FunctionPlot,
    Readout,
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
    lambda: Readout(0, label="phép so sánh", unit="lần"),
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


# --- Hình cơ bản và số chạy theo theme (thay cho API thô của Manim) ----------


@pytest.fixture
def scene():
    return ConceptFlowScene()


@pytest.mark.parametrize(
    "kind,kwargs",
    [
        ("rect", {}),
        ("square", {}),
        ("circle", {}),
        ("dot", {}),
        ("polygon", {"points": [(-1, 0), (1, 0), (0, 1.5)]}),
    ],
)
def test_shape_mang_mau_cua_theme(scene, kind, kwargs):
    """Mọi hình cơ bản phải lấy màu từ theme — đó là lý do chúng tồn tại."""
    mobject = scene.shape(kind, tone="success", **kwargs)
    color = mobject.get_color() if kind == "dot" else mobject.get_stroke_color()
    assert str(color).upper() == scene.theme.success.upper()


def test_shape_tone_la_roi_ve_accent(scene):
    """Như `Callout`: sắc thái sai không được làm hỏng cả lượt render."""
    assert str(scene.shape("rect", tone="khong-ton-tai").get_stroke_color()).upper() == (
        scene.theme.accent.upper()
    )


@pytest.mark.parametrize(
    "build",
    [
        lambda s: s.shape("polygon", points=[(0, 0)]),
        lambda s: s.path((0, 0)),
    ],
)
def test_hinh_thieu_diem_bao_loi_ro_rang(scene, build):
    with pytest.raises(ValueError):
        build(scene)


def test_path_nhieu_diem_di_qua_dung_cac_diem(scene):
    """`path` là quỹ đạo cho `travel`, nên nó phải chạm đúng các điểm đã cho."""
    line = scene.path((-3, -1), (0, 2), (3, -1))
    assert float(line.get_start()[0]) == pytest.approx(-3)
    assert float(line.get_end()[0]) == pytest.approx(3)
    assert float(line.get_top()[1]) == pytest.approx(2)


def test_connect_cong_khong_trung_voi_connect_thang(scene):
    """Hai chiều của cùng một cặp vật phải phân biệt được bằng mắt, nếu không
    thì mũi tên đi và mũi tên về chồng khít lên nhau."""
    a = scene.shape("rect")
    b = scene.shape("circle").shift([4, 0, 0])
    straight = scene.connect(a, b)[0]
    curved = scene.connect(a, b, style="curved")[0]
    assert float(curved.point_from_proportion(0.5)[1]) != pytest.approx(
        float(straight.point_from_proportion(0.5)[1])
    )


def test_connect_khong_cam_vao_giua_vat(scene):
    """`CurvedArrow` chỉ nhận toạ độ; đưa thẳng tâm vào thì mũi tên xuyên qua vật."""
    a = scene.shape("rect")
    b = scene.shape("circle").shift([5, 0, 0])
    arrow = scene.connect(a, b, style="curved")[0]
    assert float(arrow.get_start()[0]) > float(a.get_right()[0])
    assert float(arrow.get_end()[0]) < float(b.get_left()[0])


def test_brace_co_nhan_nam_ngoai_vat(scene):
    box = scene.shape("rect")
    group = scene.brace(box, "một chu kỳ")
    assert len(group) == 2
    assert float(group[1].get_top()[1]) < float(box.get_bottom()[1])


def test_so_chay_bam_theo_group_khi_script_doi_cho():
    """Bản sống dựng lại mỗi frame, nên nếu nó không bám vào bóng thì nhãn đi
    lên còn con số vẫn nằm giữa khung — lỗi thật, thấy được trên khung hình."""
    readout = Readout(0, label="phép so sánh", unit="lần")
    readout.shift([0, 2.6, 0])
    readout.update()  # ép `always_redraw` chạy như trong một frame thật
    assert float(readout.number.get_center()[1]) == pytest.approx(
        float(readout._ghost.get_center()[1]), abs=0.05
    )


def test_so_chay_co_theo_khi_bi_stack_thu_nho():
    """`stack()` co cả cụm; cỡ chữ đã chốt thành số nên phải co theo bóng."""
    scene = ConceptFlowScene()
    readout = Readout(0, label="vòng")
    before = readout.number.height
    scene.stack(scene.heading("Số vòng lặp"), readout, scene.body("x" * 120))
    readout.update()
    assert readout.number.height < before


def test_so_chay_doi_gia_tri_theo_tracker():
    readout = Readout(0, decimals=0)
    readout.tracker.set_value(128)
    readout.update()
    assert readout.value == pytest.approx(128)
    assert len(readout.number[0]) == 3  # ba chữ số trên màn hình


def test_emphasize_kieu_khoanh_khac_kieu_phong():
    """Hai kiểu nhấn phải là hai animation khác nhau, không phải một tham số trang trí."""
    from conceptflow.transitions import emphasize_animation

    square = Readout(1)
    assert type(emphasize_animation(square, 0.8, "circle", "#FFFFFF")).__name__ == (
        "Circumscribe"
    )
    assert type(emphasize_animation(square, 0.8)).__name__ == "Indicate"


def test_reveal_mot_readout_la_fade_chu_khong_phai_ve_dan():
    """`Create` vẽ dần nét chữ số (trông như lỗi font) và đánh nhau với bản
    sống dựng lại mỗi frame — từng làm cả lượt render crash."""
    from conceptflow.transitions import reveal_animation

    assert type(reveal_animation(Readout(0), 0.8)).__name__ == "FadeIn"
