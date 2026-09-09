from domain.script_lint import lint_manim_script


def test_flags_corner_radius_on_rectangle():
    script = (
        "from manim import Rectangle\n"
        "r = Rectangle(width=3.2, height=1.8, color=WHITE, corner_radius=0.15)\n"
    )
    issues = lint_manim_script(script)
    assert len(issues) == 1
    assert issues[0].line == 2
    assert "RoundedRectangle" in issues[0].message


def test_allows_corner_radius_on_rounded_rectangle():
    script = "r = RoundedRectangle(width=3.2, height=1.8, corner_radius=0.15)\n"
    assert lint_manim_script(script) == []


def test_allows_rectangle_without_corner_radius():
    script = "r = Rectangle(width=3.2, height=1.8, color=WHITE)\n"
    assert lint_manim_script(script) == []


def test_flags_syntax_error():
    issues = lint_manim_script("def broken(:\n")
    assert len(issues) == 1
    assert "not valid Python" in issues[0].message
