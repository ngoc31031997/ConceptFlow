"""Danh sách tên công khai của `conceptflow`, dưới dạng dữ liệu thuần.

Tách khỏi `__init__.py` vì **lint phải đọc được danh sách này mà không import
manim**. `script_lint` chạy trong tiến trình của Rendering Service và trong test
suite — mà test suite của Rendering có một tính chất đáng giữ: nó không cần cài
manim (README nói rõ). Import `conceptflow/__init__.py` để lấy `__all__` sẽ kéo
theo cả manim và phá tính chất đó.

`tests/test_api_surface.py` khoá hai danh sách phải trùng nhau, nên không có
chuyện chúng trôi khỏi nhau trong im lặng.
"""

from __future__ import annotations

#: Nền của mọi video.
SCENE_NAMES = frozenset({"ConceptFlowScene"})

#: Component dựng cảnh (CR-017 FR45.2).
COMPONENT_NAMES = frozenset({
    "TitleCard",
    "Callout",
    "CodePanel",
    "StepList",
    "ComparisonSplit",
    "Recap",
})

#: Theme và hình học khung an toàn.
THEME_NAMES = frozenset({"Theme", "get", "register", "available"})
GEOMETRY_NAMES = frozenset({"Box", "safe_area", "overflow", "overlaps", "contrast_ratio"})

#: Vài thứ của Manim được tái xuất vì script thực sự cần: gom nhóm và chỉ hướng.
#:
#: Danh sách này cố tình ngắn. Mỗi tên thêm vào đây là một phần bề mặt Manim mở
#: lại cho script, và mở càng nhiều thì whitelist càng mất tác dụng.
REEXPORTED_MANIM_NAMES = frozenset({
    "VGroup",
    "UP",
    "DOWN",
    "LEFT",
    "RIGHT",
    "ORIGIN",
})

PUBLIC_NAMES: frozenset[str] = (
    SCENE_NAMES | COMPONENT_NAMES | THEME_NAMES | GEOMETRY_NAMES | REEXPORTED_MANIM_NAMES
)

#: Method dựng chữ/bố cục/chuyển cảnh trên ConceptFlowScene. Lint dùng để phân
#: biệt `self.body(...)` (hợp lệ) với `self.add_something_odd(...)`.
SCENE_METHODS = frozenset({
    "title", "heading", "body", "caption", "formula", "code",
    "stack", "row", "fit",
    "reveal", "dismiss", "swap", "emphasize", "clear_stage",
})
