"""Lint theo whitelist (CR-017 FR46).

Bộ test cũ kiểm tra blacklist `INVALID_KWARGS_BY_CLASS` (corner_radius trên
Rectangle...). Blacklist đó đã bị gỡ: sau CR-017, `Rectangle` bản thân nó đã
không thuộc API cho phép, nên không còn ý nghĩa gì khi hỏi nó nhận kwarg nào.
"""

from domain.script_lint import BLOCKING, WARNING, blocking_issues, lint_manim_script

VALID = """from conceptflow import *


class DemoScene(ConceptFlowScene):
    def construct(self):
        card = TitleCard("Vòng lặp for", "Ba phần")
        self.reveal(card)
        self.dismiss(card)
"""


def test_script_dung_chuan_khong_bao_gi():
    assert lint_manim_script(VALID) == []


def test_chan_api_ngoai_whitelist():
    script = "from conceptflow import *\nr = Rectangle(width=2)\n"
    issues = blocking_issues(lint_manim_script(script))
    assert len(issues) == 1
    assert "Rectangle" in issues[0].message
    assert issues[0].line == 2


def test_chan_star_import_tu_manim():
    """Star-import che khuất chính các tên của conceptflow, nên whitelist mất
    hiệu lực — đây là lỗi chứ không phải cảnh báo như đường thoát hiểm khác."""
    issues = blocking_issues(lint_manim_script("from manim import *\n"))
    assert len(issues) == 1
    assert "che khuất" in issues[0].message


def test_import_dich_danh_tu_manim_chi_la_canh_bao():
    issues = lint_manim_script("from conceptflow import *\nfrom manim import Arrow\na = Arrow()\n")
    assert blocking_issues(issues) == []
    assert [i.severity for i in issues] == [WARNING]
    assert "API thô" in issues[0].message


def test_ten_do_script_tu_dinh_nghia_khong_bi_bao():
    script = (
        "from conceptflow import *\n"
        "def build_diagram():\n"
        "    return TitleCard('x')\n"
        "d = build_diagram()\n"
    )
    assert lint_manim_script(script) == []


def test_canh_bao_mau_hex_va_font_size_tho():
    script = "from conceptflow import *\nc = Callout('x', tone='#ff0000')\n"
    issues = lint_manim_script(script)
    assert blocking_issues(issues) == []
    assert any("hex" in i.message for i in issues)

    script2 = "from conceptflow import *\nt = TitleCard('x', font_size=37)\n"
    issues2 = lint_manim_script(script2)
    assert blocking_issues(issues2) == []
    assert any("font_size" in i.message for i in issues2)


def test_font_size_trong_thang_theme_khong_bi_bao():
    script = "from conceptflow import *\nt = TitleCard('x', font_size=28)\n"
    assert lint_manim_script(script) == []


def test_bao_loi_cu_phap():
    issues = lint_manim_script("def broken(:\n")
    assert len(issues) == 1
    assert issues[0].severity == BLOCKING
    assert "không phải Python hợp lệ" in issues[0].message
