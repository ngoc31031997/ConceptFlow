"""Sanity tests for domain value objects.

Sau CR-018 service này chỉ còn `ParsedScript`. `Scene` và `Chapter` đã rời khỏi
đây: lời thoại và chapter do lượt dry của Rendering sinh ra theo thứ tự chạy
thật, không còn đọc được từ text script.
"""

from __future__ import annotations

from domain.models import ParsedScript


def test_parsed_script_holds_scene_class_name():
    assert ParsedScript(scene_class_name="DemoScene").scene_class_name == "DemoScene"
