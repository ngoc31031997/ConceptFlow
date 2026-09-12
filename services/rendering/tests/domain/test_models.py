import unicodedata

from domain.models import ScriptRenderRequest


def _request(script_content: str) -> ScriptRenderRequest:
    return ScriptRenderRequest(
        project_id="p1",
        script_content=script_content,
        scene_class_name="DemoScene",
        narration_segments=[],
    )


def test_script_content_duoc_chuan_hoa_ve_nfc():
    # macOS clipboard trả tiếng Việt dạng NFD (chữ cái gốc + dấu tách rời) —
    # Manim's Text mobject build một submobject/codepoint qua Pango, và Pango
    # có thể gộp chuỗi tách rời thành ít glyph hơn số codepoint, khiến
    # `_gen_chars` chết với IndexError. NFC gộp lại thành đúng một codepoint
    # mỗi chữ, khớp 1:1 với glyph Pango sinh ra.
    nfd = unicodedata.normalize("NFD", "Số lần lặp")
    assert nfd != "Số lần lặp"  # xác nhận input thật sự là NFD, không phải NFC sẵn

    request = _request(nfd)

    assert request.script_content == unicodedata.normalize("NFC", nfd)
    assert request.script_content == "Số lần lặp"


def test_script_content_da_la_nfc_thi_giu_nguyen():
    request = _request("from conceptflow import *\n")
    assert request.script_content == "from conceptflow import *\n"
