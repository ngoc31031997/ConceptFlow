"""Lỗi có cấu trúc khi kiểm code AI vẽ: loại lỗi, dòng và đoạn code quanh dòng."""

from __future__ import annotations

from domain.illustration import CodeFinding, DrawErrorKind, code_snippet, draw_check_error

CODE = "line one\nline two\nline three\nline four\n"


def test_snippet_marks_the_error_line_between_its_neighbours():
    assert code_snippet(CODE, 2) == "1:   line one\n2: > line two\n3:   line three"


def test_snippet_at_the_edges_keeps_only_existing_lines():
    assert code_snippet(CODE, 1) == "1: > line one\n2:   line two"
    assert code_snippet(CODE, 4) == "3:   line three\n4: > line four"


def test_no_snippet_without_a_line_or_outside_the_code():
    assert code_snippet(CODE, None) is None
    assert code_snippet(CODE, 0) is None and code_snippet(CODE, 9) is None
    assert code_snippet("", 1) is None


def test_kind_follows_the_renderer_stage_that_refused_the_code():
    syntax = draw_check_error(CODE, [CodeFinding("TS1005: ';' expected.", 3)])
    assert syntax.kind == DrawErrorKind.SYNTAX and syntax.line == 3
    assert syntax.message == "Code lỗi cú pháp — dòng 3: TS1005: ';' expected."
    typed = draw_check_error(CODE, [CodeFinding("TS2304: Cannot find name 'x'.", 2)])
    assert typed.kind == DrawErrorKind.RENDER
    kit = draw_check_error(CODE, [CodeFinding("[S1] màu lạ", 4), CodeFinding("[S7] thiếu nhóm", None)])
    assert kit.kind == DrawErrorKind.KIT and kit.line == 4
    assert kit.message == "Code vi phạm luật vẽ của kit — dòng 4: [S1] màu lạ; [S7] thiếu nhóm."
    static = draw_check_error(CODE, [CodeFinding("không dùng 'fetch' trong hình minh hoạ", 2)])
    assert static.kind == DrawErrorKind.RENDER and static.snippet == code_snippet(CODE, 2)


def test_the_line_is_the_first_finding_that_has_one():
    err = draw_check_error(CODE, [CodeFinding("[S7] thiếu nhóm", None), CodeFinding("[S1] màu lạ", 3)])
    assert err.line == 3 and err.to_json() == {
        "kind": "kit", "message": err.message, "line": 3, "snippet": code_snippet(CODE, 3)}
